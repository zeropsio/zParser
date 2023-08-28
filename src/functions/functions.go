package functions

import (
	"crypto/ed25519"
	cryptoRand "crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"math"
	"math/big"
	mathRand "math/rand"
	"strconv"
	"strings"
	"time"

	"github.com/bykof/gostradamus"
	"golang.org/x/crypto/ssh"

	"github.com/zeropsio/zParser/src/util"
)

const maxRandStringLen = 1024
const randStringChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-."
const randStringMaxCharIdx = 64

type function func(param ...string) (string, error)

type Functions struct {
	values    map[string]string
	functions map[string]function
}

func NewFunctions(valueStore map[string]string) *Functions {
	f := &Functions{
		values: valueStore,
	}
	f.functions = map[string]function{
		"generateRandomInt":       f.generateRandomInt,
		"generateRandomString":    f.generateRandomString,
		"generateRandomStringVar": f.generateRandomStringVar,
		"pickRandom":              f.pickRandom,
		"mercuryInRetrograde":     f.mercuryInRetrograde,
		"getDatetime":             f.getDatetime,
		"setVar":                  f.setVar,
		"getVar":                  f.getVar,
		"generateED25519Key":      f.generateED25519Key,
		"generateRSA2048Key":      f.generateRSA2048Key,
		"generateRSA4096Key":      f.generateRSA4096Key,
	}
	return f
}

func (f Functions) Call(name string, params ...string) (string, error) {
	fn, found := f.functions[name]
	if !found {
		return "", fmt.Errorf("function [%s] not found", name)
	}
	return fn(params...)
}

// generates cryptographically secure random int in [min, max]
func (f Functions) generateRandomInt(param ...string) (string, error) {
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

	n, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(max-min+1))
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(n.Int64()+min, 10), nil
}

// generates cryptographically secure random string of specified length given its <= maxRandStringLen
func (f Functions) generateRandomString(param ...string) (string, error) {
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

	// original version with hex encoding (ditched due to limited character set)
	// result := make([]byte, int(math.Ceil(float64(length)/2)))
	// if _, err := cryptoRand.Read(result); err != nil {
	// 	return "", err
	// }
	// return hex.EncodeToString(result)[:length], nil

	result := make([]byte, length)
	if _, err := cryptoRand.Read(result); err != nil {
		return "", err
	}
	for i, b := range result {
		n := float64(b)
		if n > randStringMaxCharIdx {
			n = math.Mod(n, randStringMaxCharIdx)
		}
		result[i] = randStringChars[int(n)]
	}
	return string(result), nil
}

// selects one random value from all provided parameters
func (f Functions) pickRandom(param ...string) (string, error) {
	if len(param) == 0 {
		return "", fmt.Errorf("invalid parameter count, at least 1 expected %d provided", len(param))
	}
	mathRand.Seed(time.Now().UnixNano())
	return param[mathRand.Intn(len(param))], nil
}

// returns date time using formatted by format inside first parameter which supports gostradamus.FormatToken values
// if second parameter is provided, it is used as a timezone, otherwise UTC is assumed
func (f Functions) getDatetime(param ...string) (string, error) {
	if len(param) == 1 {
		return gostradamus.UTCNow().Format(param[0]), nil
	}
	if len(param) == 2 {
		if _, err := gostradamus.LoadLocation(param[1]); err != nil {
			return "", err
		}
		return gostradamus.NowInTimezone(gostradamus.Timezone(param[1])).Format(param[0]), nil
	}
	return "", fmt.Errorf("invalid parameter count, at 1 or 2 expected %d provided", len(param))
}

// returns first parameter if Mercury is in retrograde and second parameter if it is NOT in retrograde
func (f Functions) mercuryInRetrograde(param ...string) (string, error) {
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

func (f Functions) generateRandomStringVar(param ...string) (string, error) {
	if err := paramCountCheck(2, len(param)); err != nil {
		return "", err
	}
	str, err := f.generateRandomString(param[1])
	if err != nil {
		return "", err
	}

	f.values[param[0]] = str
	return str, nil
}

func (f Functions) setVar(param ...string) (string, error) {
	if err := paramCountCheck(2, len(param)); err != nil {
		return "", err
	}

	f.values[param[0]] = param[1]
	return param[1], nil
}

func (f Functions) getVar(param ...string) (string, error) {
	if err := paramCountCheck(1, len(param)); err != nil {
		return "", err
	}
	return param[0], nil
}

func (f Functions) generateED25519Key(param ...string) (string, error) {
	if err := paramCountCheck(1, len(param)); err != nil {
		return "", err
	}

	publicKey, privateKey, _ := ed25519.GenerateKey(cryptoRand.Reader)

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
	f.values[name+"Public"] = strings.TrimSpace(string(pem.EncodeToMemory(publicPem)))
	f.values[name+"Private"] = strings.TrimSpace(string(pem.EncodeToMemory(privatePem)))
	f.values[name+"PublicSsh"] = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicSshKey)))
	f.values[name+"PrivateSsh"] = strings.TrimSpace(string(pem.EncodeToMemory(privateOpenSshPem)))

	return f.values[name+"Public"], nil
}

func (f Functions) generateRSA2048Key(param ...string) (string, error) {
	if err := paramCountCheck(1, len(param)); err != nil {
		return "", err
	}

	return f.generateRSAKey(param[0], 2048)
}

func (f Functions) generateRSA4096Key(param ...string) (string, error) {
	if err := paramCountCheck(1, len(param)); err != nil {
		return "", err
	}

	return f.generateRSAKey(param[0], 4096)
}

func (f Functions) generateRSAKey(name string, bits int) (string, error) {
	privateKey, err := rsa.GenerateKey(cryptoRand.Reader, bits)
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

	f.values[name+"Public"] = strings.TrimSpace(string(pem.EncodeToMemory(publicPem)))
	f.values[name+"Private"] = strings.TrimSpace(string(pem.EncodeToMemory(privatePem)))
	f.values[name+"PublicSsh"] = strings.TrimSpace(string(ssh.MarshalAuthorizedKey(publicSshKey)))

	return f.values[name+"Public"], nil
}

func paramCountCheck(expected, received int) error {
	if expected != received {
		return fmt.Errorf("invalid parameter count, %d expected %d provided", expected, received)
	}
	return nil
}
