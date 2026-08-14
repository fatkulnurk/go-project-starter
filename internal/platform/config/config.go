// Package config loads application configuration from environment variables.
// It is read once per binary in the composition root, never mid-flow.
package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// Environment names.
const (
	EnvironmentDevelopment = "development"
	EnvironmentProduction  = "production"
)

// Cache, storage, mail, sms, queue and database driver names. Each factory in
// internal/platform switches on these constants.
const (
	DriverMySQL    = "mysql"
	DriverPostgres = "postgres"
	DriverRedis    = "redis"
	DriverMemory   = "memory"
	DriverLocal    = "local"
	DriverS3       = "s3"
	DriverLog      = "log"
	DriverSMTP     = "smtp"
	DriverSES      = "ses"
	DriverTwilio   = "twilio"
	DriverAsynq    = "asynq"
)

// Environment variable keys.
const (
	envAppEnv     = "APP_ENV"
	envAppPort    = "APP_PORT"
	envWebPort    = "WEB_PORT"
	envAppBaseURL = "APP_BASE_URL"
	envAppName    = "APP_NAME"

	envDBDriver   = "DB_DRIVER"
	envDBHost     = "DB_HOST"
	envDBPort     = "DB_PORT"
	envDBUser     = "DB_USER"
	envDBPassword = "DB_PASSWORD"
	envDBName     = "DB_NAME"

	envDBMaxOpenConns    = "DB_MAX_OPEN_CONNS"
	envDBMaxIdleConns    = "DB_MAX_IDLE_CONNS"
	envDBConnMaxLifetime = "DB_CONN_MAX_LIFETIME"
	envDBConnMaxIdleTime = "DB_CONN_MAX_IDLE_TIME"
	envDBSSLMode         = "DB_SSL_MODE"

	envCacheDriver       = "CACHE_DRIVER"
	envRedisAddr         = "REDIS_ADDR"
	envRedisPassword     = "REDIS_PASSWORD"
	envRedisDB           = "REDIS_DB"
	envRedisPoolSize     = "REDIS_POOL_SIZE"
	envRedisMinIdle      = "REDIS_MIN_IDLE_CONNS"
	envRedisDialTimeout  = "REDIS_DIAL_TIMEOUT"
	envRedisReadTimeout  = "REDIS_READ_TIMEOUT"
	envRedisWriteTimeout = "REDIS_WRITE_TIMEOUT"

	envQueueDriver      = "QUEUE_DRIVER"
	envQueueConcurrency = "QUEUE_CONCURRENCY"
	envQueueRedisDB     = "QUEUE_REDIS_DB"
	envQueueRedisPool   = "QUEUE_REDIS_POOL_SIZE"

	envStorageDriver      = "STORAGE_DRIVER"
	envStorageLocalDir    = "STORAGE_LOCAL_DIR"
	envStorageS3Endpoint  = "STORAGE_S3_ENDPOINT"
	envStorageS3Region    = "STORAGE_S3_REGION"
	envStorageS3Bucket    = "STORAGE_S3_BUCKET"
	envStorageS3AccessKey = "STORAGE_S3_ACCESS_KEY"
	envStorageS3SecretKey = "STORAGE_S3_SECRET_KEY"
	envStorageS3PathStyle = "STORAGE_S3_PATH_STYLE"

	envMailDriver   = "MAIL_DRIVER"
	envMailFrom     = "MAIL_FROM"
	envMailFromName = "MAIL_FROM_NAME"
	envSMTPHost     = "SMTP_HOST"
	envSMTPPort     = "SMTP_PORT"
	envSMTPUser     = "SMTP_USER"
	envSMTPPassword = "SMTP_PASSWORD"
	envSMTPPoolSize = "MAIL_SMTP_POOL_SIZE"
	envSMTPSSL      = "MAIL_SMTP_SSL"
	envSESRegion    = "SES_REGION"
	envSESAccessKey = "SES_ACCESS_KEY"
	envSESSecretKey = "SES_SECRET_KEY"

	envSMSDriver     = "SMS_DRIVER"
	envSMSFrom       = "SMS_FROM"
	envTwilioAccount = "TWILIO_ACCOUNT_SID"
	envTwilioAuth    = "TWILIO_AUTH_TOKEN"
	envTwilioMessSID = "TWILIO_MESSAGE_SID"

	envAuthJWTSecret            = "AUTH_JWT_SECRET"
	envAuthJWTIssuer            = "AUTH_JWT_ISSUER"
	envAuthJWTAudience          = "AUTH_JWT_AUDIENCE"
	envAuthAccessTokenTTL       = "AUTH_ACCESS_TOKEN_TTL"
	envAuthRefreshTokenTTL      = "AUTH_REFRESH_TOKEN_TTL"
	envAuthRequireEmailVerified = "AUTH_REQUIRE_EMAIL_VERIFIED"
	envOTPLength                = "OTP_LENGTH"
	envOTPTTL                   = "OTP_TTL"
	envMagicLinkTTL             = "MAGIC_LINK_TTL"
	envOTPMaxAttempts           = "OTP_MAX_ATTEMPTS"
	envRateLimitLoginMax        = "RATE_LIMIT_LOGIN_MAX"
	envRateLimitLoginWindow     = "RATE_LIMIT_LOGIN_WINDOW"
	envRateLimitPublicMax       = "RATE_LIMIT_PUBLIC_MAX"
	envRateLimitPublicWindow    = "RATE_LIMIT_PUBLIC_WINDOW"

	envRBACCacheTTL       = "RBAC_CACHE_TTL"
	envRBACSuperAdminMail = "RBAC_BOOTSTRAP_SUPER_ADMIN_EMAIL"

	envMediaMaxUploadSize = "MEDIA_MAX_UPLOAD_SIZE"

	envTrustedProxies    = "TRUSTED_PROXIES"
	envCORSAllowedOrigin = "CORS_ALLOWED_ORIGINS"
)

