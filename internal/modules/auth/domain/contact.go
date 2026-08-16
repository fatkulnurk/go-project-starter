package domain

import (
	"net/mail"
	"regexp"
	"strings"
)

// contactCleaner strips everything but digits and a leading '+'.
var contactCleaner = regexp.MustCompile(`[^0-9+]`)

// NormalizeEmail validates and canonicalizes an email address (lowercased and
// trimmed). It returns ErrInvalid for malformed input.
func NormalizeEmail(raw string) (string, error) {
	email := strings.ToLower(strings.TrimSpace(raw))
	if email == "" {
		return "", ErrInvalid
	}
	parsed, err := mail.ParseAddress(email)
	if err != nil {
		return "", ErrInvalid
	}
	normalized := strings.ToLower(strings.TrimSpace(parsed.Address))
	if !strings.Contains(normalized, "@") {
		return "", ErrInvalid
	}
	return normalized, nil
}

// NormalizePhone canonicalizes a phone number toward E.164: separators are
// stripped, a leading "00" becomes '+', and a local number starting with "0" is
// expanded with defaultCountryCode (e.g. "62" for Indonesia) when provided.
// When the country code is known, a trunk zero typed after it ("+62 0812...")
// is dropped so every form of one subscriber folds to a single canonical
// number. The result must be '+' followed by 7-15 digits and never a leading
// zero after '+' (ambiguous without a country code). Returns ErrInvalid for
// unparseable input.
func NormalizePhone(raw, defaultCountryCode string) (string, error) {
	phone := contactCleaner.ReplaceAllString(strings.TrimSpace(raw), "")
	if phone == "" {
		return "", ErrInvalid
	}
	if strings.HasPrefix(phone, "00") {
		phone = "+" + phone[2:]
	}
	cc := strings.TrimPrefix(defaultCountryCode, "+")
	if strings.HasPrefix(phone, "0") && cc != "" {
		// Local number like "0812..." → "+62" + "812...".
		phone = "+" + cc + phone[1:]
	}
	if !strings.HasPrefix(phone, "+") {
		phone = "+" + phone
	}
	if cc != "" {
		// Subscriber kept the trunk zero after the country code ("+62 0812..."):
		// drop it so it equals the "+62812..." form.
		if rest, ok := strings.CutPrefix(phone, "+"+cc+"0"); ok {
			phone = "+" + cc + rest
		}
		if strings.HasPrefix(phone, "+0") {
			return "", ErrInvalid
		}
	} else if strings.HasPrefix(phone, "+0") {
		// A leading zero without a known country code is ambiguous.
		return "", ErrInvalid
	}
	if !isValidPhone(phone) {
		return "", ErrInvalid
	}
	return phone, nil
}

// IsValidEmail is a backstop validator used by domain constructors; full
// normalization happens in the use cases that have the country code.
func IsValidEmail(raw string) bool {
	_, err := NormalizeEmail(raw)
	return err == nil
}

// IsValidPhone is a backstop validator used by domain constructors: '+' must
// be followed by 7-15 digits.
func IsValidPhone(raw string) bool {
	return isValidPhone(contactCleaner.ReplaceAllString(strings.TrimSpace(raw), ""))
}

func isValidPhone(phone string) bool {
	if !strings.HasPrefix(phone, "+") {
		return false
	}
	digits := strings.TrimPrefix(phone, "+")
	if len(digits) < 7 || len(digits) > 15 {
		return false
	}
	for _, c := range digits {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
