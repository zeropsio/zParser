package modifiers

import (
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/zeropsio/zParser/v2/src/util"
)

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
				hash, err := bcrypt.GenerateFromPassword([]byte(in), 11) // cost set to not overload the parser service
				return string(hash), err
			},
			"argon2id": func(in string) (string, error) {
				return util.Argon2IDPasswordHash(in, util.DefaultArgon2idConf())
			},
			"toHex": func(in string) (string, error) {
				return hex.EncodeToString([]byte(in)), nil
			},
			"toString": func(in string) (string, error) {
				return util.BytesToString([]byte(in)), nil
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
