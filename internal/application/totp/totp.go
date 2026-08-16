// Package totp implements RFC 6238 time-based one-time passwords with the
// standard library only (no external dependency). It is a cross-cutting
// technical helper usable by any module; the auth module uses it for MFA.
package totp

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"net/url"
	"strings"
	"time"
)

const (
	// Period is the standard 30-second TOTP time step.
	Period = 30
	// Digits is the standard 6-digit code length.
	Digits = 6
	// SecretKeyLength is the shared-secret length in bytes (160 bits), which
	// encodes to 32 base32 characters.
	SecretKeyLength = 20
)

// GenerateSecret returns a new random base32 shared secret (no padding),
// suitable for an authenticator app.
func GenerateSecret() (string, error) {
	b := make([]byte, SecretKeyLength)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(b), nil
}

// ProvisioningURI builds the otpauth:// URI users scan with an authenticator
// app. issuer is the shown account issuer; account is the user's identifier.
func ProvisioningURI(issuer, account, secret string) string {
	label := url.PathEscape(strings.TrimSpace(account))
	if label == "" {
		label = "user"
	}
	query := url.Values{}
	query.Set("secret", secret)
	query.Set("issuer", issuer)
	query.Set("algorithm", "SHA1")
	query.Set("digits", fmt.Sprintf("%d", Digits))
	query.Set("period", fmt.Sprintf("%d", Period))
	return "otpauth://totp/" + label + "?" + query.Encode()
}

// Validate reports whether code is a valid TOTP for secret within the given
// clock-drift window (in periods). A window of 1 accepts one step before and
// after the current one, which covers normal clock skew.
func Validate(secret, code string, window int) (bool, error) {
	code = strings.TrimSpace(code)
	if window < 0 {
		window = 0
	}
	counter := time.Now().Unix() / Period
	for d := -window; d <= window; d++ {
		expected, err := codeAt(counter+int64(d), secret)
		if err != nil {
			return false, err
		}
		if subtle.ConstantTimeCompare([]byte(expected), []byte(code)) == 1 {
			return true, nil
		}
	}
	return false, nil
}

// codeAt computes the TOTP code for the given time counter and secret.
func codeAt(counter int64, secret string) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", fmt.Errorf("totp: invalid secret: %w", err)
	}
	buf := make([]byte, 8)
	binary.BigEndian.PutUint64(buf, uint64(counter))
	mac := hmac.New(sha1.New, key)
	mac.Write(buf)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	trunc := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	code := trunc % 1000000
	return fmt.Sprintf("%06d", code), nil
}

func decodeSecret(secret string) ([]byte, error) {
	raw := strings.ToUpper(strings.TrimSpace(secret))
	raw = strings.ReplaceAll(raw, " ", "")
	raw = strings.TrimRight(raw, "=")
	if key, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(raw); err == nil {
		return key, nil
	}
	return base32.StdEncoding.DecodeString(raw)
}
