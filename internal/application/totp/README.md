# `internal/application/totp/`

**Cross-cutting capability** — RFC 6238 time-based one-time passwords, implemented
with the standard library only (no external dependency). The auth module uses it
for MFA.

## Constants

`Period` (30s time step), `Digits` (6-digit codes), `SecretKeyLength` (20 bytes
→ 32 base32 characters).

## Functions

- `GenerateSecret() (string, error)` — returns a new random base32 shared secret
  (no padding), suitable for an authenticator app.
- `ProvisioningURI(issuer, account, secret) string` — builds the `otpauth://`
  URI users scan with an authenticator app.
- `Validate(secret, code, window) (bool, error)` — reports whether `code` is a
  valid TOTP for `secret` within the given clock-drift window (in periods);
  `window = 1` accepts one step before and after the current one.

## Usage

```go
secret, _ := totp.GenerateSecret()
uri := totp.ProvisioningURI("MyApp", "user@example.com", secret) // QR for the app

ok, err := totp.Validate(secret, userCode, 1)
if err == nil && ok { /* code accepted */ }
```

## Dependency rules

Vendor-free contract; imports stdlib only.
