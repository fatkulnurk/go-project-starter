package commands

import (
	"context"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/otp"
	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/tasks"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// issueVerificationCode invalidates prior verify codes and issues a fresh OTP
// for the given channel, persisting it and enqueuing delivery to to. It returns
// the raw code so dev mode can echo it back to the client.
func issueVerificationCode(ctx context.Context, codes domain.VerificationCodeRepository, enqueuer queue.Enqueuer, userID, name string, channel domain.Channel, to string, otpLength int, otpTTL time.Duration, clk clock.Clock) (string, error) {
	if err := codes.InvalidateByUser(ctx, userID, domain.PurposeVerify); err != nil {
		return "", err
	}
	code, err := otp.Generate(otpLength)
	if err != nil {
		return "", err
	}
	vc := domain.NewVerificationCode(userID, channel, domain.PurposeVerify, code, otpTTL, clk.Now())
	if err := codes.Save(ctx, vc); err != nil {
		return "", err
	}
	switch channel {
	case domain.ChannelEmail:
		err = tasks.EnqueueVerificationEmail(ctx, enqueuer, tasks.VerificationEmailPayload{To: to, Name: name, Code: code})
	case domain.ChannelPhone:
		err = tasks.EnqueuePhoneVerification(ctx, enqueuer, tasks.PhoneVerificationPayload{To: to, Code: code})
	}
	if err != nil {
		return "", err
	}
	return code, nil
}
