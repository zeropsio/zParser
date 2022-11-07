package functions

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	rand2 "math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bykof/gostradamus"
	"golang.org/x/crypto/ssh"

	"git.vsh-labs.cz/zerops/yaml-parser/src/util"
)

const maxRandStringLen = 1024

type function func(param ...string) (string, error)

type Functions struct {
	namedValues map[string]string
	functions   map[string]function
}

func NewFunctions() *Functions {
	y := &Functions{
		namedValues: make(map[string]string, 50),
		functions: map[string]function{
			"generateRandomString": generateRandomString,
			"generateRandomInt":    generateRandomInt,
			"pickRandom":           pickRandom,
			"mercuryInRetrograde":  mercuryInRetrograde,
			"getDatetime":          getDatetime,
		},
	}

	y.functions["generateRandomNamedString"] = func(param ...string) (string, error) {
		if err := paramCountCheck(2, len(param)); err != nil {
			return "", err
		}
		str, err := generateRandomString(param[1])
		if err != nil {
			return "", err
		}

		y.namedValues[param[0]] = str
		return str, nil
	}

	y.functions["namedString"] = func(param ...string) (string, error) {
		if err := paramCountCheck(2, len(param)); err != nil {
			return "", err
		}

		y.namedValues[param[0]] = param[1]
		return param[1], nil
	}

	y.functions["getNamedString"] = func(param ...string) (string, error) {
		if err := paramCountCheck(1, len(param)); err != nil {
			return "", err
		}
		val, found := y.namedValues[param[0]]
		if !found {
			return "", fmt.Errorf("no stored value for key [%s]", param[0])
		}
		return val, nil
	}

	y.functions["generateED25519Key"] = func(param ...string) (string, error) {
		if err := paramCountCheck(1, len(param)); err != nil {
			return "", err
		}

		publicKey, privateKey, _ := ed25519.GenerateKey(rand.Reader)

		privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return "", err
		}
		publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
		if err != nil {
			return "", err
		}

		privatePem := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyBytes,
		}
		privateOpenSshPem := &pem.Block{
			Type:  "OPENSSH PRIVATE KEY",
			Bytes: util.MarshalED25519PrivateKey(privateKey), // <- marshals ed25519 correctly
		}
		publicPem := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		}
		publicSshKey, _ := ssh.NewPublicKey(publicKey)

		name := param[0]
		y.namedValues[name+"Public"] = strings.TrimSpace(string(pem.EncodeToMemory(publicPem)))
		y.namedValues[name+"Private"] = strings.TrimSpace(string(pem.EncodeToMemory(privatePem)))
		y.namedValues[name+"PublicSsh"] = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicSshKey)))
		y.namedValues[name+"PrivateSsh"] = strings.TrimSpace(string(pem.EncodeToMemory(privateOpenSshPem)))

		return y.namedValues[name+"Public"], nil
	}

	y.functions["generateRSA4096Key"] = func(param ...string) (string, error) {
		if err := paramCountCheck(1, len(param)); err != nil {
			return "", err
		}

		privateKey, err := rsa.GenerateKey(rand.Reader, 4096)
		if err != nil {
			return "", err
		}

		privateKeyBytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
		if err != nil {
			return "", err
		}
		publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
		if err != nil {
			return "", err
		}

		privatePem := &pem.Block{
			Type:  "PRIVATE KEY",
			Bytes: privateKeyBytes,
		}
		publicPem := &pem.Block{
			Type:  "PUBLIC KEY",
			Bytes: publicKeyBytes,
		}
		publicSshKey, _ := ssh.NewPublicKey(privateKey.Public())

		name := param[0]
		y.namedValues[name+"Public"] = strings.TrimSpace(string(pem.EncodeToMemory(publicPem)))
		y.namedValues[name+"Private"] = strings.TrimSpace(string(pem.EncodeToMemory(privatePem)))
		y.namedValues[name+"PublicSsh"] = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicSshKey)))

		return y.namedValues[name+"Public"], nil
	}

	return y
}

func (f Functions) Call(name string, params ...string) (string, error) {
	fn, found := f.functions[name]
	if !found {
		return "", fmt.Errorf("function [%s] not found", name)
	}
	return fn(params...)
}

func paramCountCheck(expected, received int) error {
	if expected != received {
		return fmt.Errorf("invalid parameter count, %d expected %d provided", expected, received)
	}
	return nil
}

// generates cryptographically secure random int in [min, max]
func generateRandomInt(param ...string) (string, error) {
	if err := paramCountCheck(2, len(param)); err != nil {
		return "", err
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

// generates cryptographically secure random string of specified length given its <= maxRandStringLen
func generateRandomString(param ...string) (string, error) {
	if err := paramCountCheck(1, len(param)); err != nil {
		return "", err
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

// selects one random value from all provided parameters
func pickRandom(param ...string) (string, error) {
	if len(param) == 0 {
		return "", fmt.Errorf("invalid parameter count, at least 1 expected %d provided", len(param))
	}
	rand2.Seed(time.Now().UnixNano())
	return param[rand2.Intn(len(param))], nil
}

// returns date time using formatted by format inside first parameter which supports gostradamus.FormatToken values
func getDatetime(param ...string) (string, error) {
	if err := paramCountCheck(1, len(param)); err != nil {
		return "", err
	}
	return gostradamus.Now().Format(param[0]), nil
}

// returns first parameter if Mercury is in retrograde and second parameter if it is NOT in retrograde
func mercuryInRetrograde(param ...string) (string, error) {
	if err := paramCountCheck(2, len(param)); err != nil {
		return "", err
	}

	yes, err := util.MercuryInRetrograde()
	if err != nil {
		return "", err
	}
	if yes {
		return param[0], nil
	}
	return param[1], nil
}
