// Package otp provides a shared generator for one-time passcodes. Cross-cutting
// technical helper (not business logic), usable by any module.
package otp

import (
	"crypto/rand"
	"math/big"
)

// Generate returns a numeric OTP of n digits (n >= 1). It is not cryptographically
// strong enough for authentication on its own — pair it with hashing, TTL and
// attempt limits, and always deliver over a trusted channel.
func Generate(n int) (string, error) {
	if n < 1 {
		n = 6
	}
	const digits = "0123456789"
	out := make([]byte, n)
	for i := range out {
		v, err := rand.Int(rand.Reader, big.NewInt(int64(len(digits))))
		if err != nil {
			return "", err
		}
		out[i] = digits[v.Int64()]
	}
	return string(out), nil
}
