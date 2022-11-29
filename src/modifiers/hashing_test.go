package modifiers

import (
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

const (
	password = "my pass"
	saltLen  = 16 // bytes
	keyLen   = 32
)

func HashPasswordBcrypt(password string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(bytes), err
}

func HashPasswordArgon(password string, iterations, memory uint32, parallelism uint8, saltLen int, keyLength uint32) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)
	return string(hash), nil
}

func benchmarkBcrypt(i int, b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _ = HashPasswordBcrypt(password, i)
	}
}

func benchmarkArgon(memory, iterations uint32, parallelism uint8, keyLength uint32, saltLen int, b *testing.B) {
	for n := 0; n < b.N; n++ {
		_, _ = HashPasswordArgon(password, iterations, memory, parallelism, saltLen, keyLength)
	}
}

/**
 * Bcrypt
 */

func BenchmarkBcrypt9(b *testing.B) {
	benchmarkBcrypt(9, b)
}

func BenchmarkBcrypt10(b *testing.B) {
	benchmarkBcrypt(10, b)
}

func BenchmarkBcrypt11(b *testing.B) {
	benchmarkBcrypt(11, b)
}

func BenchmarkBcrypt12(b *testing.B) {
	benchmarkBcrypt(12, b)
}

func BenchmarkBcrypt13(b *testing.B) {
	benchmarkBcrypt(13, b)
}

func BenchmarkBcrypt14(b *testing.B) {
	benchmarkBcrypt(14, b)
}

/**
 * Argon2id
 */

func BenchmarkArgon32m_1i_1p(b *testing.B) {
	benchmarkArgon(32*1024, 1, 1, keyLen, saltLen, b)
}

func BenchmarkArgon32m_2i_1p(b *testing.B) {
	benchmarkArgon(32*1024, 2, 1, keyLen, saltLen, b)
}

func BenchmarkArgon32m_2i_2p(b *testing.B) {
	benchmarkArgon(32*1024, 2, 2, keyLen, saltLen, b)
}

func BenchmarkArgon32m_4i_2p(b *testing.B) {
	benchmarkArgon(32*1024, 4, 2, keyLen, saltLen, b)
}

func BenchmarkArgon64m_1i_1p(b *testing.B) {
	benchmarkArgon(64*1024, 1, 1, keyLen, saltLen, b)
}

func BenchmarkArgon64m_1i_2p(b *testing.B) {
	benchmarkArgon(64*1024, 1, 2, keyLen, saltLen, b)
}

func BenchmarkArgon64m_2i_1p(b *testing.B) {
	benchmarkArgon(64*1024, 2, 1, keyLen, saltLen, b)
}

func BenchmarkArgon64m_2i_2p(b *testing.B) {
	benchmarkArgon(64*1024, 2, 2, keyLen, saltLen, b)
}

func BenchmarkArgon64m_2i_4p(b *testing.B) {
	benchmarkArgon(64*1024, 2, 4, keyLen, saltLen, b)
}

func BenchmarkArgon64m_4i_2p(b *testing.B) {
	benchmarkArgon(64*1024, 4, 2, keyLen, saltLen, b)
}

func BenchmarkArgon64m_4i_4p(b *testing.B) {
	benchmarkArgon(64*1024, 4, 4, keyLen, saltLen, b)
}

func BenchmarkArgon96m_1i_1p(b *testing.B) {
	benchmarkArgon(96*1024, 1, 1, keyLen, saltLen, b)
}

func BenchmarkArgon96m_1i_2p(b *testing.B) {
	benchmarkArgon(96*1024, 1, 2, keyLen, saltLen, b)
}

func BenchmarkArgon96m_2i_1p(b *testing.B) {
	benchmarkArgon(96*1024, 2, 1, keyLen, saltLen, b)
}

func BenchmarkArgon96m_2i_2p(b *testing.B) {
	benchmarkArgon(96*1024, 2, 2, keyLen, saltLen, b)
}

func BenchmarkArgon96m_4i_2p(b *testing.B) {
	benchmarkArgon(96*1024, 4, 2, keyLen, saltLen, b)
}

func BenchmarkArgon96m_4i_4p(b *testing.B) {
	benchmarkArgon(96*1024, 4, 2, keyLen, saltLen, b)
}