// Defaults. Nothing is required at runtime except AUTH_JWT_SECRET in
// production; everything else degrades to a sensible dev value.
const (
	defaultEnvironment           = EnvironmentDevelopment
	defaultPort                  = 8080
	defaultWebPort               = 8081
	defaultBaseURL               = "http://localhost:8080"
	defaultAppName               = "Go Project Starter"
	defaultDBDriver              = DriverMySQL
	defaultDBHost                = "localhost"
	defaultDBPort                = 3306
	defaultDBUser                = "root"
	defaultDBName                = "go_project_starter"
	defaultDBMaxOpenConns        = 25
	defaultDBMaxIdleConns        = 5
	defaultDBConnMaxLifetime     = 5 * time.Minute
	defaultDBConnMaxIdleTime     = 5 * time.Minute
	defaultDBSSLMode             = "disable"
	defaultCacheDriver           = DriverRedis
	defaultRedisAddr             = "localhost:6379"
	defaultRedisDB               = 0
	defaultRedisPoolSize         = 10
	defaultRedisMinIdle          = 1
	defaultRedisDialTimeout      = 5 * time.Second
	defaultRedisReadTimeout      = 3 * time.Second
	defaultRedisWriteTimeout     = 3 * time.Second
	defaultQueueDriver           = DriverAsynq
	defaultQueueConcurrency      = 10
	defaultQueueRedisDB          = 1
	defaultQueueRedisPool        = 10
	defaultStorageDriver         = DriverLocal
	defaultStorageLocalDir       = "./storage"
	defaultS3Region              = "us-east-1"
	defaultMailDriver            = DriverLog
	defaultMailFrom              = "no-reply@example.com"
	defaultMailFromName          = "Go Project Starter"
	defaultSMTPHost              = "smtp.example.com"
	defaultSMTPPort              = 587
	defaultSMTPPoolSize          = 5
	defaultSMTPSSL               = "starttls"
	defaultSESRegion             = "us-east-1"
	defaultSMSDriver             = DriverLog
	defaultJWTSecret             = "change-me-in-production"
	defaultJWTIssuer             = "go-project-starter"
	defaultJWTAudience           = "go-project-starter-api"
	defaultAccessTokenTTL        = 15 * time.Minute
	defaultRefreshTokenTTL       = 720 * time.Hour
	defaultOTPLength             = 6
	defaultOTPTTL                = 15 * time.Minute
	defaultMagicLinkTTL          = 15 * time.Minute
	defaultOTPMaxAttempts        = 5
	defaultRateLimitLoginMax     = 5
	defaultRateLimitWindow       = 15 * time.Minute
	defaultRateLimitPublicMax    = 60
	defaultRateLimitPublicWindow = 1 * time.Minute
	defaultRBACCacheTTL          = 5 * time.Minute
	defaultMediaMaxUploadSize    = 10 << 20 // 10 MiB
)

