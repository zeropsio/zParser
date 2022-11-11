package functions

import (
	cryptoRand "crypto/rand"
	"encoding/hex"
	"fmt"
	"math"
	"math/big"
	mathRand "math/rand"
	"strings"
	"testing"
	"time"
	"unsafe"
)

const stringLength = 16

// Comparison of different ways to generate random string in GO
//
// RandStringRange is taken from random SO/GitHub tutorials for comparison (just by the looks of it, it performs horrible)
// RandStringFmt and RandStringFmtEncode are simple custom versions
// Rest of the implementations are taken from https://stackoverflow.com/a/31832326/2228606
//
// The target is to have a relatively simple function to understand, which is also performant enough

// Implementations
func init() {
	mathRand.Seed(time.Now().UnixNano())
}

//nolint:gochecknoglobals
var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[mathRand.Intn(len(letterRunes))]
	}
	return string(b)
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

func RandStringBytes(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[mathRand.Intn(len(letterBytes))]
	}
	return string(b)
}

func RandStringBytesRmndr(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[mathRand.Int63()%int64(len(letterBytes))]
	}
	return string(b)
}

func RandStringBytesMask(n int) string {
	b := make([]byte, n)
	for i := 0; i < n; {
		if idx := int(mathRand.Int63() & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i++
		}
	}
	return string(b)
}

func RandStringBytesMaskImpr(n int) string {
	b := make([]byte, n)
	// A mathRand.Int63() generates 63 random bits, enough for letterIdxMax letters!
	for i, cache, remain := n-1, mathRand.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = mathRand.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

//nolint:gochecknoglobals
var src = mathRand.NewSource(time.Now().UnixNano())

func RandStringBytesMaskImprSrc(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return string(b)
}

func RandStringBytesMaskImprSrcSB(n int) string {
	sb := strings.Builder{}
	sb.Grow(n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			sb.WriteByte(letterBytes[idx])
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return sb.String()
}

func RandStringBytesMaskImprSrcUnsafe(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func RandStringRange(n int) string {
	b := make([]byte, n)
	for i := range b {
		num, err := cryptoRand.Int(cryptoRand.Reader, big.NewInt(int64(len(letterBytes))))
		if err != nil {
			return ""
		}
		b[i] = letterBytes[num.Int64()]
	}
	return string(b)
}

func RandStringFmt(n int) string {
	result := make([]byte, int(math.Ceil(float64(n)/2)))
	if _, err := mathRand.Read(result); err != nil {
		return ""
	}
	return fmt.Sprintf("%x", result)[:n]
}

func RandStringFmtEncode(n int) string {
	result := make([]byte, int(math.Ceil(float64(n)/2)))
	if _, err := mathRand.Read(result); err != nil {
		return ""
	}
	return hex.EncodeToString(result)[:n]
}

// Benchmark functions

func BenchmarkRunes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringRunes(stringLength)
	}
}

func BenchmarkBytes(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringBytes(stringLength)
	}
}

func BenchmarkBytesRmndr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringBytesRmndr(stringLength)
	}
}

func BenchmarkBytesMask(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringBytesMask(stringLength)
	}
}

func BenchmarkBytesMaskImpr(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringBytesMaskImpr(stringLength)
	}
}

func BenchmarkBytesMaskImprSrc(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringBytesMaskImprSrc(stringLength)
	}
}
func BenchmarkBytesMaskImprSrcSB(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringBytesMaskImprSrcSB(stringLength)
	}
}

func BenchmarkBytesMaskImprSrcUnsafe(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringBytesMaskImprSrcUnsafe(stringLength)
	}
}

func BenchmarkRandStringRange(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringRange(stringLength)
	}
}

func BenchmarkRandStringFmt(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringFmt(stringLength)
	}
}

func BenchmarkRandStringFmtEncode(b *testing.B) {
	for i := 0; i < b.N; i++ {
		RandStringFmtEncode(stringLength)
	}
}
