package command

import (
	"context"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
)

func nowUTC() time.Time { return time.Now().UTC() }

// validateCode applies shared checks to a verification code: existence,
// consumption, expiry, attempt limit and secret match. It consumes the code
// on success and increments the attempt counter on a wrong code. Consumption
// and the attempt budget are enforced atomically in the repository so
// concurrent requests cannot double-spend a code or exhaust the attempts.
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
		if err := codes.IncrementAttempts(ctx, code.ID, maxAttempts); err != nil {
			return err
		}
		return domain.ErrInvalid
	}
	ok, err := codes.Consume(ctx, code.ID)
	if err != nil {
		return err
	}
	if !ok {
		return domain.ErrInvalid
	}
	return nil
}
