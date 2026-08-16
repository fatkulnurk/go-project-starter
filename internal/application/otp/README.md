# `internal/application/otp/`

**Cross-cutting capability** — a shared generator for one-time passcodes. A
domain-neutral technical helper (not business logic), usable by any module.

## Functions

- `Generate(n int) (string, error)` — returns a numeric OTP of `n` digits
  (`n < 1` falls back to 6). The code is not cryptographically strong enough for
  authentication on its own — pair it with hashing, TTL and attempt limits, and
  always deliver it over a trusted channel.

## Usage

```go
code, err := otp.Generate(cfg.Auth.OTPLength) // e.g. "483920"
```

For time-based (RFC 6238) codes used by authenticator apps, see
`internal/application/totp`.

## Dependency rules

Vendor-free contract; imports stdlib only.
