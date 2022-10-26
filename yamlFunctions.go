package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"yamlParser/util"

	"golang.org/x/crypto/ssh"
)

type yamlFunction func(param ...string) (string, error)

type YamlFunctions struct {
	namedValues map[string]string
	functions   map[string]yamlFunction
}

func NewYamlFunctions() *YamlFunctions {
	y := &YamlFunctions{
		namedValues: make(map[string]string, 50),
		functions: map[string]yamlFunction{
			"generateRandomString": generateRandomString,
			"generateRandomInt":    generateRandomInt,
		},
	}

	y.functions["generateRandomNamedString"] = func(param ...string) (string, error) {
		if len(param) != 2 {
			return "", fmt.Errorf("invalid parameter amount, 2 expected %d provided", len(param))
		}
		str, err := generateRandomString(param[1])
		if err != nil {
			return "", err
		}

		y.namedValues[param[0]] = str
		return param[1], nil
	}

	y.functions["namedString"] = func(param ...string) (string, error) {
		if len(param) != 2 {
			return "", fmt.Errorf("invalid parameter amount, 2 expected %d provided", len(param))
		}
		y.namedValues[param[0]] = param[1]
		return param[1], nil
	}

	y.functions["getNamedString"] = func(param ...string) (string, error) {
		if len(param) != 1 {
			return "", fmt.Errorf("invalid parameter amount, 1 expected %d provided", len(param))
		}
		val, found := y.namedValues[param[0]]
		if !found {
			return "", fmt.Errorf("no stored value for key [%s]", param[0])
		}
		return val, nil
	}

	y.functions["generateEd25519Key"] = func(param ...string) (string, error) {
		if len(param) != 1 {
			return "", fmt.Errorf("invalid parameter amount, 1 expected %d provided", len(param))
		}

		pubKey, privKey, _ := ed25519.GenerateKey(rand.Reader)
		publicKey, _ := ssh.NewPublicKey(pubKey)

		pemKey := &pem.Block{
			Type:  "OPENSSH PRIVATE KEY",
			Bytes: util.MarshalED25519PrivateKey(privKey), // <- marshals ed25519 correctly
		}
		privateKey := pem.EncodeToMemory(pemKey)
		authorizedKey := ssh.MarshalAuthorizedKey(publicKey)

		name := param[0]
		y.namedValues[name+"Public"] = strings.TrimSpace(string(authorizedKey))
		y.namedValues[name+"Private"] = strings.TrimSpace(string(privateKey))

		return y.namedValues[name+"Public"], nil
	}

	return y
}

func (f YamlFunctions) Call(name string, params ...string) (string, error) {
	fn, found := f.functions[name]
	if !found {
		return "", fmt.Errorf("function [%s] not found", name)
	}
	return fn(params...)
}

func generateRandomInt(param ...string) (string, error) {
	if len(param) != 2 {
		return "", fmt.Errorf("invalid parameter amount, 2 expected %d provided", len(param))
	}
	min, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return "", err
	}
	max, err := strconv.ParseInt(param[1], 10, 64)
	if err != nil {
		return "", err
	}
	if max <= min {
		return "", fmt.Errorf("max [%d] must be bigger than min [%d]", max, min)
	}

	n, err := rand.Int(rand.Reader, big.NewInt(max-min+1))
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(n.Int64()+min, 10), nil
}

func generateRandomString(param ...string) (string, error) {
	const maxRandStringLen = 1024
	if len(param) != 1 {
		return "", fmt.Errorf("invalid parameter amount, 1 expected %d provided", len(param))
	}
	length, err := strconv.ParseInt(param[0], 10, 64)
	if err != nil {
		return "", err
	}
	if length > maxRandStringLen {
		return "", fmt.Errorf("provided length %d exceeds maximum length of %d characters", length, maxRandStringLen)
	}

	result := make([]byte, int(math.Ceil(float64(length)/2)))
	if _, err := rand.Read(result); err != nil {
		return "", err
	}
	return hex.EncodeToString(result)[:length], nil
}
