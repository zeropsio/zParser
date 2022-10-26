package main

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type modifyFunc func(in string) (string, error)

type YamlModifiers struct {
	modifiers map[string]modifyFunc
}

func NewYamlModifiers() *YamlModifiers {
	caser := cases.Title(language.English, cases.NoLower)
	return &YamlModifiers{
		modifiers: map[string]modifyFunc{
			"sha512": func(in string) (string, error) {
				hash := sha512.New()
				hash.Write([]byte(in))
				return hex.EncodeToString(hash.Sum(nil)), nil
			},
			"bcrypt": func(in string) (string, error) {
				// TODO(ms): maybe use cost 10 to avoid DoS attacks, or rather limit max amount of usages?
				hash, err := bcrypt.GenerateFromPassword([]byte(in), 11)
				return string(hash), err
			},
			"upper": func(in string) (string, error) {
				return strings.ToUpper(in), nil
			},
			"title": func(in string) (string, error) {
				// TODO(ms): use casers from shared
				return caser.String(in), nil
			},
			"lower": func(in string) (string, error) {
				return strings.ToLower(in), nil
			},
		},
	}
}

func (f YamlModifiers) Call(name, value string) (string, error) {
	fn, found := f.modifiers[name]
	if !found {
		return "", fmt.Errorf("modifier [%s] not found", name)
	}
	return fn(value)
}

func (f YamlModifiers) CallBatch(value string, modifiers ...string) (string, error) {
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
