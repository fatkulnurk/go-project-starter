package command

import (
	"strings"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

// normalizeEmailOptional validates a provided email, returning "" for empty
// input and ErrInvalid for malformed addresses.
func normalizeEmailOptional(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return domain.NormalizeEmail(raw)
}

// normalizePhoneOptional validates a provided phone, returning "" for empty
// input and ErrInvalid for unparseable numbers.
func normalizePhoneOptional(raw, countryCode string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", nil
	}
	return domain.NormalizePhone(raw, countryCode)
}

// normalizeIdentifier canonicalizes an email or phone identifier. Emails are
// lowercased/validated; phones go through E.164 normalization.
func normalizeIdentifier(raw, countryCode string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", domain.ErrInvalid
	}
	if strings.Contains(raw, "@") {
		return domain.NormalizeEmail(raw)
	}
	return domain.NormalizePhone(raw, countryCode)
}
