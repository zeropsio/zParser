package util

import "math"

const (
	randStringChars      = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789_-."
	randStringMaxCharIdx = 64
)

func BytesToString(in []byte) string {
	for i, b := range in {
		n := float64(b)
		if n > randStringMaxCharIdx {
			n = math.Mod(n, randStringMaxCharIdx)
		}
		in[i] = randStringChars[int(n)]
	}
	return string(in)
}
