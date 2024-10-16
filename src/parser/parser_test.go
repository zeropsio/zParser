package parser

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"

	"github.com/zeropsio/zParser/v2/src/metaError"
	"github.com/zeropsio/zParser/v2/src/util"
)

//goland:noinspection GoErrorStringFormat
func TestImportParser_Parse(t *testing.T) {
	type fields struct {
		out                     *bytes.Buffer
		in                      *bytes.Reader
		maxFunctionCount        int
		multiLineOutputHandling MultiLineOutputHandling
	}

	// comparison helper functions
	wantStaticLen := func(length int) func(string) error {
		return func(s string) error {
			if len(s) != length {
				return fmt.Errorf("wantLength = %d, got = %v, length = %d", length, s, len(s))
			}
			return nil
		}
	}
	wantStaticString := func(want string) func(string) error {
		return func(s string) error {
			if s != want {
				return fmt.Errorf("want = %v, got = %v", want, s)
			}
			return nil
		}
	}

	// helper functions
	getFields := func(buffSize int, maxFuncCount int, outputHandling MultiLineOutputHandling, input string) fields {
		return fields{
			out:                     bytes.NewBuffer(make([]byte, 0, buffSize)),
			in:                      bytes.NewReader([]byte(input)),
			maxFunctionCount:        maxFuncCount,
			multiLineOutputHandling: outputHandling,
		}
	}

	bgCtx := context.Background()
	tests := []struct {
		name        string
		fields      fields
		wantErr     bool // whether we want an error or not (if wantMetaErr is set to true, this is considered to be true as well)
		wantMetaErr bool // whether we want the received error to be a validation error (if set to true, returned error MUST be a meta error)
		want        func(string) error
		ctxTimeout  time.Duration
	}{
		// Generic
		{
			name:       "context cancel",
			fields:     getFields(1024, 200, MultilinePreserved, `<@generateRandomString(<50>)|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt|bcrypt>`),
			wantErr:    true,
			ctxTimeout: time.Millisecond * 500,
		},
		{
			name:        "max function count",
			fields:      getFields(1024, 1, MultilinePreserved, `<@generateRandomString(<50>) | sha256 | sha256 | sha256>`),
			wantMetaErr: true,
		},
		// Escaping
		{
			name:   "env variable",
			fields: getFields(1024, 0, MultilinePreserved, `${some_env_variable}`),
			want:   wantStaticString(`${some_env_variable}`),
		},
		{
			name:   "escaping simple",
			fields: getFields(1024, 1, MultilinePreserved, `\< \\ \\\\ \\<sTrInG| lower >\\ \\\\ \\ \>`),
			want:   wantStaticString(`< \ \\ \string\ \\ \ >`),
		},
		{
			name:   "escaping in function param",
			fields: getFields(1024, 1, MultilinePreserved, `<@setVar(<commaString>, <this is a named string, that contains some commas, and closing braces ) and backslashes \\ what do you think?>)>`),
			want:   wantStaticString(`this is a named string, that contains some commas, and closing braces ) and backslashes \ what do you think?`),
		},
		{
			name:   "escaping with supported characters",
			fields: getFields(1024, 0, MultilinePreserved, `0123456789 abcdefghijklmnopqrstuvwxyz ľščťžýáíéúäôň §~!@#$%^&*()_+}{|"':?\>\<°ˇ-=[];'\\,./`),
			want:   wantStaticString(`0123456789 abcdefghijklmnopqrstuvwxyz ľščťžýáíéúäôň §~!@#$%^&*()_+}{|"':?><°ˇ-=[];'\,./`),
		},
		// Nesting
		{
			name:   "nesting functions",
			fields: getFields(1024, 3, MultilinePreserved, `<@generateRandomInt(<@generateRandomInt(<-9>, <0>)>, <@generateRandomInt(<1>, <9>)>)>`),
			want: func(s string) error {
				num, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					return err
				}
				if num < -9 || num > 9 {
					return fmt.Errorf("expected number in [-9, 9] range, received %d", num)
				}
				return nil
			},
		},
		{
			name:   "nesting functions with modifier",
			fields: getFields(1024, 3, MultilinePreserved, `<@generateRandomString(<@generateRandomInt(<10>, <50>)>) | upper>`),
			want: func(s string) error {
				l := len(s)
				if l < 10 || l > 50 {
					return fmt.Errorf("expected random string with length [10, 50] range, received [%d]: %s", l, s)
				}
				if strings.ToUpper(s) != s {
					return fmt.Errorf("expected random string in upper case, received: %s", s)
				}
				return nil
			},
		},
		{
			name:   "nesting with spaces",
			fields: getFields(1024, 1, MultilinePreserved, `<this is < a nested string | noop> with double spaces>`),
			want:   wantStaticString(`this is  a nested string  with double spaces`),
		},
		{
			name:   "nesting without spaces",
			fields: getFields(1024, 1, MultilinePreserved, `<this is <a nested string| noop> with single spaces>`),
			want:   wantStaticString(`this is a nested string with single spaces`),
		},
		{
			name:   "nesting functions and strings with modifiers",
			fields: getFields(1024, 3, MultilinePreserved, `<@setVar(<name>, <this is <a nested string| title> with a modifier>)>`),
			want:   wantStaticString(`this is A Nested String with a modifier`),
		},
		{
			name:   "allowing env inside function param",
			fields: getFields(1024, 1, MultilinePreserved, `<@setVar(<name>, <hello ${user_name} how are you>)>`),
			want:   wantStaticString(`hello ${user_name} how are you`),
		},
		{
			name:   "env with random suffix",
			fields: getFields(1024, 1, MultilinePreserved, `${user_name<@generateRandomInt(<10>, <99>)>}`),
			want:   wantStaticLen(14),
		},
		// Functions
		{
			name:   "generate random string",
			fields: getFields(1024, 1, MultilinePreserved, `<@generateRandomString(<50>)>`),
			want:   wantStaticLen(50),
		},
		{
			name:   "generate random int",
			fields: getFields(1024, 1, MultilinePreserved, `<@generateRandomInt(<10>, <99>)>`),
			want: func(s string) error {
				num, err := strconv.ParseInt(s, 10, 64)
				if err != nil {
					return err
				}
				if num < 10 || num > 99 {
					return fmt.Errorf("expected number in [10, 99] range, received %d", num)
				}
				return nil
			},
		},
		{
			name:        "date time invalid timezone",
			fields:      getFields(1024, 1, MultilinePreserved, `<@getDatetime(<DD.MM.YYYY HH:mm:ss>, <Totally/Invalid/Zone>)>`),
			wantMetaErr: true,
		},
		{
			name:   "date time UTC",
			fields: getFields(1024, 1, MultilinePreserved, `<@getDatetime(<DD.MM.YYYY HH:mm:ss>)>`),
			want: func(s string) error {
				const layout = "02.01.2006 15:04:05"
				t, err := time.ParseInLocation(layout, s, time.UTC)
				if err != nil {
					return err
				}
				if t.Format(layout) != s {
					return fmt.Errorf("received date time string [%s] does not match parsed output [%s]", s, t.Format(layout))
				}
				n := time.Now().UTC()
				if !t.After(n.Add(-time.Second)) || !n.Add(-5*time.Second).Before(t) {
					return fmt.Errorf("received date time [%s] is not between `now-5s` and `now` [%s]", t.Format(layout), n.Format(layout))
				}

				return nil
			},
		},
		{
			name:   "date time Europe/Prague",
			fields: getFields(1024, 1, MultilinePreserved, `<@getDatetime(<DD.MM.YYYY HH:mm:ss>, <Europe/Prague>)>`),
			want: func(s string) error {
				const layout = "02.01.2006 15:04:05"
				loc, err := time.LoadLocation("Europe/Prague")
				if err != nil {
					return fmt.Errorf("failed to load location for `Europe/Prague`: %w", err)
				}
				t, err := time.ParseInLocation(layout, s, loc)
				if err != nil {
					return err
				}
				if t.Format(layout) != s {
					return fmt.Errorf("received date time string [%s] does not match parsed output [%s]", s, t.Format(layout))
				}
				n := time.Now().In(loc)
				if !t.After(n.Add(-time.Second)) || !n.Add(-5*time.Second).Before(t) {
					return fmt.Errorf("received date time [%s] is not between `now-5s` and `now` [%s]", t.Format(layout), n.Format(layout))
				}

				return nil
			},
		},
		{
			name:   "mercury in retrograde",
			fields: getFields(1024, 1, MultilinePreserved, `<@mercuryInRetrograde(<Mercury is in retrograde>, <Mercury is not in retrograde>)>`),
			want: func(s string) error {
				yes, err := util.MercuryInRetrograde()
				if err != nil {
					// failing tests in 2031 should prompt update of the map ;-)
					return fmt.Errorf("MercuryInRetrograde returned an error: %w", err)
				}
				if yes && s == "Mercury is not in retrograde" {
					return errors.New("mercury should be in retrograde, but apparently it isn't")
				}
				if !yes && s == "Mercury is in retrograde" {
					return errors.New("mercury should not be in retrograde, but apparently it is")
				}
				return nil
			},
		},
		{
			name:   "generate random string var",
			fields: getFields(1024, 1, MultilinePreserved, `<@generateRandomStringVar(<name>, <50>)>`),
			want:   wantStaticLen(50),
		},
		{
			name:   "get random string var",
			fields: getFields(1024, 2, MultilinePreserved, `<@generateRandomStringVar(<name>, <50>)>|<@getVar(name)>`),
			want:   wantStaticLen(101),
		},
		{
			name:   "custom var",
			fields: getFields(1024, 1, MultilinePreserved, `<@setVar(<name>, <my completely custom string>)>`),
			want:   wantStaticString(`my completely custom string`),
		},
		{
			name:   "get custom var",
			fields: getFields(1024, 2, MultilinePreserved, `<@setVar(<name>, <my completely custom string>)>|<@getVar(name)>`),
			want:   wantStaticString(`my completely custom string|my completely custom string`),
		},
		{
			name:   "generate JWT",
			fields: getFields(1024, 1, MultilinePreserved, `<@generateJWT(<jwtTokenSecretKey>, <{"role":"test","exp":1799535600}>)>`),
			want: func(s string) error {
				now := time.Now()

				token, err := jwt.Parse(s, func(token *jwt.Token) (interface{}, error) {
					return []byte("jwtTokenSecretKey"), nil
				})
				if err != nil {
					return err
				}

				iss, err := token.Claims.GetIssuer()
				if err != nil {
					return err
				}
				if iss != "zerops" {
					return fmt.Errorf("expected issuer to be zerops, received: %s", iss)
				}

				iat, err := token.Claims.GetIssuedAt()
				if err != nil {
					return err
				}
				if now.Sub(iat.Time).Seconds() > 1 {
					return fmt.Errorf("expected issued date to be within 1 second of now, %s vs %s", now.String(), iat.Time.String())
				}

				exp, err := token.Claims.GetExpirationTime()
				if err != nil {
					return err
				}
				if exp.Unix() != 1799535600 {
					return fmt.Errorf("expected expiry to be 1799535600, received: %v", exp.Unix())
				}

				claims, _ := token.Claims.(jwt.MapClaims)
				if claims["role"] != "test" {
					return fmt.Errorf("expected role to be test, received: %v", claims["role"])
				}

				return nil
			},
		},
		{
			name:   "generate JWT with different issuer",
			fields: getFields(1024, 1, MultilinePreserved, `<@generateJWT(<jwtTokenSecretKey>, <{"role":"test","iss":"test","exp":1799535600}>)>`),
			want: func(s string) error {
				token, err := jwt.Parse(s, func(token *jwt.Token) (interface{}, error) {
					return []byte("jwtTokenSecretKey"), nil
				})
				if err != nil {
					return err
				}

				iss, err := token.Claims.GetIssuer()
				if err != nil {
					return err
				}
				if iss != "test" {
					return fmt.Errorf("expected issuer to be test, received: %s", iss)
				}
				return nil
			},
		},
		{
			name:   "multi line output preserve",
			fields: getFields(1024, 1, MultilinePreserved, "\t\t<@generateED25519Key(<key>)>"),
			want: func(s string) error {
				parts := strings.Split(s, "\n")
				if len(parts) <= 1 {
					return fmt.Errorf("expected 3 lines, found %d, got = %v", len(parts), s)
				}
				for n, line := range parts {
					if n == 0 && !strings.HasPrefix(line, "\t\t") {
						return fmt.Errorf("expected first line to be prefixed with `\\t\\t`, got = %v", line)
					}
					if n > 0 && strings.HasPrefix(line, "\t\t") {
						return fmt.Errorf("expected other lines to NOT be prefixed with `\\t\\t`, got = %v", line)
					}
				}
				return nil
			},
		},
		{
			name:   "multi line output preserve with indent",
			fields: getFields(1024, 1, MultilineWithIndent, "\t\t<@generateED25519Key(<key>)>"),
			want: func(s string) error {
				parts := strings.Split(s, "\n")
				if len(parts) <= 1 {
					return fmt.Errorf("expected 3 lines, found %d, got = %v", len(parts), s)
				}
				for _, line := range parts {
					if !strings.HasPrefix(line, "\t\t") {
						return fmt.Errorf("expected every line to be prefixed with `\\t\\t`, got = %v", line)
					}
				}
				return nil
			},
		},
		{
			name:   "multi line output preserve with indent - nested",
			fields: getFields(1024, 1, MultilineWithIndent, "\t\t<Text\n\t\t<@generateED25519Key(<key>)>\n\t\tText>"),
			want: func(s string) error {
				parts := strings.Split(s, "\n")
				if len(parts) != 5 {
					return fmt.Errorf("expected 5 lines, found %d, got = %v", len(parts), s)
				}
				for _, line := range parts {
					if !strings.HasPrefix(line, "\t\t") {
						return fmt.Errorf("expected every line to be prefixed with `\\t\\t`, got = %v", line)
					}
				}
				return nil
			},
		},
		{
			name:   "multi line output squashed",
			fields: getFields(1024, 1, MultilineToOneLine, "\t\t<@generateED25519Key(<key>)>"),
			want: func(s string) error {
				if newLines := strings.Count(s, "\n"); newLines > 0 {
					return fmt.Errorf("expected 1 line, found %d, got = %v", newLines, s)
				}
				if newLines := strings.Count(s, "\\n"); newLines != 2 {
					return fmt.Errorf("expected 3 lines squashed to one, found %d, got = %v", newLines, s)
				}
				return nil
			},
		},
		{
			// tests complete generation of public, public ssh, private and private ssh keys
			name:   "get ED25519 private key",
			fields: getFields(1024, 4, MultilinePreserved, "<@generateED25519Key(<key>)>|<@getVar(keyPrivate)>|<@getVar(keyPrivateSsh)>|<@getVar(keyPublicSsh)>"),
			want: func(s string) error {
				parts := strings.Split(s, "|")
				if len(parts) != 4 {
					return fmt.Errorf("expected 4 parts, found %d, got = %v", len(parts), s)
				}

				privatePem, _ := pem.Decode([]byte(parts[1]))
				if privatePem == nil || privatePem.Type != "PRIVATE KEY" {
					return fmt.Errorf("failed to decode PEM block containing private key: %+v", privatePem)
				}
				privateKeyAny, err := x509.ParsePKCS8PrivateKey(privatePem.Bytes)
				if err != nil {
					return fmt.Errorf("failed to ParsePKCS8PrivateKey: %w\n%+v", err, privatePem)
				}
				privateKey, ok := privateKeyAny.(ed25519.PrivateKey)
				if !ok {
					return fmt.Errorf("failed to type cast private key to ed25519.PrivateKey")
				}

				publicPem, _ := pem.Decode([]byte(parts[0]))
				if publicPem == nil || publicPem.Type != "PUBLIC KEY" {
					return fmt.Errorf("failed to decode PEM block containing public key: %+v", publicPem)
				}
				publicKeyAny, err := x509.ParsePKIXPublicKey(publicPem.Bytes)
				if err != nil {
					return fmt.Errorf("failed to ParsePKIXPublicKey: %w\n%+v", err, publicPem)
				}
				pubLicKey, ok := publicKeyAny.(ed25519.PublicKey)
				if !ok {
					return fmt.Errorf("failed to type cast public key to ed25519.PublicKey")
				}

				if !pubLicKey.Equal(privateKey.Public()) {
					return fmt.Errorf("provided privateKey does not match provided publicKey: %v", s)
				}

				// TODO(ms): verify if PublicSsh <=> PrivateSsh

				return nil
			},
		},
		{
			// tests complete generation of public, public ssh and private keys
			name:   "generate RSA2048 key",
			fields: getFields(1024, 4, MultilinePreserved, "<@generateRSA2048Key(<key>)>|<@getVar(keyPrivate)>|<@getVar(keyPublicSsh)>"),
			want: func(s string) error {
				parts := strings.Split(s, "|")
				if len(parts) != 3 {
					return fmt.Errorf("expected 3 parts, found %d, got = %v", len(parts), s)
				}

				return validateRsaKey(parts[0], parts[1], parts[2])
			},
		},
		{
			// tests complete generation of public, public ssh and private keys
			name:   "generate RSA4096 key",
			fields: getFields(1024, 4, MultilinePreserved, "<@generateRSA4096Key(<key>)>|<@getVar(keyPrivate)>|<@getVar(keyPublicSsh)>"),
			want: func(s string) error {
				parts := strings.Split(s, "|")
				if len(parts) != 3 {
					return fmt.Errorf("expected 3 parts, found %d, got = %v", len(parts), s)
				}

				return validateRsaKey(parts[0], parts[1], parts[2])
			},
		},
		// Modifiers
		{
			name:   "modifier title",
			fields: getFields(1024, 1, MultilinePreserved, `<my string in title case| title>`),
			want:   wantStaticString(`My String In Title Case`),
		},
		{
			name:   "modifier upper",
			fields: getFields(1024, 1, MultilinePreserved, `<mY StriNg iN UppER caSe| upper>`),
			want:   wantStaticString(`MY STRING IN UPPER CASE`),
		},
		{
			name:   "modifier lower",
			fields: getFields(1024, 1, MultilinePreserved, `<My sTRing In lOWer cAsE| lower>`),
			want:   wantStaticString(`my string in lower case`),
		},
		{
			name:   "modifier noop",
			fields: getFields(1024, 1, MultilinePreserved, `<My sTRing wIthoUt { any } ChangEs !@!| noop>`),
			want:   wantStaticString(`My sTRing wIthoUt { any } ChangEs !@!`),
		},
		{
			name:   "modifier sha256",
			fields: getFields(1024, 2, MultilinePreserved, `<this string should be hashed using sha256 algorithm| sha256>`),
			want:   wantStaticString(`28aa52395ab73ec770e95ebe006d6e560e15effb227f2c3ebf743259ebd62bb8`),
		},
		{
			name:   "modifier sha512",
			fields: getFields(1024, 2, MultilinePreserved, `<this string should be hashed using sha512 algorithm| sha512>`),
			want:   wantStaticString(`3ff0c00ebf7d9b69efefcb38ccf98ee46927e16e01200dcc8bc9071dbe8089360d779206928447df5a3004e66cbc118b3d7e731dd15bfde7ccbac9530678ec99`),
		},
		{
			name:   "modifier bcrypt",
			fields: getFields(1024, 2, MultilinePreserved, `<this string should be hashed using bcrypt| bcrypt>`),
			want: func(s string) error {
				if err := bcrypt.CompareHashAndPassword([]byte(s), []byte("this string should be hashed using bcrypt")); err != nil {
					return fmt.Errorf("received bcrypt hash is not the hash of the given string, got = %v", s)
				}
				return nil
			},
		},
		{
			name:   "modifier argon2id",
			fields: getFields(1024, 2, MultilinePreserved, `<this string should be hashed using argon2id| argon2id>`),
			want: func(s string) error {
				if err := util.Argon2IDPasswordVerify(s, "this string should be hashed using argon2id"); err != nil {
					return fmt.Errorf("received argon2id hash is not the hash of the given string, got = %v", s)
				}
				return nil
			},
		},
		{
			name:   "modifiers title and sha256",
			fields: getFields(1024, 2, MultilinePreserved, `<my string in title case| title | sha256>`),
			want:   wantStaticString(`bb8973c3a99ec24dff29210d336fbdce5568b853acd3c0ca68f3cc9e6fb86659`),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := bgCtx
			if tt.ctxTimeout.Nanoseconds() > 0 {
				var ctxCancel context.CancelFunc
				ctx, ctxCancel = context.WithTimeout(ctx, tt.ctxTimeout)
				defer ctxCancel()
			}

			p := NewParser(tt.fields.in, tt.fields.out, WithMaxFunctionCount(tt.fields.maxFunctionCount), WithMultilineOutputHandling(tt.fields.multiLineOutputHandling))
			err := p.Parse(ctx)

			if err == nil && (tt.wantErr || tt.wantMetaErr) {
				t.Errorf("Parse() error = %v, wantErr %v, wantMetaErr %v", err, tt.wantErr, tt.wantMetaErr)
			}
			if err != nil && !(tt.wantErr || tt.wantMetaErr) {
				t.Errorf("Parse() error = %v, wantErr %v, wantMetaErr %v", err, tt.wantErr, tt.wantMetaErr)
			}
			if err != nil {
				metaErr := new(metaError.MetaError)
				if errors.As(err, &metaErr) && !tt.wantMetaErr {
					t.Errorf("Parse() meta error = %v, wantMetaErr %v\n%s", err, tt.wantMetaErr, metaErr.GetMetaAsString())
				}
				return
			}

			if tt.want != nil {
				out := tt.fields.out.String()
				if err := tt.want(out); err != nil {
					t.Errorf("Parser() %s", err)
				}
				return
			}
		})
	}
}

