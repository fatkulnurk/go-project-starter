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

// Environment names. Only these values pass validation for APP_ENV. The
// default for a missing APP_ENV is EnvironmentDevelopment.
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
	DriverDB       = "db"
	DriverLocal    = "local"
	DriverS3       = "s3"
	DriverLog      = "log"
	DriverSMTP     = "smtp"
	DriverSES      = "ses"
	DriverTwilio   = "twilio"
	DriverAsynq    = "asynq"
	DriverRabbitMQ = "rabbitmq"
	DriverKafka    = "kafka"
)

// Environment variable keys. Each maps directly to a field in Config and is
// read through the builder helpers below.
const (
	envAppEnv      = "APP_ENV"
	envAppPort     = "APP_PORT"
	envWebPort     = "WEB_PORT"
	envAppBaseURL  = "APP_BASE_URL"
	envAppName     = "APP_NAME"
	envAppTimeZone = "APP_TIMEZONE"

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

	envPubSubDriver     = "PUBSUB_DRIVER"
	envPubSubInstanceID = "PUBSUB_INSTANCE_ID"
	envRabbitMQURL      = "PUBSUB_RABBITMQ_URL"
	envRabbitMQExchange = "PUBSUB_RABBITMQ_EXCHANGE"
	envRabbitMQDurable  = "PUBSUB_RABBITMQ_DURABLE"
	envKafkaBrokers     = "PUBSUB_KAFKA_BROKERS"
	envKafkaClientID    = "PUBSUB_KAFKA_CLIENT_ID"
	envKafkaGroupPrefix = "PUBSUB_KAFKA_GROUP_PREFIX"

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
	envDefaultCountryCode       = "AUTH_DEFAULT_COUNTRY_CODE"
	envMaxActiveSessions        = "AUTH_MAX_SESSIONS"

	envRBACCacheTTL = "RBAC_CACHE_TTL"

	envMediaMaxUploadSize = "MEDIA_MAX_UPLOAD_SIZE"

	envPublicDir         = "PUBLIC_DIR"
	envAssetsBaseURL     = "ASSETS_BASE_URL"
	envTrustedProxies    = "TRUSTED_PROXIES"
	envCORSAllowedOrigin = "CORS_ALLOWED_ORIGINS"
)

// Defaults. Nothing is required at runtime except AUTH_JWT_SECRET in
// production; everything else degrades to a sensible dev value.
const (
	defaultEnvironment           = EnvironmentDevelopment
	defaultPort                  = 32100
	defaultWebPort               = 32101
	defaultBaseURL               = "http://localhost:32100"
	defaultAppName               = "Go Project Starter"
	defaultTimeZone              = "UTC"
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
	defaultPubSubDriver          = DriverMemory
	defaultRabbitMQURL           = "amqp://guest:guest@localhost:32130"
	defaultRabbitMQExchange      = "pubsub"
	defaultKafkaClientID         = "pubsub"
	defaultKafkaGroupPrefix      = "pubsub"
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
	defaultJWTSecret             = "change-me-in-production-go-project-starter"
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
	defaultPublicDir             = "./public"
	defaultMaxActiveSessions     = 10
)

// Config is the union of all settings needed by every binary.
// It is produced by Load and validated before any service is constructed.
type Config struct {
	Environment string
	Port        int
	// WebPort serves the public web (homepage) module. Kept separate from
	// Port so the API and the web front can run independently.
	WebPort int
	BaseURL string
	AppName string
	// PublicDir is the root directory served at /assets/* (static files for
	// email and web). Defaults to ./public.
	PublicDir string
	// AssetsBaseURL is the absolute base URL of static assets (defaults to
	// BaseURL when empty, so it can point at a CDN).
	AssetsBaseURL string
	// TimeZone is an IANA location name (e.g. "UTC", "Asia/Jakarta") the
	// whole app follows: DB sessions, clock and stored timestamps.
	TimeZone string

	// TrustedProxies lists CIDRs/IPs whose X-Forwarded-For header is trusted.
	TrustedProxies []string
	// CORSAllowedOrigins lists origins allowed to call the API. Empty means
	// same-origin only (no CORS headers emitted).
	CORSAllowedOrigins []string

	Database DatabaseConfig
	Cache    CacheConfig
	Queue    QueueConfig
	PubSub   PubSubConfig
	Storage  StorageConfig
	Mail     MailConfig
	SMS      SMSConfig
	Auth     AuthConfig
	RBAC     RBACConfig
	Media    MediaConfig
}

