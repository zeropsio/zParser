package main

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type modifyFunc func(in string) string

type YamlModifiers struct {
	modifiers map[string]modifyFunc
}

func NewYamlModifiers() *YamlModifiers {
	caser := cases.Title(language.English, cases.NoLower)
	return &YamlModifiers{
		modifiers: map[string]modifyFunc{
			"sha512": func(in string) string {
				hash := sha512.New()
				hash.Write([]byte(in))
				return hex.EncodeToString(hash.Sum(nil))
			},
			"upper": func(in string) string {
				return strings.ToUpper(in)
			},
			"title": func(in string) string {
				// TODO(ms): use casers from shared
				return caser.String(in)
			},
			"lower": func(in string) string {
				return strings.ToLower(in)
			},
		},
	}
}

func (f YamlModifiers) Call(name, value string) (string, error) {
	fn, found := f.modifiers[name]
	if !found {
		return "", fmt.Errorf("modifier [%s] not found", name)
	}
	return fn(value), nil
}

func (f YamlModifiers) CallBatch(value string, modifiers ...string) (string, error) {
	for _, name := range modifiers {
		fn, found := f.modifiers[name]
		if !found {
			return "", fmt.Errorf("modifier [%s] not found", name)
		}
		value = fn(value)
	}
	return value, nil
}
