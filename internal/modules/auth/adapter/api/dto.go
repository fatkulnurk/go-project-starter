package api

import (
	"net/http"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/application/query"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth/domain"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
)

// Well-known literals, kept as constants so no magic strings appear in DTOs.
const (
	tokenTypeBearer      = "Bearer"
	responseVerified     = "verified"
	responseLoggedOut    = "logged_out"
	responseReset        = "reset"
	responseUserID       = "user_id"
	responseUser         = "user"
	responseDevEmailCode = "dev_email_code"
	responseDevPhoneCode = "dev_phone_code"
	responseExpiresIn    = "expires_in"
	responseAccessToken  = "access_token"
	responseRefreshToken = "refresh_token"
	responseTokenType    = "token_type"
	responseMFARequired  = "mfa_required"
	responseChallenge    = "challenge"
	responseSecret       = "secret"
	responseProvisioning = "provisioning_uri"
	responseRecovery     = "recovery_codes"
	responseSessions     = "sessions"
	responseTOTPEnabled  = "totp_enabled"
)

// userResponse is the public view of a user account. Roles are owned by the
// RBAC module and returned by profileResponse.
type userResponse struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Email         *string   `json:"email"`
	Phone         *string   `json:"phone"`
	EmailVerified bool      `json:"email_verified"`
	PhoneVerified bool      `json:"phone_verified"`
	Status        string    `json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:            u.ID,
		Name:          u.Name,
		Email:         u.Email,
		Phone:         u.Phone,
		EmailVerified: u.IsEmailVerified(),
		PhoneVerified: u.IsPhoneVerified(),
		Status:        string(u.Status),
		CreatedAt:     u.CreatedAt,
	}
}

// profileResponse extends the user view with the roles and permissions
// resolved from the RBAC module.
type profileResponse struct {
	userResponse
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

func toProfileResponse(p *query.ProfileResult) profileResponse {
	return profileResponse{
		userResponse: toUserResponse(p.User),
		Roles:        p.Roles,
		Permissions:  p.Permissions,
	}
}

type tokenResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	TokenType    string       `json:"token_type"`
	ExpiresIn    int64        `json:"expires_in"`
	User         userResponse `json:"user"`
}

func toTokenResponse(access, refresh string, expiresIn time.Duration, u *domain.User) tokenResponse {
	return tokenResponse{
		AccessToken:  access,
		RefreshToken: refresh,
		TokenType:    tokenTypeBearer,
		ExpiresIn:    int64(expiresIn.Seconds()),
		User:         toUserResponse(u),
	}
}

func clientIP(r *http.Request) string {
	return platformhttp.ClientIP(r)
}