// Config is the union of all settings needed by every binary.
type Config struct {
	Environment string
	Port        int
	// WebPort serves the public web (homepage) module. Kept separate from
	// Port so the API and the web front can run independently.
	WebPort int
	BaseURL string
	AppName string

	// TrustedProxies lists CIDRs/IPs whose X-Forwarded-For header is trusted.
	TrustedProxies []string
	// CORSAllowedOrigins lists origins allowed to call the API. Empty means
	// same-origin only (no CORS headers emitted).
	CORSAllowedOrigins []string

	Database DatabaseConfig
	Cache    CacheConfig
	Queue    QueueConfig
	Storage  StorageConfig
	Mail     MailConfig
	SMS      SMSConfig
	Auth     AuthConfig
	RBAC     RBACConfig
	Media    MediaConfig
}

// MediaConfig holds upload constraints.
type MediaConfig struct {
	MaxUploadSize int64
}

// RBACConfig holds role/permission bootstrap and caching settings.
type RBACConfig struct {
	PermissionCacheTTL time.Duration
	SuperAdminEmail    string
}

// DatabaseConfig selects the SQL driver and connection. Pool sizes control
// how many connections database/sql keeps open/idle.
type DatabaseConfig struct {
	Driver          string
	Host            string
	Port            int
	User            string
	Password        string
	Name            string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	// SSLMode controls transport encryption: "disable", "require",
	// "verify-ca", "verify-full" (postgres) or "disable", "require",
	// "skip-verify" (mysql).
	SSLMode string
}

// CacheConfig selects the cache driver.
type CacheConfig struct {
	Driver string
	Redis  RedisConfig
}

// RedisConfig holds go-redis connection settings.
type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

// QueueConfig selects the queue backend. RedisAddr/Password reuse the cache
// Redis settings; DB is kept separate from the cache so both can share one
// Redis server without key collisions.
type QueueConfig struct {
	Driver        string
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	RedisPoolSize int
	Concurrency   int
}

// StorageConfig selects the storage driver.
type StorageConfig struct {
	Driver string
	Local  LocalStorageConfig
	S3     S3Config
}

// LocalStorageConfig points at the root directory for the local driver.
type LocalStorageConfig struct {
	Dir string
}

// S3Config holds AWS S3 / S3-compatible settings.
type S3Config struct {
	Endpoint     string
	Region       string
	Bucket       string
	AccessKey    string
	SecretKey    string
	UsePathStyle bool
}

// MailConfig selects the mail driver.
type MailConfig struct {
	Driver   string
	From     string
	FromName string
	SMTP     SMTPConfig
	SES      SESConfig
}

// SMTPConfig holds plain SMTP settings.
type SMTPConfig struct {
	Host     string
	Port     int
	User     string
	Password string
	PoolSize int
	// SSL is one of "none", "tls", "starttls". Defaults to "starttls".
	SSL string
}

// SESConfig holds Amazon SES (SESv2) settings.
type SESConfig struct {
	Region    string
	AccessKey string
	SecretKey string
}

// SMSConfig selects the SMS driver.
type SMSConfig struct {
	Driver string
	From   string
	Twilio TwilioConfig
}

// TwilioConfig holds Twilio settings.
type TwilioConfig struct {
	AccountSID   string
	AuthToken    string
	MessagingSID string
}

// AuthConfig holds token, verification and rate-limit settings.
type AuthConfig struct {
	JWTSecret             string
	JWTIssuer             string
	JWTAudience           string
	AccessTokenTTL        time.Duration
	RefreshTokenTTL       time.Duration
	RequireEmailVerified  bool
	OTPLength             int
	OTPTTL                time.Duration
	OTPMaxAttempts        int
	MagicLinkTTL          time.Duration
	RateLimitLoginMax     int64
	RateLimitLoginWindow  time.Duration
	RateLimitPublicMax    int64
	RateLimitPublicWindow time.Duration
}

// builder accumulates env values and reports any parse failures.
type builder struct {
	errs []string
}

