// Package tasks defines the queue task types and payloads used by the auth
// module, together with helpers to enqueue them. The worker processes these
// tasks by sending email/SMS through the application contracts.
package tasks

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
)

// Task types.
const (
	SendVerificationEmail   = "auth:send_verification_email"
	SendPhoneVerification   = "auth:send_phone_verification"
	SendForgotPasswordEmail = "auth:send_forgot_password_email"
	SendForgotPasswordSMS   = "auth:send_forgot_password_sms"
	SendMagicLinkEmail      = "auth:send_magic_link_email"
)

// VerificationEmailPayload carries what the worker needs to send a
// verification code by email.
type VerificationEmailPayload struct {
	To   string `json:"to"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// PhoneVerificationPayload carries what the worker needs to send a
// verification code by SMS.
type PhoneVerificationPayload struct {
	To   string `json:"to"`
	Code string `json:"code"`
}

// ForgotPasswordEmailPayload carries a password reset code by email.
type ForgotPasswordEmailPayload struct {
	To   string `json:"to"`
	Name string `json:"name"`
	Code string `json:"code"`
}

// ForgotPasswordSMSPayload carries a password reset code by SMS.
type ForgotPasswordSMSPayload struct {
	To   string `json:"to"`
	Code string `json:"code"`
}

// MagicLinkEmailPayload carries a magic login link by email.
type MagicLinkEmailPayload struct {
	To   string `json:"to"`
	Name string `json:"name"`
	Link string `json:"link"`
}

// EnqueueVerificationEmail pushes a verification-email task.
func EnqueueVerificationEmail(ctx context.Context, e queue.Enqueuer, p VerificationEmailPayload) error {
	return enqueue(ctx, e, SendVerificationEmail, p, 5)
}

// EnqueuePhoneVerification pushes a phone verification task.
func EnqueuePhoneVerification(ctx context.Context, e queue.Enqueuer, p PhoneVerificationPayload) error {
	return enqueue(ctx, e, SendPhoneVerification, p, 5)
}

// EnqueueForgotPasswordEmail pushes a reset-code email task.
func EnqueueForgotPasswordEmail(ctx context.Context, e queue.Enqueuer, p ForgotPasswordEmailPayload) error {
	return enqueue(ctx, e, SendForgotPasswordEmail, p, 5)
}

// EnqueueForgotPasswordSMS pushes a reset-code SMS task.
func EnqueueForgotPasswordSMS(ctx context.Context, e queue.Enqueuer, p ForgotPasswordSMSPayload) error {
	return enqueue(ctx, e, SendForgotPasswordSMS, p, 5)
}

// EnqueueMagicLinkEmail pushes a magic link email task.
func EnqueueMagicLinkEmail(ctx context.Context, e queue.Enqueuer, p MagicLinkEmailPayload) error {
	return enqueue(ctx, e, SendMagicLinkEmail, p, 5)
}

func enqueue(ctx context.Context, e queue.Enqueuer, taskType string, payload any, maxRetry int) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal %s payload: %w", taskType, err)
	}
	return e.Enqueue(ctx, queue.Task{Type: taskType, Payload: b, MaxRetry: maxRetry})
}
