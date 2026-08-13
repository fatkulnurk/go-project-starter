// Package config loads application configuration from environment variables.
// It is read once per binary in the composition root, never mid-flow.
package config

import (
	"errors"
	"os"
	"strconv"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/platform/dbdriver"
)

// Environment names.
const (
	EnvironmentDevelopment = "development"
	EnvironmentProduction  = "production"
)

// Cache, storage, mail, sms and queue driver names. Each factory in
// internal/platform switches on these constants.
const (
	DriverRedis  = "redis"
	DriverMemory = "memory"
	DriverLocal  = "local"
	DriverS3     = "s3"
	DriverLog    = "log"
	DriverSMTP   = "smtp"
	DriverSES    = "ses"
	DriverTwilio = "twilio"
	DriverAsynq  = "asynq"
)

// Environment variable keys.
const (
	envAppEnv     = "APP_ENV"
	envAppPort    = "APP_PORT"
	envAppBaseURL = "APP_BASE_URL"

	envDBDriver   = "DB_DRIVER"
	envDBHost     = "DB_HOST"
	envDBPort     = "DB_PORT"
	envDBUser     = "DB_USER"
	envDBPassword = "DB_PASSWORD"
	envDBName     = "DB_NAME"

	envCacheDriver   = "CACHE_DRIVER"
	envRedisAddr     = "REDIS_ADDR"
	envRedisPassword = "REDIS_PASSWORD"
	envRedisDB       = "REDIS_DB"

	envQueueDriver      = "QUEUE_DRIVER"
	envQueueConcurrency = "QUEUE_CONCURRENCY"

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
	envSESRegion    = "SES_REGION"
	envSESAccessKey = "SES_ACCESS_KEY"
	envSESSecretKey = "SES_SECRET_KEY"

	envSMSDriver     = "SMS_DRIVER"
	envSMSFrom       = "SMS_FROM"
	envTwilioAccount = "TWILIO_ACCOUNT_SID"
	envTwilioAuth    = "TWILIO_AUTH_TOKEN"
	envTwilioMessSID = "TWILIO_MESSAGE_SID"

	envAuthJWTSecret            = "AUTH_JWT_SECRET"
	envAuthAccessTokenTTL       = "AUTH_ACCESS_TOKEN_TTL"
	envAuthRefreshTokenTTL      = "AUTH_REFRESH_TOKEN_TTL"
	envAuthRequireEmailVerified = "AUTH_REQUIRE_EMAIL_VERIFIED"
	envOTPLength                = "OTP_LENGTH"
	envOTPTTL                   = "OTP_TTL"
	envOTPMaxAttempts           = "OTP_MAX_ATTEMPTS"
	envRateLimitLoginMax        = "RATE_LIMIT_LOGIN_MAX"
	envRateLimitLoginWindow     = "RATE_LIMIT_LOGIN_WINDOW"

	envRBACCacheTTL       = "RBAC_CACHE_TTL"
	envRBACSuperAdminMail = "RBAC_BOOTSTRAP_SUPER_ADMIN_EMAIL"
)

// Defaults. Nothing is required at runtime except AUTH_JWT_SECRET in
// production; everything else degrades to a sensible dev value.
const (
	defaultEnvironment       = EnvironmentDevelopment
	defaultPort              = 8080
	defaultBaseURL           = "http://localhost:8080"
	defaultDBDriver          = dbdriver.MySQL
	defaultDBHost            = "localhost"
	defaultDBPort            = 3306
	defaultDBUser            = "root"
	defaultDBName            = "go_project_starter"
	defaultCacheDriver       = DriverRedis
	defaultRedisAddr         = "localhost:6379"
	defaultRedisDB           = 0
	defaultQueueDriver       = DriverAsynq
	defaultQueueConcurrency  = 10
	defaultStorageDriver     = DriverLocal
	defaultStorageLocalDir   = "./storage"
	defaultS3Region          = "us-east-1"
	defaultMailDriver        = DriverLog
	defaultMailFrom          = "no-reply@example.com"
	defaultMailFromName      = "Go Project Starter"
	defaultSMTPHost          = "smtp.example.com"
	defaultSMTPPort          = 587
	defaultSESRegion         = "us-east-1"
	defaultSMSDriver         = DriverLog
	defaultJWTSecret         = "change-me-in-production"
	defaultAccessTokenTTL    = 15 * time.Minute
	defaultRefreshTokenTTL   = 720 * time.Hour
	defaultOTPLength         = 6
	defaultOTPTTL            = 15 * time.Minute
	defaultOTPMaxAttempts    = 5
	defaultRateLimitLoginMax = 5
	defaultRateLimitWindow   = 15 * time.Minute
	defaultRBACCacheTTL      = 5 * time.Minute
)

// Config is the union of all settings needed by every binary.
type Config struct {
	Environment string
	Port        int
	BaseURL     string

	Database DatabaseConfig
	Cache    CacheConfig
	Queue    QueueConfig
	Storage  StorageConfig
	Mail     MailConfig
	SMS      SMSConfig
	Auth     AuthConfig
	RBAC     RBACConfig
}

// RBACConfig holds role/permission bootstrap and caching settings.
type RBACConfig struct {
	PermissionCacheTTL time.Duration
	SuperAdminEmail    string
}

// DatabaseConfig selects the SQL driver and connection.
type DatabaseConfig struct {
	Driver   string
	Host     string
	Port     int
	User     string
	Password string
	Name     string
}

// CacheConfig selects the cache driver.
type CacheConfig struct {
	Driver string
	Redis  RedisConfig
}

// RedisConfig holds go-redis connection settings.
type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