// MediaConfig holds upload constraints.
// MaxUploadSize caps the size of a single upload in bytes.
type MediaConfig struct {
	MaxUploadSize int64
}

// RBACConfig holds role/permission caching settings.
// PermissionCacheTTL controls how long cached permissions stay valid.
type RBACConfig struct {
	PermissionCacheTTL time.Duration
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
	// TimeZone is the IANA location the DB session runs in, so
	// CURRENT_TIMESTAMP matches Go-written timestamps. Mirrors Config.
	TimeZone string
}

// Location resolves TimeZone to a *time.Location, falling back to UTC when
// the value is empty. Callers use it for the clock and DB session setup.
func (c Config) Location() *time.Location {
	if c.TimeZone == "" {
		return time.UTC
	}
	if loc, err := time.LoadLocation(c.TimeZone); err == nil {
		return loc
	}
	return time.UTC
}

// AssetsBaseURLOrDefault returns the configured ASSETS_BASE_URL, falling back
// to APP_BASE_URL when empty so static asset URLs work out of the box while
// still allowing a dedicated CDN.
func (c Config) AssetsBaseURLOrDefault() string {
	if strings.TrimSpace(c.AssetsBaseURL) != "" {
		return strings.TrimRight(c.AssetsBaseURL, "/")
	}
	return strings.TrimRight(c.BaseURL, "/")
}

// CacheConfig selects the cache driver.
// Driver is one of "redis", "memory" or "db"; Redis holds the connection
// settings used when the driver is "redis".
type CacheConfig struct {
	Driver string
	Redis  RedisConfig
}

// RedisConfig holds go-redis connection settings.
// PoolSize bounds concurrent connections; the timeouts bound dial, read and
// write operations.
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

// PubSubConfig selects the pub/sub broker.
// Driver is one of "memory", "redis", "rabbitmq" or "kafka". Redis reuses the
// cache Redis settings; RabbitMQ and Kafka hold their own connection settings.
// InstanceID disambiguates subscriber instances (used to build RabbitMQ queue
// names and Kafka consumer-group IDs); empty means a random value per process.
type PubSubConfig struct {
	Driver     string
	Redis      RedisConfig
	RabbitMQ   RabbitMQConfig
	Kafka      KafkaConfig
	InstanceID string
}

// RabbitMQConfig holds AMQP settings. Durable makes subscriber queues survive
// broker restarts (messages wait until the subscriber consumes them).
type RabbitMQConfig struct {
	URL      string
	Exchange string
	Durable  bool
}

// KafkaConfig holds Kafka connection settings. GroupPrefix names the consumer
// group prefix; a unique group per instance (GroupPrefix-InstanceID) yields a
// broadcast, a shared group yields competing consumers.
type KafkaConfig struct {
	Brokers     []string
	ClientID    string
	GroupPrefix string
}

// StorageConfig selects the storage driver.
// Driver is one of "local" or "s3"; Local and S3 hold the per-driver settings.
type StorageConfig struct {
	Driver string
	Local  LocalStorageConfig
	S3     S3Config
}

// LocalStorageConfig points at the root directory for the local driver.
// Dir is the filesystem root under which object keys are resolved.
type LocalStorageConfig struct {
	Dir string
}

// S3Config holds AWS S3 / S3-compatible settings.
// Endpoint is optional (empty means real AWS); UsePathStyle enables the
// virtual-host workaround needed by MinIO-style services.
type S3Config struct {
	Endpoint     string
	Region       string
	Bucket       string
	AccessKey    string
	SecretKey    string
	UsePathStyle bool
}

// MailConfig selects the mail driver.
// Driver is one of "log", "smtp" or "ses"; From/FromName are the default
// sender, and SMTP/SES hold the per-driver settings.
type MailConfig struct {
	Driver   string
	From     string
	FromName string
	SMTP     SMTPConfig
	SES      SESConfig
}

