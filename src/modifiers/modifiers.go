package modifiers

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// TODO(ms): test and choose better bcrypt and argon parameters
//  - one password hash should take about 300ms with bcrypt
//  - argon2id hashing may be faster than bcrypt (no need for 300ms), but must use more memory

type modifyFunc func(in string) (string, error)

type Modifiers struct {
	modifiers map[string]modifyFunc
}

func NewModifiers() *Modifiers {
	titleCaser := cases.Title(language.English, cases.NoLower)
	return &Modifiers{
		modifiers: map[string]modifyFunc{
			"sha256": func(in string) (string, error) {
				hash := sha256.New()
				hash.Write([]byte(in))
				return hex.EncodeToString(hash.Sum(nil)), nil
			},
			"sha512": func(in string) (string, error) {
				hash := sha512.New()
				hash.Write([]byte(in))
				return hex.EncodeToString(hash.Sum(nil)), nil
			},
			"bcrypt": func(in string) (string, error) {
				hash, err := bcrypt.GenerateFromPassword([]byte(in), 10) // cost 10 to not overload the parser service
				return string(hash), err
			},
			"argon2id": func(in string) (string, error) {
				// standard sane parameters chosen to not overload the parser service
				const (
					saltLen     = 16        // bytes
					memory      = 96 * 1024 // kilobytes - main "knob" to turn for more expensive hashes
					iterations  = 1
					parallelism = 3
					keyLength   = 32
				)
				salt := make([]byte, saltLen)
				if _, err := rand.Read(salt); err != nil {
					return "", err
				}

				hash := argon2.IDKey([]byte(in), salt, iterations, memory, parallelism, keyLength)

				// Base64 encode the salt and hashed password.
				b64Salt := base64.RawStdEncoding.EncodeToString(salt)
				b64Hash := base64.RawStdEncoding.EncodeToString(hash)

				// Return a string using the standard encoded hash representation.
				return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, memory, iterations, parallelism, b64Salt, b64Hash), nil
			},
			"upper": func(in string) (string, error) {
				return strings.ToUpper(in), nil
			},
			"title": func(in string) (string, error) {
				return titleCaser.String(in), nil
			},
			"lower": func(in string) (string, error) {
				return strings.ToLower(in), nil
			},
			"noop": func(in string) (string, error) {
				return in, nil
			},
		},
	}
}

func (f Modifiers) Call(name, value string) (string, error) {
	fn, found := f.modifiers[name]
	if !found {
		return "", fmt.Errorf("modifier [%s] not found", name)
	}
	return fn(value)
}

func (f Modifiers) CallBatch(value string, modifiers ...string) (string, error) {
	for _, name := range modifiers {
		fn, found := f.modifiers[name]
		if !found {
			return "", fmt.Errorf("modifier [%s] not found", name)
		}

		var err error
		value, err = fn(value)
		if err != nil {
			return "", err
		}
	}
	return value, nil
}
