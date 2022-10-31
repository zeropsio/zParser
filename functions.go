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
	"time"

	"github.com/bykof/gostradamus"
	"golang.org/x/crypto/ssh"

	"git.vsh-labs.cz/zerops/yaml-parser/util"
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
			"mercuryInRetrograde":  mercuryInRetrograde,
			"getDatetime":          getDatetime,
			"writeString":          writeString,
		},
	}

	y.functions["generateRandomNamedString"] = func(param ...string) (string, error) {
		if len(param) != 2 {
			return "", fmt.Errorf("invalid parameter count, 2 expected %d provided", len(param))
		}
		str, err := generateRandomString(param[1])
		if err != nil {
			return "", err
		}

		y.namedValues[param[0]] = str
		return str, nil
	}

	y.functions["namedString"] = func(param ...string) (string, error) {
		if len(param) != 2 {
			return "", fmt.Errorf("invalid parameter count, 2 expected %d provided", len(param))
		}
		y.namedValues[param[0]] = param[1]
		return param[1], nil
	}

	y.functions["getNamedString"] = func(param ...string) (string, error) {
		if len(param) != 1 {
			return "", fmt.Errorf("invalid parameter count, 1 expected %d provided", len(param))
		}
		val, found := y.namedValues[param[0]]
		if !found {
			return "", fmt.Errorf("no stored value for key [%s]", param[0])
		}
		return val, nil
	}

	y.functions["generateEd25519Key"] = func(param ...string) (string, error) {
		if len(param) != 1 {
			return "", fmt.Errorf("invalid parameter count, 1 expected %d provided", len(param))
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
		return "", fmt.Errorf("invalid parameter count, 2 expected %d provided", len(param))
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
		return "", fmt.Errorf("invalid parameter count, 1 expected %d provided", len(param))
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

func getDatetime(param ...string) (string, error) {
	if len(param) != 1 {
		return "", fmt.Errorf("invalid parameter count, 1 expected %d provided", len(param))
	}

	return gostradamus.Now().Format(param[0]), nil
}

func writeString(param ...string) (string, error) {
	if len(param) != 1 {
		return "", fmt.Errorf("invalid parameter count, 1 expected %d provided", len(param))
	}
	return param[0], nil
}

func mercuryInRetrograde(param ...string) (string, error) {
	if len(param) != 2 {
		return "", fmt.Errorf("invalid parameter count, 2 expected %d provided", len(param))
	}

	type dateRange struct {
		begin []int
		end   []int
	}

	dates := map[int][]dateRange{
		2022: {
			dateRange{begin: []int{14, 1, 2022}, end: []int{3, 2, 2022}},
			dateRange{begin: []int{10, 5, 2022}, end: []int{3, 6, 2022}},
			dateRange{begin: []int{9, 9, 2022}, end: []int{2, 10, 2022}},
			dateRange{begin: []int{29, 12, 2022}, end: []int{18, 1, 2023}},
		},
		2023: {
			dateRange{begin: []int{29, 12, 2022}, end: []int{18, 1, 2023}},
			dateRange{begin: []int{21, 4, 2023}, end: []int{14, 5, 2023}},
			dateRange{begin: []int{23, 8, 2023}, end: []int{15, 9, 2023}},
			dateRange{begin: []int{13, 12, 2023}, end: []int{1, 1, 2024}},
		},
		2024: {
			dateRange{begin: []int{1, 4, 2024}, end: []int{25, 4, 2024}},
			dateRange{begin: []int{4, 8, 2024}, end: []int{28, 8, 2024}},
			dateRange{begin: []int{25, 11, 2024}, end: []int{15, 12, 2024}},
		},
		2025: {
			dateRange{begin: []int{25, 2, 2025}, end: []int{20, 3, 2025}},
			dateRange{begin: []int{29, 6, 2025}, end: []int{23, 7, 2025}},
			dateRange{begin: []int{24, 10, 2025}, end: []int{13, 11, 2025}},
		},
		2026: {
			dateRange{begin: []int{25, 2, 2026}, end: []int{203, 3, 2026}},
			dateRange{begin: []int{29, 6, 2026}, end: []int{23, 7, 2026}},
			dateRange{begin: []int{24, 10, 2026}, end: []int{13, 11, 2026}},
		},
		2027: {
			dateRange{begin: []int{9, 2, 2027}, end: []int{3, 3, 2027}},
			dateRange{begin: []int{10, 6, 2027}, end: []int{4, 7, 2027}},
			dateRange{begin: []int{7, 10, 2027}, end: []int{28, 10, 2027}},
		},
		2028: {
			dateRange{begin: []int{24, 1, 2028}, end: []int{14, 2, 2028}},
			dateRange{begin: []int{21, 5, 2028}, end: []int{13, 6, 2028}},
			dateRange{begin: []int{19, 9, 2028}, end: []int{11, 10, 2028}},
		},
		2029: {
			dateRange{begin: []int{7, 1, 2029}, end: []int{27, 1, 2029}},
			dateRange{begin: []int{1, 5, 2029}, end: []int{25, 5, 2029}},
			dateRange{begin: []int{2, 9, 2029}, end: []int{24, 9, 2029}},
			dateRange{begin: []int{21, 12, 2029}, end: []int{10, 1, 2030}},
		},
		2030: {
			dateRange{begin: []int{21, 12, 2029}, end: []int{10, 1, 2030}},
			dateRange{begin: []int{12, 4, 2030}, end: []int{6, 5, 2030}},
			dateRange{begin: []int{15, 8, 2030}, end: []int{8, 9, 2030}},
			dateRange{begin: []int{5, 12, 2030}, end: []int{25, 12, 2030}},
		},
	}

	now := time.Now()
	d, found := dates[now.Year()]
	if !found {
		return fmt.Sprintf("current year [%d] is not supported, latest supported year was [%d]", now.Year(), 2030), nil
	}

	for _, dateR := range d {
		if now.After(time.Date(dateR.begin[2], time.Month(dateR.begin[1]), dateR.begin[0], 0, 0, 0, 0, time.UTC)) &&
			now.Before(time.Date(dateR.end[2], time.Month(dateR.end[1]), dateR.end[0], 0, 0, 0, 0, time.UTC)) {
			return param[0], nil
		}
	}

	return param[1], nil
}
