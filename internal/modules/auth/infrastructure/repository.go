// Package infrastructure implements the auth module's domain repositories
// with database/sql. SQL is written with '?' placeholders and rebound for
// postgres, so the same code runs on MySQL and PostgreSQL.
package infrastructure

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
)

// base carries the shared pool, driver and app timezone used by every
// repository type.
type base struct {
	db     *sql.DB
	driver string
	loc    *time.Location
}

func (b base) q(query string) string { return database.Rebind(query, b.driver) }

// now returns the current time in the app timezone (UTC when unset).
func (b base) now() time.Time {
	if b.loc == nil {
		return time.Now().UTC()
	}
	return time.Now().In(b.loc)
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

// UserRepository implements domain.UserRepository.
type UserRepository struct{ base }

// NewUserRepository builds a user repository.
func NewUserRepository(db *sql.DB, driver string, loc *time.Location) *UserRepository {
	return &UserRepository{base{db: db, driver: driver, loc: loc}}
}

const userColumns = `id, name, email, phone, password_hash, email_verified_at, phone_verified_at, status, created_at, updated_at`

// Save implements domain.UserRepository.
func (r *UserRepository) Save(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx, r.q(`
		INSERT INTO users (`+userColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		u.ID, u.Name, nullString(u.Email), nullString(u.Phone), u.PasswordHash,
		nullTime(u.EmailVerifiedAt), nullTime(u.PhoneVerifiedAt),
		string(u.Status), u.CreatedAt, u.UpdatedAt)
	return err
}

// FindByID implements domain.UserRepository.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return r.scanUser(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id)
}

// FindByEmail implements domain.UserRepository.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanUser(ctx, `SELECT `+userColumns+` FROM users WHERE email = ?`, email)
}

// FindByPhone implements domain.UserRepository.
func (r *UserRepository) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	return r.scanUser(ctx, `SELECT `+userColumns+` FROM users WHERE phone = ?`, phone)
}

// Update implements domain.UserRepository.
func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	_, err := r.db.ExecContext(ctx, r.q(`
		UPDATE users
		SET name = ?, email = ?, phone = ?, password_hash = ?, email_verified_at = ?,
		    phone_verified_at = ?, status = ?, updated_at = ?
		WHERE id = ?`),
		u.Name, nullString(u.Email), nullString(u.Phone), u.PasswordHash,
		nullTime(u.EmailVerifiedAt), nullTime(u.PhoneVerifiedAt),
		string(u.Status), u.UpdatedAt, u.ID)
	return err
}

func (r *UserRepository) scanUser(ctx context.Context, query string, args ...any) (*domain.User, error) {
	row := r.db.QueryRowContext(ctx, r.q(query), args...)
	var u sqlUser
	err := row.Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash,
		&u.EmailVerifiedAt, &u.PhoneVerifiedAt, &u.Status, &u.CreatedAt, &u.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return u.toDomain(), nil
}