// QueueConfig selects the queue backend.
type QueueConfig struct {
	Driver      string
	RedisAddr   string
	Concurrency int
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
	JWTSecret            string
	AccessTokenTTL       time.Duration
	RefreshTokenTTL      time.Duration
	RequireEmailVerified bool
	OTPLength            int
	OTPTTL               time.Duration
	OTPMaxAttempts       int
	MagicLinkTTL         time.Duration
	RateLimitLoginMax    int64
	RateLimitLoginWindow time.Duration
}

// Load reads configuration from the environment. It fails fast when required
// values are missing.
func Load() (Config, error) {
	cfg := Config{
		Environment: getenv(envAppEnv, defaultEnvironment),
		Port:        getenvInt(envAppPort, defaultPort),
		BaseURL:     getenv(envAppBaseURL, defaultBaseURL),
		Database: DatabaseConfig{
			Driver:   getenv(envDBDriver, defaultDBDriver),
			Host:     getenv(envDBHost, defaultDBHost),
			Port:     getenvInt(envDBPort, defaultDBPort),
			User:     getenv(envDBUser, defaultDBUser),
			Password: getenv(envDBPassword, ""),
			Name:     getenv(envDBName, defaultDBName),
		},
		Cache: CacheConfig{
			Driver: getenv(envCacheDriver, defaultCacheDriver),
			Redis: RedisConfig{
				Addr:     getenv(envRedisAddr, defaultRedisAddr),
				Password: getenv(envRedisPassword, ""),
				DB:       getenvInt(envRedisDB, defaultRedisDB),
			},
		},
		Queue: QueueConfig{
			Driver:      getenv(envQueueDriver, defaultQueueDriver),
			RedisAddr:   getenv(envRedisAddr, defaultRedisAddr),
			Concurrency: getenvInt(envQueueConcurrency, defaultQueueConcurrency),
		},
		Storage: StorageConfig{
			Driver: getenv(envStorageDriver, defaultStorageDriver),
			Local:  LocalStorageConfig{Dir: getenv(envStorageLocalDir, defaultStorageLocalDir)},
			S3: S3Config{
				Endpoint:     getenv(envStorageS3Endpoint, ""),
				Region:       getenv(envStorageS3Region, defaultS3Region),
				Bucket:       getenv(envStorageS3Bucket, ""),
				AccessKey:    getenv(envStorageS3AccessKey, ""),
				SecretKey:    getenv(envStorageS3SecretKey, ""),
				UsePathStyle: getenv(envStorageS3PathStyle, "") == "true",
			},
		},
		Mail: MailConfig{
			Driver:   getenv(envMailDriver, defaultMailDriver),
			From:     getenv(envMailFrom, defaultMailFrom),
			FromName: getenv(envMailFromName, defaultMailFromName),
			SMTP: SMTPConfig{
				Host:     getenv(envSMTPHost, defaultSMTPHost),
				Port:     getenvInt(envSMTPPort, defaultSMTPPort),
				User:     getenv(envSMTPUser, ""),
				Password: getenv(envSMTPPassword, ""),
			},
			SES: SESConfig{
				Region:    getenv(envSESRegion, defaultSESRegion),
				AccessKey: getenv(envSESAccessKey, ""),
				SecretKey: getenv(envSESSecretKey, ""),
			},
		},
		SMS: SMSConfig{
			Driver: getenv(envSMSDriver, defaultSMSDriver),
			From:   getenv(envSMSFrom, ""),
			Twilio: TwilioConfig{
				AccountSID:   getenv(envTwilioAccount, ""),
				AuthToken:    getenv(envTwilioAuth, ""),
				MessagingSID: getenv(envTwilioMessSID, ""),
			},
		},
		Auth: AuthConfig{
			JWTSecret:            getenv(envAuthJWTSecret, defaultJWTSecret),
			AccessTokenTTL:       getenvDuration(envAuthAccessTokenTTL, defaultAccessTokenTTL),
			RefreshTokenTTL:      getenvDuration(envAuthRefreshTokenTTL, defaultRefreshTokenTTL),
			RequireEmailVerified: getenv(envAuthRequireEmailVerified, "") == "true",
			OTPLength:            getenvInt(envOTPLength, defaultOTPLength),
			OTPTTL:               getenvDuration(envOTPTTL, defaultOTPTTL),
			OTPMaxAttempts:       getenvInt(envOTPMaxAttempts, defaultOTPMaxAttempts),
			MagicLinkTTL:         getenvDuration(envOTPTTL, defaultOTPTTL),
			RateLimitLoginMax:    int64(getenvInt(envRateLimitLoginMax, defaultRateLimitLoginMax)),
			RateLimitLoginWindow: getenvDuration(envRateLimitLoginWindow, defaultRateLimitWindow),
		},
		RBAC: RBACConfig{
			PermissionCacheTTL: getenvDuration(envRBACCacheTTL, defaultRBACCacheTTL),
			SuperAdminEmail:    getenv(envRBACSuperAdminMail, ""),
		},
	}

	if err := cfg.validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) validate() error {
	switch c.Database.Driver {
	case dbdriver.MySQL, dbdriver.Postgres:
	default:
		return errors.New("DB_DRIVER must be 'mysql' or 'postgres'")
	}
	switch c.Cache.Driver {
	case DriverRedis, DriverMemory:
	default:
		return errors.New("CACHE_DRIVER must be 'redis' or 'memory'")
	}
	if c.Auth.JWTSecret == defaultJWTSecret && c.Environment == EnvironmentProduction {
		return errors.New("AUTH_JWT_SECRET must be set in production")
	}
	return nil
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getenvInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func getenvDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
