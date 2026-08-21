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
// repository type. readDB is the read replica (falls back to writeDB when nil).
type base struct {
	readDB  *sql.DB
	writeDB *sql.DB
	driver  string
	loc     *time.Location
}

func (b base) q(query string) string { return database.Rebind(query, b.driver) }

// r returns the read pool (read replica or primary when no replica is configured).
func (b base) r() *sql.DB {
	if b.readDB != nil {
		return b.readDB
	}
	return b.writeDB
}

// w returns the write pool (always primary).
func (b base) w() *sql.DB { return b.writeDB }

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

// UserRepository implements domain.UserRepository with database/sql queries
// over the users table.
type UserRepository struct{ base }

// NewUserRepository builds a user repository bound to the read/write pools,
// driver and app timezone.
func NewUserRepository(readDB, writeDB *sql.DB, driver string, loc *time.Location) *UserRepository {
	return &UserRepository{base{readDB: readDB, writeDB: writeDB, driver: driver, loc: loc}}
}

const userColumns = `id, name, email, phone, password_hash, email_verified_at, phone_verified_at, totp_secret, totp_confirmed_at, status, created_at, updated_at`

// Save implements domain.UserWriteRepository, inserting the user with all columns
// including the nullable timestamps.
func (r *UserRepository) Save(ctx context.Context, u *domain.User) error {
	_, err := r.w().ExecContext(ctx, r.q(`
		INSERT INTO users (`+userColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		u.ID, u.Name, nullString(u.Email), nullString(u.Phone), u.PasswordHash,
		nullTime(u.EmailVerifiedAt), nullTime(u.PhoneVerifiedAt),
		u.TOTPSecret, nullTime(u.TOTPConfirmedAt),
		string(u.Status), u.CreatedAt, u.UpdatedAt)
	return err
}

// FindByID implements domain.UserReadRepository, returning (nil, nil) when no row
// matches the id.
func (r *UserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	return r.scanUser(ctx, `SELECT `+userColumns+` FROM users WHERE id = ?`, id)
}

// FindByEmail implements domain.UserReadRepository, returning (nil, nil) when no
// row matches the email.
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	return r.scanUser(ctx, `SELECT `+userColumns+` FROM users WHERE email = ?`, email)
}

// FindByPhone implements domain.UserReadRepository, returning (nil, nil) when no
// row matches the phone.
func (r *UserRepository) FindByPhone(ctx context.Context, phone string) (*domain.User, error) {
	return r.scanUser(ctx, `SELECT `+userColumns+` FROM users WHERE phone = ?`, phone)
}

// Update implements domain.UserWriteRepository, overwriting the row's mutable
// columns for the user's id.
func (r *UserRepository) Update(ctx context.Context, u *domain.User) error {
	_, err := r.w().ExecContext(ctx, r.q(`
		UPDATE users
		SET name = ?, email = ?, phone = ?, password_hash = ?, email_verified_at = ?,
		    phone_verified_at = ?, totp_secret = ?, totp_confirmed_at = ?, status = ?, updated_at = ?
		WHERE id = ?`),
		u.Name, nullString(u.Email), nullString(u.Phone), u.PasswordHash,
		nullTime(u.EmailVerifiedAt), nullTime(u.PhoneVerifiedAt),
		u.TOTPSecret, nullTime(u.TOTPConfirmedAt),
		string(u.Status), u.UpdatedAt, u.ID)
	return err
}

