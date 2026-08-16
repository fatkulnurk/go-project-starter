# `internal/platform/config/`

**Technical infrastructure** — loads application configuration from environment
variables. It is read once per binary in the composition root, never mid-flow.
Any parse error or validation failure aborts startup with a descriptive message.

## Key functions

- `Load() (Config, error)` — reads the environment and validates the result.
- `Config.Location() *time.Location` — resolves `TimeZone` to a `*time.Location`
  (UTC when empty or unknown).
- `Config.AssetsBaseURLOrDefault() string` — returns `ASSETS_BASE_URL`, falling
  back to `APP_BASE_URL` when empty.

## Key types

`Config` aggregates per-concern settings:

| Type                | Selected by / holds                                    |
|---------------------|--------------------------------------------------------|
| `DatabaseConfig`    | `DB_*` — driver, host, pool sizes, SSL mode            |
| `CacheConfig`/`RedisConfig` | `CACHE_DRIVER`, `REDIS_*`                    |
| `QueueConfig`       | `QUEUE_*` (reuses `REDIS_ADDR`/`REDIS_PASSWORD`, own DB) |
| `StorageConfig`/`LocalStorageConfig`/`S3Config` | `STORAGE_*` |
| `MailConfig`/`SMTPConfig`/`SESConfig` | `MAIL_*`, `SMTP_*`, `SES_*` |
| `SMSConfig`/`TwilioConfig` | `SMS_*`, `TWILIO_*`                      |
| `AuthConfig`        | `AUTH_*`, `OTP_*`, `MAGIC_LINK_*`, `RATE_LIMIT_*`      |
| `RBACConfig`        | `RBAC_CACHE_TTL`                                        |
| `MediaConfig`       | `MEDIA_MAX_UPLOAD_SIZE`                                 |

## Driver-name constants

The platform factories switch on these: `DriverMySQL`, `DriverPostgres`,
`DriverRedis`, `DriverMemory`, `DriverDB`, `DriverLocal`, `DriverS3`,
`DriverLog`, `DriverSMTP`, `DriverSES`, `DriverTwilio`, `DriverAsynq`.

Environment names: `EnvironmentDevelopment` (default), `EnvironmentProduction`.

## Usage

```go
cfg, err := config.Load()
if err != nil { log.Fatal(err) }

db, err := database.New(cfg.Database)
sender, err := mailer.New(cfg.Mail)
```

## Dependency rules

Platform helper; imports stdlib only.
