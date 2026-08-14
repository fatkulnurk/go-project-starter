// Package main is the API composition root. All dependency wiring happens
// here, never inside modules.
package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/fatkulnurk/go-project-starter/internal/modules/auth"
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage"
	"github.com/fatkulnurk/go-project-starter/internal/modules/media"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac"
	"github.com/fatkulnurk/go-project-starter/internal/platform/cache"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
	"github.com/fatkulnurk/go-project-starter/internal/platform/hash"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/fatkulnurk/go-project-starter/internal/platform/logger"
	"github.com/fatkulnurk/go-project-starter/internal/platform/mailer"
	"github.com/fatkulnurk/go-project-starter/internal/platform/queue"
	"github.com/fatkulnurk/go-project-starter/internal/platform/sms"
	"github.com/fatkulnurk/go-project-starter/internal/platform/storage"
	"github.com/fatkulnurk/go-project-starter/internal/platform/token"
)

func main() {
	if err := run(); err != nil {
		slog.Error("api", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	log := logger.New(cfg.Environment)
	slog.SetDefault(log)

	clk := clock.Real{}
	devMode := cfg.Environment != config.EnvironmentProduction

	// --- infrastructure -----------------------------------------------------
	db, err := database.New(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	store, err := storage.New(cfg.Storage)
	if err != nil {
		return err
	}

	cacheClient, err := cache.New(cfg.Cache)
	if err != nil {
		return err
	}
	defer cacheClient.Close()

	queueClient, err := queue.NewClient(cfg.Queue)
	if err != nil {
		return err
	}
	defer queueClient.Close()

	mailSender, err := mailer.New(cfg.Mail)
	if err != nil {
		return err
	}

	smsSender, err := sms.New(cfg.SMS)
	if err != nil {
		return err
	}

	tokenManager := token.NewManager(cfg.Auth.JWTSecret)
	hasher := hash.NewHasher(0)

	// --- modules ------------------------------------------------------------
	rbacModule := rbac.New(rbac.Dependencies{
		DB:       db,
		DBDriver: cfg.Database.Driver,
		Cache:    cacheClient,
		CacheTTL: cfg.RBAC.PermissionCacheTTL,
	})

	authModule := auth.New(auth.Dependencies{
		DB:       db,
		DBDriver: cfg.Database.Driver,
		Cache:    cacheClient,
		Enqueuer: queueClient,
		Mailer:   mailSender,
		SMS:      smsSender,
		Tokens:   tokenManager,
		Hasher:   hasher,
		Clock:    clk,
		RBAC:     rbacModule.Service(),
		Settings: auth.Settings{
			AccessTokenTTL:       cfg.Auth.AccessTokenTTL,
			RefreshTokenTTL:      cfg.Auth.RefreshTokenTTL,
			OTPLength:            cfg.Auth.OTPLength,
			OTPTTL:               cfg.Auth.OTPTTL,
			OTPMaxAttempts:       cfg.Auth.OTPMaxAttempts,
			MagicLinkTTL:         cfg.Auth.MagicLinkTTL,
			RequireEmailVerified: cfg.Auth.RequireEmailVerified,
			RateLimitMax:         cfg.Auth.RateLimitLoginMax,
			RateLimitWindow:      cfg.Auth.RateLimitLoginWindow,
			BaseURL:              cfg.BaseURL,
			AppName:              cfg.AppName,
			DevMode:              devMode,
		},
	})

	mediaModule := media.New(media.Dependencies{
		DB:       db,
		DBDriver: cfg.Database.Driver,
		Storage:  store,
		Disk:     cfg.Storage.Driver,
	})

	homepageModule := homepage.New(homepage.Dependencies{
		Settings: homepage.Settings{
			AppName: cfg.AppName,
			BaseURL: cfg.BaseURL,
			Year:    clk.Now().Year(),
		},
	})

	// Bootstrap default roles/permissions and assign the super admin.
	if err := rbacModule.Bootstrap(context.Background(), rbac.BootstrapOptions{
		SuperAdminEmail: cfg.RBAC.SuperAdminEmail,
		FindUserID: func(ctx context.Context, email string) (string, error) {
			user, err := authModule.API.FindUserByEmail.Execute(ctx, email)
			if err != nil {
				return "", err
			}
			return user.ID, nil
		},
	}); err != nil {
		return err
	}

	// --- HTTP server ---------------------------------------------------------
	authorizer := rbacModule.Authorizer()

	router := platformhttp.NewRouter()
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		platformhttp.WriteSuccess(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	homepageModule.RegisterHTTP(router)
	authModule.RegisterHTTP(router)
	mediaModule.RegisterHTTP(router, authModule.Authenticator(), authorizer)
	rbacModule.RegisterHTTP(router, authModule.Authenticator(), authorizer)

	srv := platformhttp.NewServer(cfg.Port, router)
	go func() {
		log.Info("api listening", "addr", srv.Addr, "env", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			slog.Error("http server", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	log.Info("api stopped")
	return nil
}
