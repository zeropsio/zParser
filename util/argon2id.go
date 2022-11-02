package util

import (
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/argon2"
)

func Argon2IDPasswordVerify(hash, plain string) error {
	var version int
	var memory uint32
	var time uint32
	var threads uint8

	hashParts := strings.Split(hash, "$")
	if len(hashParts) != 6 {
		return fmt.Errorf("invalid hash, expected 6 parts %d found", len(hashParts))
	}

	if hashParts[1] != "argon2id" {
		return fmt.Errorf("invalid hash, expected [argon2id] algorithm found [%s]", hashParts[1])
	}

	if _, err := fmt.Sscanf(hashParts[2], "v=%d", &version); err != nil {
		return err
	}
	if version != argon2.Version {
		return fmt.Errorf("incompatible version, expected [%d] received [%d]", argon2.Version, version)
	}

	_, err := fmt.Sscanf(hashParts[3], "m=%d,t=%d,p=%d", &memory, &time, &threads)
	if err != nil {
		return err
	}

	salt, err := base64.RawStdEncoding.DecodeString(hashParts[4])
	if err != nil {
		return err
	}

	decodedHash, err := base64.RawStdEncoding.DecodeString(hashParts[5])
	if err != nil {
		return err
	}

	hashToCompare := argon2.IDKey([]byte(plain), salt, time, memory, threads, uint32(len(decodedHash)))
	if subtle.ConstantTimeCompare(decodedHash, hashToCompare) != 1 {
		return errors.New("hashedPassword is not the hash of the given password")
	}
	return nil
}
