package commands

import (
	"context"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"github.com/fatkulnurk/go-project-starter/internal/application/otp"
	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/ports"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/tasks"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

// RegisterCommand is the input for user registration.
type RegisterCommand struct {
	Name     string
	Email    string
	Phone    string
	Password string
}

// RegisterResult reports the created user; codes are returned only in dev
// mode so the flow can be exercised without a real mail/SMS channel.
type RegisterResult struct {
	UserID       string
	DevEmailCode string
	DevPhoneCode string
}

// Register creates a user and issues verification codes.
type Register struct {
	users          domain.UserRepository
	codes          domain.VerificationCodeRepository
	hasher         hash.PasswordHasher
	enqueuer       queue.Enqueuer
	roles          ports.Roles
	clock          clock.Clock
	otpLength      int
	otpTTL         time.Duration
	otpMaxAttempts int
	devMode        bool
}

// NewRegister builds the register use case. roles may be nil when RBAC is not
// wired; the user is then created without any role.
func NewRegister(users domain.UserRepository, codes domain.VerificationCodeRepository, hasher hash.PasswordHasher, enqueuer queue.Enqueuer, roles ports.Roles, clk clock.Clock, otpLength int, otpTTL time.Duration, otpMaxAttempts int, devMode bool) *Register {
	return &Register{users: users, codes: codes, hasher: hasher, enqueuer: enqueuer, roles: roles, clock: clk, otpLength: otpLength, otpTTL: otpTTL, otpMaxAttempts: otpMaxAttempts, devMode: devMode}
}

// Execute runs the use case.
func (uc *Register) Execute(ctx context.Context, cmd RegisterCommand) (*RegisterResult, error) {
	cmd.Name = strings.TrimSpace(cmd.Name)
	cmd.Email = strings.ToLower(strings.TrimSpace(cmd.Email))
	cmd.Phone = strings.TrimSpace(cmd.Phone)
	if cmd.Name == "" {
		return nil, domain.ErrInvalid
	}
	if len(cmd.Password) < 8 {
		return nil, domain.ErrInvalid
	}
	if cmd.Email == "" && cmd.Phone == "" {
		return nil, domain.ErrInvalid
	}

	if cmd.Email != "" {
		if existing, err := uc.users.FindByEmail(ctx, cmd.Email); err != nil {
			return nil, err
		} else if existing != nil {
			return nil, domain.ErrConflict
		}
	}
	if cmd.Phone != "" {
		if existing, err := uc.users.FindByPhone(ctx, cmd.Phone); err != nil {
			return nil, err
		} else if existing != nil {
			return nil, domain.ErrConflict
		}
	}

	passwordHash, err := uc.hasher.Hash(ctx, cmd.Password)
	if err != nil {
		return nil, err
	}
	user, err := domain.NewUser(cmd.Name, cmd.Email, cmd.Phone, passwordHash)
	if err != nil {
		return nil, err
	}
	if err := uc.users.Save(ctx, user); err != nil {
		return nil, err
	}
	if uc.roles != nil {
		if err := uc.roles.AssignDefaultRole(ctx, user.ID); err != nil {
			return nil, err
		}
	}

	res := &RegisterResult{UserID: user.ID}
	if cmd.Email != "" {
		if err := uc.codes.InvalidateByUser(ctx, user.ID, domain.PurposeVerify); err != nil {
			return nil, err
		}
		code, err := otp.Generate(uc.otpLength)
		if err != nil {
			return nil, err
		}
		vc := domain.NewVerificationCode(user.ID, domain.ChannelEmail, domain.PurposeVerify, code, uc.otpTTL, uc.clock.Now())
		if err := uc.codes.Save(ctx, vc); err != nil {
			return nil, err
		}
		if err := tasks.EnqueueVerificationEmail(ctx, uc.enqueuer, tasks.VerificationEmailPayload{
			To: *user.Email, Name: user.Name, Code: code,
		}); err != nil {
			return nil, err
		}
		if uc.devMode {
			res.DevEmailCode = code
		}
	}
	if cmd.Phone != "" {
		if err := uc.codes.InvalidateByUser(ctx, user.ID, domain.PurposeVerify); err != nil {
			return nil, err
		}
		code, err := otp.Generate(uc.otpLength)
		if err != nil {
			return nil, err
		}
		vc := domain.NewVerificationCode(user.ID, domain.ChannelPhone, domain.PurposeVerify, code, uc.otpTTL, uc.clock.Now())
		if err := uc.codes.Save(ctx, vc); err != nil {
			return nil, err
		}
		if err := tasks.EnqueuePhoneVerification(ctx, uc.enqueuer, tasks.PhoneVerificationPayload{
			To: *user.Phone, Code: code,
		}); err != nil {
			return nil, err
		}
		if uc.devMode {
			res.DevPhoneCode = code
		}
	}
	return res, nil
}
