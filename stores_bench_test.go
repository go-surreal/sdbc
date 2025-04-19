package sdbc

import (
	cryptorand "crypto/rand"
	"encoding/base64"
	"fmt"
	"testing"
)

func BenchmarkNewRequestKey0(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newRequestKey0()
	}
}

func BenchmarkNewRequestKey1(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newRequestKey1()
	}
}

func BenchmarkNewRequestKey2(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newRequestKey2()
	}
}

func BenchmarkNewRequestKey4(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newRequestKey()
	}
}

func BenchmarkNewRequestKey5(b *testing.B) {
	for i := 0; i < b.N; i++ {
		newRequestKey5()
	}
}

// Uniform distribution, but slower and variable key length (<= RequestKeyLength).
func newRequestKey5() string {
	key := make([]byte, RequestKeyLength)
	randBytes.Read(key)

	offs := 0
	for i, b := range key {
		if b > unbiasedMaxVal {
			offs--
			continue
		}
		key[i+offs] = charset[int(b)%charsetLen]
	}

	return string(key[:len(key)+offs])
}

// newRequestKey4 is newRequestKey from stores.go

// newRequestKey3 was a failed attempt

// Similar to official driver.
func newRequestKey2() string {
	return randBytes.Base62Str(RequestKeyLength)
}

// Using simpler rng, and base64.
func newRequestKey1() string {
	key := make([]byte, RequestKeyLength)
	randBytes.Read(key)

	return base64.RawURLEncoding.EncodeToString(key)
}

// Original uuid-like implementation.
func newRequestKey0() string {
	key := make([]byte, RequestKeyLength)

	if _, err := cryptorand.Read(key); err != nil {
		return "" // TODO: error?
	}

	return fmt.Sprintf("%X-%X-%X-%X-%X", key[0:4], key[4:6], key[6:8], key[8:10], key[10:])
}
