package commands

import (
	"context"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

func nowUTC() time.Time { return time.Now().UTC() }

// validateCode applies shared checks to a verification code: existence,
// consumption, expiry, attempt limit and secret match. It consumes the code
// on success and increments the attempt counter on a wrong code.
func validateCode(ctx context.Context, codes domain.VerificationCodeRepository, code *domain.VerificationCode, rawCode string, maxAttempts int) error {
	if code == nil {
		return domain.ErrInvalid
	}
	if code.IsConsumed() {
		return domain.ErrInvalid
	}
	if code.IsExpired(nowUTC()) {
		return domain.ErrCodeExpired
	}
	if code.Attempts >= maxAttempts {
		return domain.ErrTooManyAttempts
	}
	if !code.Matches(rawCode) {
		if err := codes.IncrementAttempts(ctx, code.ID, code.Attempts+1); err != nil {
			return err
		}
		return domain.ErrInvalid
	}
	if err := codes.Consume(ctx, code.ID); err != nil {
		return err
	}
	return nil
}
