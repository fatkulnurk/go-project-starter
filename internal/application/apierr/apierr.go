// Package apierr defines the shared API error sentinels. Every module maps its
// errors onto these values (or implements HTTPStatus()/ErrorCode()) so the HTTP
// layer renders one consistent error envelope: {"error":{code,message}}.
//
// The package has no dependencies besides stdlib + the cross-cutting
// application contracts, so domain packages may safely reference the sentinels.
package apierr

import (
	"errors"
	"fmt"

	appauth "github.com/fatkulnurk/go-project-starter/internal/application/auth"
	"github.com/fatkulnurk/go-project-starter/internal/application/authorization"
)

// StatusCoder is implemented by errors that know their HTTP status.
type StatusCoder interface {
	HTTPStatus() int
}

// ErrorCoder is implemented by errors that know their machine-readable code.
type ErrorCoder interface {
	ErrorCode() string
}

// Kind pairs an HTTP status with a stable machine-readable code.
type Kind struct {
	Status int
	Code   string
}

// HTTPStatus implements StatusCoder.
func (k Kind) HTTPStatus() int { return k.Status }

// ErrorCode implements ErrorCoder.
func (k Kind) ErrorCode() string { return k.Code }

// API kinds. The zero case (Unknown) maps to 500 "internal".
var (
	KindUnauthenticated    = Kind{Status: 401, Code: "unauthenticated"}
	KindUnauthorized       = Kind{Status: 401, Code: "unauthorized"}
	KindForbidden          = Kind{Status: 403, Code: "forbidden"}
	KindNotFound           = Kind{Status: 404, Code: "not_found"}
	KindConflict           = Kind{Status: 409, Code: "conflict"}
	KindVerificationNeeded = Kind{Status: 403, Code: "verification_required"}
	KindCodeExpired        = Kind{Status: 410, Code: "code_expired"}
	KindTooManyRequests    = Kind{Status: 429, Code: "too_many_requests"}
	KindPayloadTooLarge    = Kind{Status: 413, Code: "payload_too_large"}
	KindInvalid            = Kind{Status: 422, Code: "invalid"}
	KindInternal           = Kind{Status: 500, Code: "internal"}
)

// Error is a sentinel error with an attached kind.
type Error struct {
	Kind Kind
	Msg  string
}

// New builds a sentinel error for kind.
func New(kind Kind, message string) error {
	return &Error{Kind: kind, Msg: message}
}

// Wrap adds context to an existing error, keeping its kind.
func Wrap(kind Kind, format string, args ...any) error {
	return &Error{Kind: kind, Msg: fmt.Sprintf(format, args...)}
}

// Error implements error.
func (e *Error) Error() string { return e.Msg }

// HTTPStatus implements StatusCoder.
func (e *Error) HTTPStatus() int { return e.Kind.Status }

// ErrorCode implements ErrorCoder.
func (e *Error) ErrorCode() string { return e.Kind.Code }

// Unwrap lets errors.Is match the underlying kind sentinel.
func (e *Error) Unwrap() error { return underlying(e.Kind) }

func underlying(k Kind) error {
	switch k {
	case KindUnauthenticated:
		return appauth.ErrUnauthenticated
	case KindForbidden:
		return authorization.ErrForbidden
	default:
		return nil
	}
}

// KindOf extracts the kind carried by err (or KindInternal when unknown).
func KindOf(err error) Kind {
	if err == nil {
		return KindInternal
	}
	for _, e := range unwrapChain(err) {
		if sc, ok := e.(StatusCoder); ok {
			code := "error"
			if ec, ok2 := e.(ErrorCoder); ok2 {
				code = ec.ErrorCode()
			}
			return Kind{Status: sc.HTTPStatus(), Code: code}
		}
	}
	return KindInternal
}

// Sentinels usable directly by use cases and domain packages.
var (
	ErrUnauthenticated    = New(KindUnauthenticated, "unauthenticated")
	ErrUnauthorized       = New(KindUnauthorized, "unauthorized")
	ErrForbidden          = New(KindForbidden, "forbidden")
	ErrNotFound           = New(KindNotFound, "not found")
	ErrConflict           = New(KindConflict, "conflict")
	ErrVerificationNeeded = New(KindVerificationNeeded, "verification required")
	ErrCodeExpired        = New(KindCodeExpired, "code expired")
	ErrTooManyRequests    = New(KindTooManyRequests, "too many requests")
	ErrPayloadTooLarge    = New(KindPayloadTooLarge, "payload too large")
	ErrInvalid            = New(KindInvalid, "invalid")
	ErrInternal           = New(KindInternal, "internal error")
)

// unwrapChain returns err followed by its chain of wrapped errors.
func unwrapChain(err error) []error {
	var chain []error
	for e := err; e != nil; e = errors.Unwrap(e) {
		chain = append(chain, e)
	}
	return chain
}