func (b *builder) str(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func (b *builder) bool(key string) bool { return os.Getenv(key) == "true" }

func (b *builder) int(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		b.errs = append(b.errs, fmt.Sprintf("%s must be an integer, got %q", key, v))
		return fallback
	}
	return n
}

func (b *builder) int64(key string, fallback int64) int64 {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		b.errs = append(b.errs, fmt.Sprintf("%s must be an integer, got %q", key, v))
		return fallback
	}
	return n
}

func (b *builder) duration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		b.errs = append(b.errs, fmt.Sprintf("%s must be a duration (e.g. 15m, 1h), got %q", key, v))
		return fallback
	}
	return d
}

func (b *builder) list(key string) []string {
	raw := os.Getenv(key)
	if raw == "" {
		return nil
	}
	var out []string
	for _, part := range strings.Split(raw, ",") {
		if p := strings.TrimSpace(part); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// Load reads configuration from the environment. Any parse error or
// validation failure aborts startup with a descriptive message.
func Load() (Config, error) {
	b := &builder{}
	cfg := Config{
		Environment: b.str(envAppEnv, defaultEnvironment),
		Port:        b.int(envAppPort, defaultPort),
		WebPort:     b.int(envWebPort, defaultWebPort),
		BaseURL:     b.str(envAppBaseURL, defaultBaseURL),
		AppName:     b.str(envAppName, defaultAppName),

		TrustedProxies:     b.list(envTrustedProxies),
		CORSAllowedOrigins: b.list(envCORSAllowedOrigin),

		Database: DatabaseConfig{
			Driver:          b.str(envDBDriver, defaultDBDriver),
			Host:            b.str(envDBHost, defaultDBHost),
			Port:            b.int(envDBPort, defaultDBPort),
			User:            b.str(envDBUser, defaultDBUser),
			Password:        b.str(envDBPassword, ""),
			Name:            b.str(envDBName, defaultDBName),
			MaxOpenConns:    b.int(envDBMaxOpenConns, defaultDBMaxOpenConns),
			MaxIdleConns:    b.int(envDBMaxIdleConns, defaultDBMaxIdleConns),
			ConnMaxLifetime: b.duration(envDBConnMaxLifetime, defaultDBConnMaxLifetime),
			ConnMaxIdleTime: b.duration(envDBConnMaxIdleTime, defaultDBConnMaxIdleTime),
			SSLMode:         b.str(envDBSSLMode, defaultDBSSLMode),
		},
		Cache: CacheConfig{
			Driver: b.str(envCacheDriver, defaultCacheDriver),
			Redis: RedisConfig{
				Addr:         b.str(envRedisAddr, defaultRedisAddr),
				Password:     b.str(envRedisPassword, ""),
				DB:           b.int(envRedisDB, defaultRedisDB),
				PoolSize:     b.int(envRedisPoolSize, defaultRedisPoolSize),
				MinIdleConns: b.int(envRedisMinIdle, defaultRedisMinIdle),
				DialTimeout:  b.duration(envRedisDialTimeout, defaultRedisDialTimeout),
				ReadTimeout:  b.duration(envRedisReadTimeout, defaultRedisReadTimeout),
				WriteTimeout: b.duration(envRedisWriteTimeout, defaultRedisWriteTimeout),
			},
		},
		Queue: QueueConfig{
			Driver:        b.str(envQueueDriver, defaultQueueDriver),
			RedisAddr:     b.str(envRedisAddr, defaultRedisAddr),
			RedisPassword: b.str(envRedisPassword, ""),
			RedisDB:       b.int(envQueueRedisDB, defaultQueueRedisDB),
			RedisPoolSize: b.int(envQueueRedisPool, defaultQueueRedisPool),
			Concurrency:   b.int(envQueueConcurrency, defaultQueueConcurrency),
		},
		Storage: StorageConfig{
			Driver: b.str(envStorageDriver, defaultStorageDriver),
			Local:  LocalStorageConfig{Dir: b.str(envStorageLocalDir, defaultStorageLocalDir)},
			S3: S3Config{
				Endpoint:     b.str(envStorageS3Endpoint, ""),
				Region:       b.str(envStorageS3Region, defaultS3Region),
				Bucket:       b.str(envStorageS3Bucket, ""),
				AccessKey:    b.str(envStorageS3AccessKey, ""),
				SecretKey:    b.str(envStorageS3SecretKey, ""),
				UsePathStyle: b.bool(envStorageS3PathStyle),
			},
		},
		Mail: MailConfig{
			Driver:   b.str(envMailDriver, defaultMailDriver),
			From:     b.str(envMailFrom, defaultMailFrom),
			FromName: b.str(envMailFromName, defaultMailFromName),
			SMTP: SMTPConfig{
				Host:     b.str(envSMTPHost, defaultSMTPHost),
				Port:     b.int(envSMTPPort, defaultSMTPPort),
				User:     b.str(envSMTPUser, ""),
				Password: b.str(envSMTPPassword, ""),
				PoolSize: b.int(envSMTPPoolSize, defaultSMTPPoolSize),
				SSL:      b.str(envSMTPSSL, defaultSMTPSSL),
			},
			SES: SESConfig{
				Region:    b.str(envSESRegion, defaultSESRegion),
				AccessKey: b.str(envSESAccessKey, ""),
				SecretKey: b.str(envSESSecretKey, ""),
			},
		},
		SMS: SMSConfig{
			Driver: b.str(envSMSDriver, defaultSMSDriver),
			From:   b.str(envSMSFrom, ""),
			Twilio: TwilioConfig{
				AccountSID:   b.str(envTwilioAccount, ""),
				AuthToken:    b.str(envTwilioAuth, ""),
				MessagingSID: b.str(envTwilioMessSID, ""),
			},
		},
		Auth: AuthConfig{
			JWTSecret:             b.str(envAuthJWTSecret, defaultJWTSecret),
			JWTIssuer:             b.str(envAuthJWTIssuer, defaultJWTIssuer),
			JWTAudience:           b.str(envAuthJWTAudience, defaultJWTAudience),
			AccessTokenTTL:        b.duration(envAuthAccessTokenTTL, defaultAccessTokenTTL),
			RefreshTokenTTL:       b.duration(envAuthRefreshTokenTTL, defaultRefreshTokenTTL),
			RequireEmailVerified:  b.bool(envAuthRequireEmailVerified),
			OTPLength:             b.int(envOTPLength, defaultOTPLength),
			OTPTTL:                b.duration(envOTPTTL, defaultOTPTTL),
			OTPMaxAttempts:        b.int(envOTPMaxAttempts, defaultOTPMaxAttempts),
			MagicLinkTTL:          b.duration(envMagicLinkTTL, defaultMagicLinkTTL),
			RateLimitLoginMax:     b.int64(envRateLimitLoginMax, defaultRateLimitLoginMax),
			RateLimitLoginWindow:  b.duration(envRateLimitLoginWindow, defaultRateLimitWindow),
			RateLimitPublicMax:    b.int64(envRateLimitPublicMax, defaultRateLimitPublicMax),
			RateLimitPublicWindow: b.duration(envRateLimitPublicWindow, defaultRateLimitPublicWindow),
		},
		RBAC: RBACConfig{
			PermissionCacheTTL: b.duration(envRBACCacheTTL, defaultRBACCacheTTL),
			SuperAdminEmail:    b.str(envRBACSuperAdminMail, ""),
		},
		Media: MediaConfig{
			MaxUploadSize: b.int64(envMediaMaxUploadSize, defaultMediaMaxUploadSize),
		},
	}

	b.errs = append(b.errs, cfg.validate()...)
	if len(b.errs) > 0 {
		return Config{}, errors.New(strings.Join(b.errs, "; "))
	}
	return cfg, nil
}

// validate returns a list of configuration problems, empty when valid.
func (c Config) validate() []string {
	var errs []string

	switch c.Environment {
	case EnvironmentDevelopment, EnvironmentProduction:
	default:
		errs = append(errs, "APP_ENV must be 'development' or 'production'")
	}
	if c.Port < 1 || c.Port > 65535 {
		errs = append(errs, fmt.Sprintf("APP_PORT must be between 1 and 65535, got %d", c.Port))
	}
	if c.WebPort < 1 || c.WebPort > 65535 {
		errs = append(errs, fmt.Sprintf("WEB_PORT must be between 1 and 65535, got %d", c.WebPort))
	}
	if _, err := url.ParseRequestURI(c.BaseURL); err != nil || !strings.Contains(c.BaseURL, "://") {
		errs = append(errs, fmt.Sprintf("APP_BASE_URL must be an absolute URL, got %q", c.BaseURL))
	}

	switch c.Database.Driver {
	case DriverMySQL, DriverPostgres:
	default:
		errs = append(errs, "DB_DRIVER must be 'mysql' or 'postgres'")
	}
	if c.Database.MaxOpenConns < 1 {
		errs = append(errs, fmt.Sprintf("DB_MAX_OPEN_CONNS must be >= 1, got %d", c.Database.MaxOpenConns))
	}
	if c.Database.MaxIdleConns < 0 || c.Database.MaxIdleConns > c.Database.MaxOpenConns {
		errs = append(errs, "DB_MAX_IDLE_CONNS must be between 0 and DB_MAX_OPEN_CONNS")
	}
	if c.Database.ConnMaxLifetime < 0 || c.Database.ConnMaxIdleTime < 0 {
		errs = append(errs, "DB_CONN_MAX_LIFETIME / DB_CONN_MAX_IDLE_TIME must not be negative")
	}
	if !validSSLMode(c.Database.Driver, c.Database.SSLMode) {
		errs = append(errs, fmt.Sprintf("DB_SSL_MODE=%q is invalid for %s", c.Database.SSLMode, c.Database.Driver))
	}

	switch c.Cache.Driver {
	case DriverRedis, DriverMemory:
	default:
		errs = append(errs, "CACHE_DRIVER must be 'redis' or 'memory'")
	}
	if c.Cache.Redis.PoolSize < 1 {
		errs = append(errs, "REDIS_POOL_SIZE must be >= 1")
	}
	if c.Cache.Redis.DialTimeout <= 0 || c.Cache.Redis.ReadTimeout <= 0 || c.Cache.Redis.WriteTimeout <= 0 {
		errs = append(errs, "REDIS_DIAL_TIMEOUT / REDIS_READ_TIMEOUT / REDIS_WRITE_TIMEOUT must be positive")
	}

	switch c.Queue.Driver {
	case DriverAsynq:
	default:
		errs = append(errs, "QUEUE_DRIVER must be 'asynq'")
	}
	if c.Queue.Concurrency < 1 {
		errs = append(errs, "QUEUE_CONCURRENCY must be >= 1")
	}
	if c.Queue.RedisPoolSize < 1 {
		errs = append(errs, "QUEUE_REDIS_POOL_SIZE must be >= 1")
	}

	switch c.Storage.Driver {
	case DriverLocal, DriverS3:
	default:
		errs = append(errs, "STORAGE_DRIVER must be 'local' or 's3'")
	}
	if c.Storage.Driver == DriverS3 {
		if c.Storage.S3.Bucket == "" {
			errs = append(errs, "STORAGE_S3_BUCKET is required when STORAGE_DRIVER=s3")
		}
		if c.Storage.S3.Region == "" {
			errs = append(errs, "STORAGE_S3_REGION is required when STORAGE_DRIVER=s3")
		}
		if c.Storage.S3.AccessKey == "" || c.Storage.S3.SecretKey == "" {
			errs = append(errs, "STORAGE_S3_ACCESS_KEY and STORAGE_S3_SECRET_KEY are required when STORAGE_DRIVER=s3")
		}
	}

	switch c.Mail.Driver {
	case DriverLog, DriverSMTP, DriverSES:
	default:
		errs = append(errs, "MAIL_DRIVER must be 'log', 'smtp' or 'ses'")
	}
	if c.Mail.Driver == DriverSMTP {
		if c.Mail.SMTP.Host == "" {
			errs = append(errs, "SMTP_HOST is required when MAIL_DRIVER=smtp")
		}
		switch c.Mail.SMTP.SSL {
		case "none", "tls", "starttls":
		default:
			errs = append(errs, "MAIL_SMTP_SSL must be 'none', 'tls' or 'starttls'")
		}
		if c.Mail.SMTP.PoolSize < 1 {
			errs = append(errs, "MAIL_SMTP_POOL_SIZE must be >= 1")
		}
	}
	if c.Mail.Driver == DriverSES && (c.Mail.SES.Region == "" || c.Mail.SES.AccessKey == "" || c.Mail.SES.SecretKey == "") {
		errs = append(errs, "SES_REGION, SES_ACCESS_KEY and SES_SECRET_KEY are required when MAIL_DRIVER=ses")
	}

	switch c.SMS.Driver {
	case DriverLog, DriverTwilio:
	default:
		errs = append(errs, "SMS_DRIVER must be 'log' or 'twilio'")
	}
	if c.SMS.Driver == DriverTwilio {
		if c.SMS.Twilio.AccountSID == "" || c.SMS.Twilio.AuthToken == "" || c.SMS.Twilio.MessagingSID == "" {
			errs = append(errs, "TWILIO_ACCOUNT_SID, TWILIO_AUTH_TOKEN and TWILIO_MESSAGE_SID are required when SMS_DRIVER=twilio")
		}
	}

	if c.Auth.JWTSecret == "" {
		errs = append(errs, "AUTH_JWT_SECRET must not be empty")
	}
	if c.Auth.JWTSecret == defaultJWTSecret && c.Environment == EnvironmentProduction {
		errs = append(errs, "AUTH_JWT_SECRET must be changed from the default in production")
	}
	if c.Auth.JWTSecret != defaultJWTSecret && len(c.Auth.JWTSecret) < 32 {
		errs = append(errs, "AUTH_JWT_SECRET should be at least 32 characters")
	}
	if c.Auth.JWTIssuer == "" || c.Auth.JWTAudience == "" {
		errs = append(errs, "AUTH_JWT_ISSUER and AUTH_JWT_AUDIENCE must not be empty")
	}
	if c.Auth.AccessTokenTTL <= 0 || c.Auth.RefreshTokenTTL <= 0 {
		errs = append(errs, "AUTH_ACCESS_TOKEN_TTL and AUTH_REFRESH_TOKEN_TTL must be positive")
	}
	if c.Auth.RefreshTokenTTL <= c.Auth.AccessTokenTTL {
		errs = append(errs, "AUTH_REFRESH_TOKEN_TTL must be greater than AUTH_ACCESS_TOKEN_TTL")
	}
	if c.Auth.OTPLength < 4 || c.Auth.OTPLength > 10 {
		errs = append(errs, "OTP_LENGTH must be between 4 and 10")
	}
	if c.Auth.OTPTTL <= 0 || c.Auth.MagicLinkTTL <= 0 {
		errs = append(errs, "OTP_TTL and MAGIC_LINK_TTL must be positive")
	}
	if c.Auth.OTPMaxAttempts < 1 {
		errs = append(errs, "OTP_MAX_ATTEMPTS must be >= 1")
	}
	if c.Auth.RateLimitLoginMax < 1 || c.Auth.RateLimitLoginWindow <= 0 {
		errs = append(errs, "RATE_LIMIT_LOGIN_MAX / RATE_LIMIT_LOGIN_WINDOW must be positive")
	}
	if c.Auth.RateLimitPublicMax < 1 || c.Auth.RateLimitPublicWindow <= 0 {
		errs = append(errs, "RATE_LIMIT_PUBLIC_MAX / RATE_LIMIT_PUBLIC_WINDOW must be positive")
	}

	if c.RBAC.PermissionCacheTTL <= 0 {
		errs = append(errs, "RBAC_CACHE_TTL must be positive")
	}

	if c.Media.MaxUploadSize < 1 {
		errs = append(errs, "MEDIA_MAX_UPLOAD_SIZE must be >= 1 byte")
	}

	for _, origin := range c.CORSAllowedOrigins {
		if _, err := url.ParseRequestURI(origin); err != nil {
			errs = append(errs, fmt.Sprintf("CORS_ALLOWED_ORIGINS contains invalid origin %q", origin))
		}
	}
	for _, p := range c.TrustedProxies {
		if !strings.Contains(p, "/") && net.ParseIP(p) == nil {
			errs = append(errs, fmt.Sprintf("TRUSTED_PROXIES contains invalid IP/CIDR %q", p))
		}
		if strings.Contains(p, "/") {
			if _, _, err := net.ParseCIDR(p); err != nil {
				errs = append(errs, fmt.Sprintf("TRUSTED_PROXIES contains invalid CIDR %q", p))
			}
		}
	}

	return errs
}

func validSSLMode(driver, mode string) bool {
	switch driver {
	case DriverPostgres:
		switch mode {
		case "disable", "require", "verify-ca", "verify-full":
			return true
		}
	case DriverMySQL:
		switch mode {
		case "disable", "require", "skip-verify":
			return true
		}
	}
	return false
}