type sqlUser struct {
	ID              string
	Name            string
	Email           sql.NullString
	Phone           sql.NullString
	PasswordHash    string
	EmailVerifiedAt sql.NullTime
	PhoneVerifiedAt sql.NullTime
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (u sqlUser) toDomain() *domain.User {
	user := &domain.User{
		ID:           u.ID,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		Status:       domain.UserStatus(u.Status),
		CreatedAt:    u.CreatedAt,
		UpdatedAt:    u.UpdatedAt,
	}
	if u.Email.Valid {
		v := u.Email.String
		user.Email = &v
	}
	if u.Phone.Valid {
		v := u.Phone.String
		user.Phone = &v
	}
	if u.EmailVerifiedAt.Valid {
		v := u.EmailVerifiedAt.Time
		user.EmailVerifiedAt = &v
	}
	if u.PhoneVerifiedAt.Valid {
		v := u.PhoneVerifiedAt.Time
		user.PhoneVerifiedAt = &v
	}
	return user
}

// ---------------------------------------------------------------------------
// Refresh tokens
// ---------------------------------------------------------------------------

// RefreshTokenRepository implements domain.RefreshTokenRepository.
type RefreshTokenRepository struct{ base }

// NewRefreshTokenRepository builds a refresh token repository.
func NewRefreshTokenRepository(db *sql.DB, driver string, loc *time.Location) *RefreshTokenRepository {
	return &RefreshTokenRepository{base{db: db, driver: driver, loc: loc}}
}

const refreshColumns = `id, user_id, token_hash, expires_at, revoked_at, created_at, updated_at`

// Save implements domain.RefreshTokenRepository.
func (r *RefreshTokenRepository) Save(ctx context.Context, t *domain.RefreshToken) error {
	_, err := r.db.ExecContext(ctx, r.q(`
		INSERT INTO refresh_tokens (`+refreshColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?)`),
		t.ID, t.UserID, t.TokenHash, t.ExpiresAt, nullTime(t.RevokedAt), t.CreatedAt, t.UpdatedAt)
	return err
}

// FindByHash implements domain.RefreshTokenRepository.
func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	row := r.db.QueryRowContext(ctx, r.q(`SELECT `+refreshColumns+` FROM refresh_tokens WHERE token_hash = ?`), tokenHash)
	return scanRefreshToken(row)
}

// RevokeByID implements domain.RefreshTokenRepository.
func (r *RefreshTokenRepository) RevokeByID(ctx context.Context, id string) error {
	now := r.now()
	_, err := r.db.ExecContext(ctx, r.q(`UPDATE refresh_tokens SET revoked_at = ?, updated_at = ? WHERE id = ? AND revoked_at IS NULL`), now, now, id)
	return err
}

// RevokeByIDIfActive implements domain.RefreshTokenRepository.
func (r *RefreshTokenRepository) RevokeByIDIfActive(ctx context.Context, id string) (bool, error) {
	now := r.now()
	res, err := r.db.ExecContext(ctx, r.q(`UPDATE refresh_tokens SET revoked_at = ?, updated_at = ? WHERE id = ? AND revoked_at IS NULL`), now, now, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// RevokeByUserID implements domain.RefreshTokenRepository.
func (r *RefreshTokenRepository) RevokeByUserID(ctx context.Context, userID string) error {
	now := r.now()
	_, err := r.db.ExecContext(ctx, r.q(`UPDATE refresh_tokens SET revoked_at = ?, updated_at = ? WHERE user_id = ? AND revoked_at IS NULL`), now, now, userID)
	return err
}

func scanRefreshToken(row *sql.Row) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	var revokedAt sql.NullTime
	err := row.Scan(&t.ID, &t.UserID, &t.TokenHash, &t.ExpiresAt, &revokedAt, &t.CreatedAt, &t.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if revokedAt.Valid {
		v := revokedAt.Time
		t.RevokedAt = &v
	}
	return &t, nil
}

// ---------------------------------------------------------------------------
// Verification codes / magic links
// ---------------------------------------------------------------------------

// VerificationCodeRepository implements domain.VerificationCodeRepository.
type VerificationCodeRepository struct{ base }

// NewVerificationCodeRepository builds a verification code repository.
func NewVerificationCodeRepository(db *sql.DB, driver string, loc *time.Location) *VerificationCodeRepository {
	return &VerificationCodeRepository{base{db: db, driver: driver, loc: loc}}
}

const codeColumns = `id, user_id, channel, purpose, code_hash, attempts, expires_at, consumed_at, created_at, updated_at`

// Save implements domain.VerificationCodeRepository.
func (r *VerificationCodeRepository) Save(ctx context.Context, c *domain.VerificationCode) error {
	_, err := r.db.ExecContext(ctx, r.q(`
		INSERT INTO verification_codes (`+codeColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		c.ID, c.UserID, string(c.Channel), string(c.Purpose), c.CodeHash,
		c.Attempts, c.ExpiresAt, nullTime(c.ConsumedAt), c.CreatedAt, c.UpdatedAt)
	return err
}

// FindLatestActive implements domain.VerificationCodeRepository.
func (r *VerificationCodeRepository) FindLatestActive(ctx context.Context, userID string, purpose domain.Purpose, channel domain.Channel) (*domain.VerificationCode, error) {
	return r.scanCode(ctx, `
		SELECT `+codeColumns+` FROM verification_codes
		WHERE user_id = ? AND purpose = ? AND channel = ?
		  AND consumed_at IS NULL AND expires_at > ?
		ORDER BY created_at DESC LIMIT 1`,
		userID, string(purpose), string(channel), r.now())
}

// FindActiveByHash implements domain.VerificationCodeRepository.
func (r *VerificationCodeRepository) FindActiveByHash(ctx context.Context, purpose domain.Purpose, codeHash string) (*domain.VerificationCode, error) {
	return r.scanCode(ctx, `
		SELECT `+codeColumns+` FROM verification_codes
		WHERE purpose = ? AND code_hash = ?
		  AND consumed_at IS NULL AND expires_at > ?
		ORDER BY created_at DESC LIMIT 1`,
		string(purpose), codeHash, r.now())
}

// Consume implements domain.VerificationCodeRepository.
func (r *VerificationCodeRepository) Consume(ctx context.Context, id string) error {
	now := r.now()
	_, err := r.db.ExecContext(ctx, r.q(`UPDATE verification_codes SET consumed_at = ?, updated_at = ? WHERE id = ? AND consumed_at IS NULL`), now, now, id)
	return err
}

// IncrementAttempts implements domain.VerificationCodeRepository.
func (r *VerificationCodeRepository) IncrementAttempts(ctx context.Context, id string, attempts int) error {
	_, err := r.db.ExecContext(ctx, r.q(`UPDATE verification_codes SET attempts = ?, updated_at = ? WHERE id = ?`), attempts, r.now(), id)
	return err
}

// InvalidateByUser implements domain.VerificationCodeRepository.
func (r *VerificationCodeRepository) InvalidateByUser(ctx context.Context, userID string, purpose domain.Purpose) error {
	now := r.now()
	_, err := r.db.ExecContext(ctx, r.q(`
		UPDATE verification_codes SET consumed_at = ?, updated_at = ?
		WHERE user_id = ? AND purpose = ? AND consumed_at IS NULL`),
		now, now, userID, string(purpose))
	return err
}

func (r *VerificationCodeRepository) scanCode(ctx context.Context, query string, args ...any) (*domain.VerificationCode, error) {
	row := r.db.QueryRowContext(ctx, r.q(query), args...)
	var c domain.VerificationCode
	var consumedAt sql.NullTime
	var channel, purpose string
	err := row.Scan(&c.ID, &c.UserID, &channel, &purpose, &c.CodeHash,
		&c.Attempts, &c.ExpiresAt, &consumedAt, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	c.Channel = domain.Channel(channel)
	c.Purpose = domain.Purpose(purpose)
	if consumedAt.Valid {
		v := consumedAt.Time
		c.ConsumedAt = &v
	}
	return &c, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func nullString(v *string) sql.NullString {
	if v == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *v, Valid: true}
}

func nullTime(v *time.Time) sql.NullTime {
	if v == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *v, Valid: true}
}
