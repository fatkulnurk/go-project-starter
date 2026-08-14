package commands

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	appid "github.com/fatkulnurk/go-project-starter/internal/application/id"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
)

func TestMain(m *testing.M) {
	appid.SetDefault(testIDGen{})
	os.Exit(m.Run())
}

type testIDGen struct{}

func (testIDGen) New() string { return "id-1" }

type fakeUserRepo struct {
	byEmail map[string]*domain.User
	byPhone map[string]*domain.User
}

func (f *fakeUserRepo) Save(context.Context, *domain.User) error               { return nil }
func (f *fakeUserRepo) Update(context.Context, *domain.User) error             { return nil }
func (f *fakeUserRepo) FindByID(context.Context, string) (*domain.User, error) { return nil, nil }
func (f *fakeUserRepo) FindByEmail(_ context.Context, email string) (*domain.User, error) {
	return f.byEmail[strings.ToLower(email)], nil
}
func (f *fakeUserRepo) FindByPhone(_ context.Context, phone string) (*domain.User, error) {
	return f.byPhone[phone], nil
}

type fakeCodeRepo struct {
	saved []*domain.VerificationCode
}

func (f *fakeCodeRepo) Save(_ context.Context, c *domain.VerificationCode) error {
	f.saved = append(f.saved, c)
	return nil
}
func (f *fakeCodeRepo) FindLatestActive(context.Context, string, domain.Purpose, domain.Channel) (*domain.VerificationCode, error) {
	return nil, nil
}
func (f *fakeCodeRepo) FindActiveByHash(context.Context, domain.Purpose, string) (*domain.VerificationCode, error) {
	return nil, nil
}
func (f *fakeCodeRepo) Consume(context.Context, string) error                { return nil }
func (f *fakeCodeRepo) IncrementAttempts(context.Context, string, int) error { return nil }
func (f *fakeCodeRepo) InvalidateByUser(context.Context, string, domain.Purpose) error {
	return nil
}

func userWith(id, email, phone string) *domain.User {
	var e, p *string
	if email != "" {
		e = &email
	}
	if phone != "" {
		p = &phone
	}
	u := &domain.User{ID: id, Name: "Tester", Email: e, Phone: p}
	return u
}

func TestProcessForgotPasswordSkipsUnknownIdentifier(t *testing.T) {
	users := &fakeUserRepo{byEmail: map[string]*domain.User{}}
	codes := &fakeCodeRepo{}
	uc := NewProcessForgotPassword(users, codes, 6, 15*time.Minute, clock.Fixed{T: time.Now()})

	res, err := uc.Execute(context.Background(), "nobody@example.com")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if res.User != nil {
		t.Fatalf("expected nil user for unknown identifier, got %+v", res.User)
	}
	if len(codes.saved) != 0 {
		t.Fatalf("expected no code saved for unknown identifier, got %d", len(codes.saved))
	}
}

func TestProcessForgotPasswordIssuesCodeForKnownUser(t *testing.T) {
	u := userWith("u1", "someone@example.com", "")
	users := &fakeUserRepo{byEmail: map[string]*domain.User{"someone@example.com": u}}
	codes := &fakeCodeRepo{}
	uc := NewProcessForgotPassword(users, codes, 6, 15*time.Minute, clock.Fixed{T: time.Now()})

	res, err := uc.Execute(context.Background(), "someone@example.com")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if res.User == nil {
		t.Fatal("expected a user for a registered email")
	}
	if len(res.Code) != 6 {
		t.Fatalf("code length = %d, want 6", len(res.Code))
	}
	if len(codes.saved) != 1 {
		t.Fatalf("saved codes = %d, want 1", len(codes.saved))
	}
	if codes.saved[0].Purpose != domain.PurposeReset {
		t.Fatalf("purpose = %q, want reset", codes.saved[0].Purpose)
	}
}

func TestProcessMagicLinkSkipsUnknownEmail(t *testing.T) {
	users := &fakeUserRepo{byEmail: map[string]*domain.User{}}
	codes := &fakeCodeRepo{}
	uc := NewProcessMagicLink(users, codes, "https://app.test", 15*time.Minute, clock.Fixed{T: time.Now()})

	res, err := uc.Execute(context.Background(), "nobody@example.com")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if res.User != nil {
		t.Fatalf("expected nil user for unknown email, got %+v", res.User)
	}
	if len(codes.saved) != 0 {
		t.Fatalf("expected no code saved for unknown email, got %d", len(codes.saved))
	}
}

func TestProcessMagicLinkIssuesLinkForKnownUser(t *testing.T) {
	u := userWith("u1", "someone@example.com", "")
	users := &fakeUserRepo{byEmail: map[string]*domain.User{"someone@example.com": u}}
	codes := &fakeCodeRepo{}
	uc := NewProcessMagicLink(users, codes, "https://app.test/", 15*time.Minute, clock.Fixed{T: time.Now()})

	res, err := uc.Execute(context.Background(), "someone@example.com")
	if err != nil {
		t.Fatalf("Execute error: %v", err)
	}
	if res.User == nil {
		t.Fatal("expected a user for a registered email")
	}
	if !strings.Contains(res.Link, "https://app.test/api/v1/auth/magic-link/verify?token=") {
		t.Fatalf("unexpected link: %q", res.Link)
	}
	if len(codes.saved) != 1 {
		t.Fatalf("saved codes = %d, want 1", len(codes.saved))
	}
	if codes.saved[0].Purpose != domain.PurposeMagicLink {
		t.Fatalf("purpose = %q, want magic_link", codes.saved[0].Purpose)
	}
}
