package command

import (
	"context"
	"strings"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/application/audit"
	"github.com/fatkulnurk/go-project-starter/internal/application/hash"
	"github.com/fatkulnurk/go-project-starter/internal/application/queue"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/port"
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
	roles          port.Roles
	auditor        audit.Recorder
	clock          clock.Clock
	otpLength      int
	otpTTL         time.Duration
	otpMaxAttempts int
	devMode        bool
}

// NewRegister builds the register use case. roles may be nil when RBAC is not
// wired; the user is then created without any role.
func NewRegister(users domain.UserRepository, codes domain.VerificationCodeRepository, hasher hash.PasswordHasher, enqueuer queue.Enqueuer, roles port.Roles, auditor audit.Recorder, clk clock.Clock, otpLength int, otpTTL time.Duration, otpMaxAttempts int, devMode bool) *Register {
	return &Register{users: users, codes: codes, hasher: hasher, enqueuer: enqueuer, roles: roles, auditor: auditor, clock: clk, otpLength: otpLength, otpTTL: otpTTL, otpMaxAttempts: otpMaxAttempts, devMode: devMode}
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
	user, err := domain.NewUser(cmd.Name, cmd.Email, cmd.Phone, passwordHash, uc.clock.Now())
	if err != nil {
		return nil, err
	}
	if err := uc.users.Save(ctx, user); err != nil {
		return nil, err
	}
	if uc.auditor != nil {
		_ = uc.auditor.Record(ctx, audit.Entry{
			SubjectType: "users",
			SubjectID:   user.ID,
			Action:      audit.ActionCreated,
			NewValues: map[string]any{
				"name":  user.Name,
				"email": user.Email,
				"phone": user.Phone,
			},
			Actor: audit.ActorFrom(ctx),
		})
	}
	if uc.roles != nil {
		if err := uc.roles.AssignDefaultRole(ctx, user.ID); err != nil {
			return nil, err
		}
	}

	res := &RegisterResult{UserID: user.ID}
	if cmd.Email != "" {
		code, err := issueVerificationCode(ctx, uc.codes, uc.enqueuer, user.ID, user.Name, domain.ChannelEmail, *user.Email, uc.otpLength, uc.otpTTL, uc.clock)
		if err != nil {
			return nil, err
		}
		if uc.devMode {
			res.DevEmailCode = code
		}
	}
	if cmd.Phone != "" {
		code, err := issueVerificationCode(ctx, uc.codes, uc.enqueuer, user.ID, user.Name, domain.ChannelPhone, *user.Phone, uc.otpLength, uc.otpTTL, uc.clock)
		if err != nil {
			return nil, err
		}
		if uc.devMode {
			res.DevPhoneCode = code
		}
	}
	return res, nil
}