func (r *UserRepository) scanUser(ctx context.Context, query string, args ...any) (*domain.User, error) {
	row := r.r().QueryRowContext(ctx, r.q(query), args...)
	var u sqlUser
	err := row.Scan(
		&u.ID, &u.Name, &u.Email, &u.Phone, &u.PasswordHash,
		&u.EmailVerifiedAt, &u.PhoneVerifiedAt, &u.TOTPSecret, &u.TOTPConfirmedAt,
		&u.Status, &u.CreatedAt, &u.UpdatedAt)
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
	TOTPSecret      string
	TOTPConfirmedAt sql.NullTime
	Status          string
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (u sqlUser) toDomain() *domain.User {
	user := &domain.User{
		ID:           u.ID,
		Name:         u.Name,
		PasswordHash: u.PasswordHash,
		TOTPSecret:   u.TOTPSecret,
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
	if u.TOTPConfirmedAt.Valid {
		v := u.TOTPConfirmedAt.Time
		user.TOTPConfirmedAt = &v
	}
	return user
}

// ---------------------------------------------------------------------------
// Refresh tokens
// ---------------------------------------------------------------------------

// RefreshTokenRepository implements domain.RefreshTokenRepository with
// database/sql queries over the refresh_tokens table.
type RefreshTokenRepository struct{ base }

// NewRefreshTokenRepository builds a refresh-token repository bound to the
// read/write pools, driver and app timezone.
func NewRefreshTokenRepository(readDB, writeDB *sql.DB, driver string, loc *time.Location) *RefreshTokenRepository {
	return &RefreshTokenRepository{base{readDB: readDB, writeDB: writeDB, driver: driver, loc: loc}}
}

const refreshColumns = `id, user_id, family_id, jti, token_hash, expires_at, revoked_at, created_at, updated_at`

// Save implements domain.RefreshTokenWriteRepository, inserting the token with its
// hash, family and JTI.
func (r *RefreshTokenRepository) Save(ctx context.Context, t *domain.RefreshToken) error {
	_, err := r.w().ExecContext(ctx, r.q(`
		INSERT INTO refresh_tokens (`+refreshColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		t.ID, t.UserID, t.FamilyID, t.JTI, t.TokenHash, t.ExpiresAt, nullTime(t.RevokedAt), t.CreatedAt, t.UpdatedAt)
	return err
}

// FindByHash implements domain.RefreshTokenReadRepository, returning (nil, nil)
// when no token matches the hash.
func (r *RefreshTokenRepository) FindByHash(ctx context.Context, tokenHash string) (*domain.RefreshToken, error) {
	row := r.r().QueryRowContext(ctx, r.q(`SELECT `+refreshColumns+` FROM refresh_tokens WHERE token_hash = ?`), tokenHash)
	return scanRefreshToken(row)
}

// RevokeByID implements domain.RefreshTokenWriteRepository, revoking the token if
// it is not already revoked.
func (r *RefreshTokenRepository) RevokeByID(ctx context.Context, id string) error {
	now := r.now()
	_, err := r.w().ExecContext(ctx, r.q(`UPDATE refresh_tokens SET revoked_at = ?, updated_at = ? WHERE id = ? AND revoked_at IS NULL`), now, now, id)
	return err
}

// RevokeByIDIfActive implements domain.RefreshTokenWriteRepository, reporting
// whether it actually revoked a still-active token. It is the atomic operation
// backing refresh-token rotation.
func (r *RefreshTokenRepository) RevokeByIDIfActive(ctx context.Context, id string) (bool, error) {
	now := r.now()
	res, err := r.w().ExecContext(ctx, r.q(`UPDATE refresh_tokens SET revoked_at = ?, updated_at = ? WHERE id = ? AND revoked_at IS NULL`), now, now, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// RevokeByUserID implements domain.RefreshTokenWriteRepository, revoking every
// un-revoked token of the user.
func (r *RefreshTokenRepository) RevokeByUserID(ctx context.Context, userID string) error {
	now := r.now()
	_, err := r.w().ExecContext(ctx, r.q(`UPDATE refresh_tokens SET revoked_at = ?, updated_at = ? WHERE user_id = ? AND revoked_at IS NULL`), now, now, userID)
	return err
}

// RevokeFamily implements domain.RefreshTokenWriteRepository, revoking every
// un-revoked token sharing the family id.
func (r *RefreshTokenRepository) RevokeFamily(ctx context.Context, familyID string) error {
	now := r.now()
	_, err := r.w().ExecContext(ctx, r.q(`UPDATE refresh_tokens SET revoked_at = ?, updated_at = ? WHERE family_id = ? AND revoked_at IS NULL`), now, now, familyID)
	return err
}

// JtisByFamily implements domain.RefreshTokenReadRepository, returning the
// access-token ids minted for the family.
func (r *RefreshTokenRepository) JtisByFamily(ctx context.Context, familyID string) ([]string, error) {
	return r.scanStrings(ctx, `SELECT jti FROM refresh_tokens WHERE family_id = ?`, familyID)
}

// JtisByUser implements domain.RefreshTokenReadRepository, returning every
// access-token id minted for the user.
func (r *RefreshTokenRepository) JtisByUser(ctx context.Context, userID string) ([]string, error) {
	return r.scanStrings(ctx, `SELECT jti FROM refresh_tokens WHERE user_id = ?`, userID)
}

// ListActiveFamilies implements domain.RefreshTokenReadRepository, returning the
// user's families that still hold an un-revoked, un-expired token. The
// comparison time is normalized to UTC because expiry timestamps are stored as
// UTC wall-clock.
func (r *RefreshTokenRepository) ListActiveFamilies(ctx context.Context, userID string, now time.Time) ([]domain.RefreshFamily, error) {
	rows, err := r.r().QueryContext(ctx, r.q(`
		SELECT family_id, MIN(created_at), MAX(updated_at), MAX(expires_at)
		FROM refresh_tokens
		WHERE user_id = ? AND revoked_at IS NULL AND expires_at > ?
		GROUP BY family_id
		ORDER BY MAX(created_at) ASC`), userID, now.UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []domain.RefreshFamily
	for rows.Next() {
		var f domain.RefreshFamily
		if err := rows.Scan(&f.ID, &f.CreatedAt, &f.LastUsed, &f.ExpiresAt); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *RefreshTokenRepository) scanStrings(ctx context.Context, query string, args ...any) ([]string, error) {
	rows, err := r.r().QueryContext(ctx, r.q(query), args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanRefreshToken(row *sql.Row) (*domain.RefreshToken, error) {
	var t domain.RefreshToken
	var revokedAt sql.NullTime
	err := row.Scan(&t.ID, &t.UserID, &t.FamilyID, &t.JTI, &t.TokenHash, &t.ExpiresAt, &revokedAt, &t.CreatedAt, &t.UpdatedAt)
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

// VerificationCodeRepository implements domain.VerificationCodeRepository with
// database/sql queries over the verification_codes table.
type VerificationCodeRepository struct{ base }

// NewVerificationCodeRepository builds a verification-code repository bound to
// the read/write pools, driver and app timezone.
func NewVerificationCodeRepository(readDB, writeDB *sql.DB, driver string, loc *time.Location) *VerificationCodeRepository {
	return &VerificationCodeRepository{base{readDB: readDB, writeDB: writeDB, driver: driver, loc: loc}}
}

const codeColumns = `id, user_id, channel, purpose, code_hash, attempts, expires_at, consumed_at, created_at, updated_at`

// Save implements domain.VerificationCodeWriteRepository, inserting the code with
// its channel, purpose and attempt state.
func (r *VerificationCodeRepository) Save(ctx context.Context, c *domain.VerificationCode) error {
	_, err := r.w().ExecContext(ctx, r.q(`
		INSERT INTO verification_codes (`+codeColumns+`)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`),
		c.ID, c.UserID, string(c.Channel), string(c.Purpose), c.CodeHash,
		c.Attempts, c.ExpiresAt, nullTime(c.ConsumedAt), c.CreatedAt, c.UpdatedAt)
	return err
}

// FindLatestActive implements domain.VerificationCodeReadRepository, returning the
// newest matching code or (nil, nil) when none is active. The comparison time
// is normalized to UTC because expiry timestamps are stored as UTC wall-clock.
func (r *VerificationCodeRepository) FindLatestActive(ctx context.Context, userID string, purpose domain.Purpose, channel domain.Channel) (*domain.VerificationCode, error) {
	return r.scanCode(ctx, `
		SELECT `+codeColumns+` FROM verification_codes
		WHERE user_id = ? AND purpose = ? AND channel = ?
		  AND consumed_at IS NULL AND expires_at > ?
		ORDER BY created_at DESC LIMIT 1`,
		userID, string(purpose), string(channel), r.now().UTC())
}

// FindActiveByHash implements domain.VerificationCodeReadRepository, returning the
// newest code matching the hash or (nil, nil) when none is active. The
// comparison time is normalized to UTC (see FindLatestActive).
func (r *VerificationCodeRepository) FindActiveByHash(ctx context.Context, purpose domain.Purpose, codeHash string) (*domain.VerificationCode, error) {
	return r.scanCode(ctx, `
		SELECT `+codeColumns+` FROM verification_codes
		WHERE purpose = ? AND code_hash = ?
		  AND consumed_at IS NULL AND expires_at > ?
		ORDER BY created_at DESC LIMIT 1`,
		string(purpose), codeHash, r.now().UTC())
}

// Consume implements domain.VerificationCodeWriteRepository. It only succeeds when
// the code is still active, so two concurrent requests cannot both win.
func (r *VerificationCodeRepository) Consume(ctx context.Context, id string) (bool, error) {
	now := r.now()
	res, err := r.w().ExecContext(ctx, r.q(`UPDATE verification_codes SET consumed_at = ?, updated_at = ? WHERE id = ? AND consumed_at IS NULL`), now, now, id)
	if err != nil {
		return false, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return false, err
	}
	return n > 0, nil
}

// IncrementAttempts implements domain.VerificationCodeWriteRepository. The counter
// is incremented in SQL (not read-modify-write) and only while it is below
// maxAttempts, so the attempt budget holds under concurrency.
func (r *VerificationCodeRepository) IncrementAttempts(ctx context.Context, id string, maxAttempts int) error {
	now := r.now()
	res, err := r.w().ExecContext(ctx, r.q(`UPDATE verification_codes SET attempts = attempts + 1, updated_at = ? WHERE id = ? AND attempts < ?`), now, id, maxAttempts)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return domain.ErrTooManyAttempts
	}
	return nil
}

// InvalidateByUser implements domain.VerificationCodeWriteRepository, consuming
// every active code of the user/purpose.
func (r *VerificationCodeRepository) InvalidateByUser(ctx context.Context, userID string, purpose domain.Purpose) error {
	now := r.now()
	_, err := r.w().ExecContext(ctx, r.q(`
		UPDATE verification_codes SET consumed_at = ?, updated_at = ?
		WHERE user_id = ? AND purpose = ? AND consumed_at IS NULL`),
		now, now, userID, string(purpose))
	return err
}

// InvalidateByUserChannel implements domain.VerificationCodeWriteRepository,
// consuming every active code of the user/purpose/channel.
func (r *VerificationCodeRepository) InvalidateByUserChannel(ctx context.Context, userID string, purpose domain.Purpose, channel domain.Channel) error {
	now := r.now()
	_, err := r.w().ExecContext(ctx, r.q(`
		UPDATE verification_codes SET consumed_at = ?, updated_at = ?
		WHERE user_id = ? AND purpose = ? AND channel = ? AND consumed_at IS NULL`),
		now, now, userID, string(purpose), string(channel))
	return err
}

func (r *VerificationCodeRepository) scanCode(ctx context.Context, query string, args ...any) (*domain.VerificationCode, error) {
	row := r.r().QueryRowContext(ctx, r.q(query), args...)
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
