package command

import (
	"context"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/otp"
	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/task"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// issueVerificationCode invalidates prior verify codes and issues a fresh OTP
// for the given channel, persisting it and enqueuing delivery to to. It returns
// the raw code so dev mode can echo it back to the client.
func issueVerificationCode(ctx context.Context, codes domain.VerificationCodeRepository, enqueuer queue.Enqueuer, userID, name string, channel domain.Channel, to string, otpLength int, otpTTL time.Duration, clk clock.Clock) (string, error) {
	return issueCode(ctx, codes, enqueuer, userID, name, channel, domain.PurposeVerify, to, otpLength, otpTTL, clk)
}

// issueCode is issueVerificationCode generalized to any purpose. PurposeVerify
// codes confirm a new contact value; PurposeVerifyOld codes are sent to the
// current address when a contact change must be confirmed on the old channel.
func issueCode(ctx context.Context, codes domain.VerificationCodeRepository, enqueuer queue.Enqueuer, userID, name string, channel domain.Channel, purpose domain.Purpose, to string, otpLength int, otpTTL time.Duration, clk clock.Clock) (string, error) {
	if err := codes.InvalidateByUserChannel(ctx, userID, purpose, channel); err != nil {
		return "", err
	}
	code, err := otp.Generate(otpLength)
	if err != nil {
		return "", err
	}
	vc, err := domain.NewVerificationCode(userID, channel, purpose, code, otpTTL, clk.Now())
	if err != nil {
		return "", err
	}
	if err := codes.Save(ctx, vc); err != nil {
		return "", err
	}
	switch channel {
	case domain.ChannelEmail:
		err = task.EnqueueVerificationEmail(ctx, enqueuer, task.VerificationEmailPayload{To: to, Name: name, Code: code})
	case domain.ChannelPhone:
		err = task.EnqueuePhoneVerification(ctx, enqueuer, task.PhoneVerificationPayload{To: to, Code: code})
	}
	if err != nil {
		return "", err
	}
	return code, nil
}
