package ypars

import (
	"crypto/rand"
	"testing"

	"golang.org/x/crypto/argon2"
	"golang.org/x/crypto/bcrypt"
)

const (
	saltLen     = 16        // bytes
	memory      = 64 * 1024 // bytes
	iterations  = 3
	parallelism = 2
	keyLength   = 32
)

func HashPassword(password string, cost int) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	return string(bytes), err
}

func HashPasswordArgon(password string, saltLen int, iterations, memory uint32, parallelism uint8, keyLength uint32) (string, error) {
	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}

	hash := argon2.IDKey([]byte(password), salt, iterations, memory, parallelism, keyLength)
	return string(hash), nil
}

func benchmarkBcrypt(i int, b *testing.B) {
	for n := 0; n < b.N; n++ {
		HashPassword("my pass", i)
	}
}

func benchmarkArgon(saltLen int, iterations, memory uint32, parallelism uint8, keyLength uint32, b *testing.B) {
	for n := 0; n < b.N; n++ {
		HashPasswordArgon("my pass", saltLen, iterations, memory, parallelism, keyLength)
	}
}

func BenchmarkArgonBase(b *testing.B) {
	benchmarkArgon(saltLen, iterations, memory, parallelism, keyLength, b)
}

func BenchmarkArgonJustShyOfRecommended(b *testing.B) {
	benchmarkArgon(saltLen, 1, memory, 3, keyLength, b)
}

func BenchmarkArgonRecommended(b *testing.B) {
	benchmarkArgon(saltLen, 1, memory, 4, keyLength, b)
}

func BenchmarkArgonRecommended128MB(b *testing.B) {
	benchmarkArgon(saltLen, 1, 128*1024, 4, keyLength, b)
}

func BenchmarkArgonRecommended96MB(b *testing.B) {
	benchmarkArgon(saltLen, 1, 96*1024, 4, keyLength, b)
}

func BenchmarkArgonRecommended2(b *testing.B) {
	benchmarkArgon(saltLen, 1, memory, parallelism, keyLength, b)
}

func BenchmarkArgonSaltx2(b *testing.B) {
	benchmarkArgon(saltLen*2, iterations, memory, parallelism, keyLength, b)
}

func BenchmarkArgonIterationsx2(b *testing.B) {
	benchmarkArgon(saltLen, iterations*2, memory, parallelism, keyLength, b)
}

func BenchmarkArgonMemoryx2(b *testing.B) {
	benchmarkArgon(saltLen, iterations, memory*2, parallelism, keyLength, b)
}

func BenchmarkArgonParallelismx2(b *testing.B) {
	benchmarkArgon(saltLen, iterations, memory, parallelism*2, keyLength, b)
}

func BenchmarkArgonKeyLengthx2(b *testing.B) {
	benchmarkArgon(saltLen, iterations, memory, parallelism, keyLength*2, b)
}

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
