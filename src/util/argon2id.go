package util

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"math"
	"strings"

	"golang.org/x/crypto/argon2"
)

// Argon2idConfig provides an easier structure for Argon2id password hashing.
//
// The main gist (not 100% accurate, but close enough) is: complexity = memory * iterations / parallelism.
type Argon2idConfig struct {
	Memory      uint32 // KiB - main "knob" to turn for more expensive hashes
	Iterations  uint32 // if memory cant go higher, iterations should, to compensate
	Parallelism uint8  // used to spread the load across multiple CPU threads (if used, iterations should be increased)

	SaltLen uint32 // bytes
	KeyLen  uint32 // bytes
}

// DefaultArgon2idConf provides standard sane parameters chosen to not overload the parser service.
func DefaultArgon2idConf() Argon2idConfig {
	return Argon2idConfig{
		Memory:      64 * 1024,
		Iterations:  4,
		Parallelism: 4, // parallelism set to 4 to better spread the CPU load, that's why iterations = 4

		SaltLen: 16,
		KeyLen:  32,
	}
}

func Argon2IDPasswordHash(plain string, conf Argon2idConfig) (string, error) {
	salt := make([]byte, conf.SaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(plain), salt, conf.Iterations, conf.Memory, conf.Parallelism, conf.KeyLen)

	// Base64 encode the salt and hashed password.
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Return a string using the standard encoded hash representation.
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s", argon2.Version, conf.Memory, conf.Iterations, conf.Parallelism, b64Salt, b64Hash), nil
}

func Argon2IDPasswordVerify(hash, plain string) error {
	var version int
	var memory uint32
	var time uint32
	var threads uint8

	hashParts := strings.Split(hash, "$")
	if len(hashParts) != 6 {
		return fmt.Errorf("invalid hash, expected [6] parts found [%d]", len(hashParts))
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

	hashLen := len(decodedHash)
	if hashLen > math.MaxUint32 {
		return fmt.Errorf("invalid decoded hash length %d, exceeds max value for uint32", hashLen)
	}

	hashToCompare := argon2.IDKey([]byte(plain), salt, time, memory, threads, uint32(hashLen))
	if subtle.ConstantTimeCompare(decodedHash, hashToCompare) != 1 {
		return errors.New("hashedPassword is not the hash of the given password")
	}
	return nil
}