// SMTPConfig holds plain SMTP settings.
// PoolSize bounds concurrent connections; SSL is "none", "tls" or "starttls".
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
// Credentials fall back to the default AWS credential chain when empty.
type SESConfig struct {
	Region    string
	AccessKey string
	SecretKey string
}

// SMSConfig selects the SMS driver.
// Driver is one of "log" or "twilio"; From is the default sender number and
// Twilio holds the per-driver credentials.
type SMSConfig struct {
	Driver string
	From   string
	Twilio TwilioConfig
}

// TwilioConfig holds Twilio settings.
// AccountSID and AuthToken authenticate the API; MessagingSID opts into a
// messaging service and takes precedence over the From number.
type TwilioConfig struct {
	AccountSID   string
	AuthToken    string
	MessagingSID string
}

// AuthConfig holds token, verification and rate-limit settings.
// JWTSecret signs and verifies tokens; the TTLs bound access and refresh
// tokens, and the rate-limit fields throttle login and public endpoints.
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
	// DefaultCountryCode expands local phone numbers (leading 0) into E.164
	// during normalization, e.g. "62". Empty keeps "+" numbers unchanged.
	DefaultCountryCode string
	// MaxActiveSessions caps concurrent refresh-token families per user.
	MaxActiveSessions int
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
// Load reads configuration from the environment, preceded by the .env file
// (path from ENV_FILE, default ".env" in the working directory). .env only
// fills variables that are not already set, so real env vars (and docker
// compose) keep precedence.
func Load() (Config, error) {
	envFile := os.Getenv("ENV_FILE")
	if envFile == "" {
		envFile = ".env"
	}
	if err := loadDotEnv(envFile); err != nil {
		return Config{}, err
	}
	b := &builder{}
	redis := RedisConfig{
		Addr:         b.str(envRedisAddr, defaultRedisAddr),
		Password:     b.str(envRedisPassword, ""),
		DB:           b.int(envRedisDB, defaultRedisDB),
		PoolSize:     b.int(envRedisPoolSize, defaultRedisPoolSize),
		MinIdleConns: b.int(envRedisMinIdle, defaultRedisMinIdle),
		DialTimeout:  b.duration(envRedisDialTimeout, defaultRedisDialTimeout),
		ReadTimeout:  b.duration(envRedisReadTimeout, defaultRedisReadTimeout),
		WriteTimeout: b.duration(envRedisWriteTimeout, defaultRedisWriteTimeout),
	}
	cfg := Config{
		Environment:   b.str(envAppEnv, defaultEnvironment),
		Port:          b.int(envAppPort, defaultPort),
		WebPort:       b.int(envWebPort, defaultWebPort),
		BaseURL:       b.str(envAppBaseURL, defaultBaseURL),
		AppName:       b.str(envAppName, defaultAppName),
		PublicDir:     b.str(envPublicDir, defaultPublicDir),
		AssetsBaseURL: b.str(envAssetsBaseURL, ""),
		TimeZone:      b.str(envAppTimeZone, defaultTimeZone),

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
			TimeZone:        b.str(envAppTimeZone, defaultTimeZone),
		},
		Cache: CacheConfig{
			Driver: b.str(envCacheDriver, defaultCacheDriver),
			Redis:  redis,
		},
		Queue: QueueConfig{
			Driver:        b.str(envQueueDriver, defaultQueueDriver),
			RedisAddr:     b.str(envRedisAddr, defaultRedisAddr),
			RedisPassword: b.str(envRedisPassword, ""),
			RedisDB:       b.int(envQueueRedisDB, defaultQueueRedisDB),
			RedisPoolSize: b.int(envQueueRedisPool, defaultQueueRedisPool),
			Concurrency:   b.int(envQueueConcurrency, defaultQueueConcurrency),
		},
		PubSub: PubSubConfig{
			Driver:     b.str(envPubSubDriver, defaultPubSubDriver),
			InstanceID: b.str(envPubSubInstanceID, ""),
			Redis:      redis,
			RabbitMQ: RabbitMQConfig{
				URL:      b.str(envRabbitMQURL, defaultRabbitMQURL),
				Exchange: b.str(envRabbitMQExchange, defaultRabbitMQExchange),
				Durable:  b.bool(envRabbitMQDurable),
			},
			Kafka: KafkaConfig{
				Brokers:     b.list(envKafkaBrokers),
				ClientID:    b.str(envKafkaClientID, defaultKafkaClientID),
				GroupPrefix: b.str(envKafkaGroupPrefix, defaultKafkaGroupPrefix),
			},
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
			DefaultCountryCode:    b.str(envDefaultCountryCode, ""),
			MaxActiveSessions:     b.int(envMaxActiveSessions, defaultMaxActiveSessions),
		},
		RBAC: RBACConfig{
			PermissionCacheTTL: b.duration(envRBACCacheTTL, defaultRBACCacheTTL),
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
	if strings.TrimSpace(c.PublicDir) == "" {
		errs = append(errs, "PUBLIC_DIR must not be empty")
	}
	if _, err := time.LoadLocation(c.TimeZone); err != nil {
		errs = append(errs, fmt.Sprintf("APP_TIMEZONE must be a valid IANA location (e.g. UTC, Asia/Jakarta), got %q", c.TimeZone))
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
	case DriverRedis, DriverMemory, DriverDB:
	default:
		errs = append(errs, "CACHE_DRIVER must be 'redis', 'memory' or 'db'")
	}
	if c.Cache.Redis.PoolSize < 1 {
		errs = append(errs, "REDIS_POOL_SIZE must be >= 1")
	}
	if c.Cache.Redis.DialTimeout <= 0 || c.Cache.Redis.ReadTimeout <= 0 || c.Cache.Redis.WriteTimeout <= 0 {
		errs = append(errs, "REDIS_DIAL_TIMEOUT / REDIS_READ_TIMEOUT / REDIS_WRITE_TIMEOUT must be positive")
	}

	switch c.Queue.Driver {
	case DriverAsynq, DriverDB:
	default:
		errs = append(errs, "QUEUE_DRIVER must be 'asynq' or 'db'")
	}
	if c.Queue.Concurrency < 1 {
		errs = append(errs, "QUEUE_CONCURRENCY must be >= 1")
	}
	if c.Queue.Driver == DriverAsynq && c.Queue.RedisPoolSize < 1 {
		errs = append(errs, "QUEUE_REDIS_POOL_SIZE must be >= 1")
	}

	switch c.PubSub.Driver {
	case DriverMemory, DriverRedis, DriverRabbitMQ, DriverKafka:
	default:
		errs = append(errs, "PUBSUB_DRIVER must be 'memory', 'redis', 'rabbitmq' or 'kafka'")
	}
	if c.PubSub.Driver == DriverRabbitMQ {
		u, err := url.Parse(c.PubSub.RabbitMQ.URL)
		if err != nil || (u.Scheme != "amqp" && u.Scheme != "amqps") || u.Host == "" {
			errs = append(errs, fmt.Sprintf("PUBSUB_RABBITMQ_URL must be a valid amqp(s) URL, got %q", c.PubSub.RabbitMQ.URL))
		}
		if c.PubSub.RabbitMQ.Exchange == "" {
			errs = append(errs, "PUBSUB_RABBITMQ_EXCHANGE must not be empty")
		}
	}
	if c.PubSub.Driver == DriverKafka && len(c.PubSub.Kafka.Brokers) == 0 {
		errs = append(errs, "PUBSUB_KAFKA_BROKERS must contain at least one broker when PUBSUB_DRIVER=kafka")
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
	if len(c.Auth.JWTSecret) < 32 {
		errs = append(errs, "AUTH_JWT_SECRET must be at least 32 characters")
	}
	if c.Auth.JWTSecret == defaultJWTSecret && c.Environment == EnvironmentProduction {
		errs = append(errs, "AUTH_JWT_SECRET must be changed from the default in production")
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
	if c.Auth.MaxActiveSessions < 1 {
		errs = append(errs, "AUTH_MAX_SESSIONS must be >= 1")
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