func validateRsaKey(public, private, publicSsh string) error {
	pubBlock, _ := pem.Decode([]byte(public))
	if pubBlock == nil || pubBlock.Type != "PUBLIC KEY" {
		return fmt.Errorf("failed to decode PEM block containing public key: %+v \n%s", pubBlock, public)
	}
	pubKey, err := x509.ParsePKIXPublicKey(pubBlock.Bytes)
	if err != nil {
		return err
	}

	privBlock, _ := pem.Decode([]byte(private))
	if privBlock == nil || privBlock.Type != "PRIVATE KEY" {
		return fmt.Errorf("failed to decode PEM block containing private key: %+v \n%s", privBlock, private)
	}
	privKey, err := x509.ParsePKCS8PrivateKey(privBlock.Bytes)
	if err != nil {
		return err
	}

	if !(privKey.(*rsa.PrivateKey)).PublicKey.Equal(pubKey) {
		return fmt.Errorf("provided privateKey does not match provided publicKey: %v\n%v", public, private)
	}

	// TODO(ms): verify if public ssh key is valid for the private key
	// pubSshBlock, _, _, _, err := ssh.ParseAuthorizedKey([]byte(publicSsh))
	// if err != nil {
	// 	return err
	// }
	// pubSshKey, err := x509.ParsePKCS1PublicKey(pubSshBlock.Marshal())
	// if err != nil {
	// 	return err
	// }
	// if !(pubKey.(*rsa.PublicKey).Equal(pubSshKey)) {
	// 	return fmt.Errorf("provided publicKey does not match provided publicKeySsh: %v\n%v", public, publicSsh)
	// }
	// if !(privKey.(*rsa.PrivateKey).PublicKey.Equal(pubSshKey)) {
	// 	return fmt.Errorf("provided publicKey does not match provided publicKeySsh: %v\n%v", public, publicSsh)
	// }

	return nil
}
