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

	appid "github.com/fatkulnurk/go-project-starter/internal/application/id"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth"
	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac"
	"github.com/fatkulnurk/go-project-starter/internal/modules/rbac/application/command"
	rbacseeder "github.com/fatkulnurk/go-project-starter/internal/modules/rbac/seeder"
	"github.com/fatkulnurk/go-project-starter/internal/platform/audit"
	"github.com/fatkulnurk/go-project-starter/internal/platform/cache"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
	"github.com/fatkulnurk/go-project-starter/internal/platform/hash"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	platformid "github.com/fatkulnurk/go-project-starter/internal/platform/id"
	"github.com/fatkulnurk/go-project-starter/internal/platform/logger"
	"github.com/fatkulnurk/go-project-starter/internal/platform/mailer"
	"github.com/fatkulnurk/go-project-starter/internal/platform/queue"
	"github.com/fatkulnurk/go-project-starter/internal/platform/sms"
	"github.com/fatkulnurk/go-project-starter/internal/platform/token"
	_ "time/tzdata" // embed IANA timezone data so APP_TIMEZONE works anywhere
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

	// Wire the shared identifier generator before anything mints IDs.
	appid.SetDefault(platformid.Generator{})

	clk := clock.Real{Loc: cfg.Location()}
	devMode := cfg.Environment != config.EnvironmentProduction

	// --- infrastructure -----------------------------------------------------
	db, err := database.New(cfg.Database)
	if err != nil {
		return err
	}
	defer db.Close()

	cacheClient, err := cache.New(cfg.Cache, db, cfg.Database.Driver)
	if err != nil {
		return err
	}
	defer cacheClient.Close()

	queueClient, err := queue.NewClient(cfg.Queue, db, cfg.Database.Driver)
	if err != nil {
		return err
	}
	defer queueClient.Close()

	mailSender, err := mailer.New(cfg.Mail)
	if err != nil {
		return err
	}
	defer mailSender.Close()

	smsSender, err := sms.New(cfg.SMS)
	if err != nil {
		return err
	}

	tokenManager := token.NewManager(cfg.Auth.JWTSecret, cfg.Auth.JWTIssuer, cfg.Auth.JWTAudience)
	hasher := hash.NewBCrypt(0)
	auditor := audit.New(db, cfg.Database.Driver, cfg.Location())

	// --- modules ------------------------------------------------------------
	rbacModule := rbac.New(rbac.Dependencies{
		DB:       db,
		DBDriver: cfg.Database.Driver,
		Cache:    cacheClient,
		CacheTTL: cfg.RBAC.PermissionCacheTTL,
		Auditor:  auditor,
	})

	// Ensure the well-known roles/permissions exist so registration and the
	// RBAC admin API work on a fresh database without a manual seed step.
	if err := rbacModule.Bootstrap(context.Background(), command.BootstrapOptions{
		DefaultRoles:       rbacseeder.DefaultRoles,
		DefaultPermissions: rbacseeder.DefaultPermissions,
	}); err != nil {
		return err
	}

	authModule := auth.New(auth.Dependencies{
		DB:       db,
		DBDriver: cfg.Database.Driver,
		Cache:    cacheClient,
		Enqueuer: queueClient, Mailer: mailSender,
		SMS:      smsSender,
		Tokens:   tokenManager,
		Hasher:   hasher,
		Clock:    clk,
		RBAC:     rbacModule.Service(),
		Auditor:  auditor,
		Location: cfg.Location(),
		Settings: auth.Settings{
			AccessTokenTTL:        cfg.Auth.AccessTokenTTL,
			RefreshTokenTTL:       cfg.Auth.RefreshTokenTTL,
			OTPLength:             cfg.Auth.OTPLength,
			OTPTTL:                cfg.Auth.OTPTTL,
			OTPMaxAttempts:        cfg.Auth.OTPMaxAttempts,
			MagicLinkTTL:          cfg.Auth.MagicLinkTTL,
			RequireEmailVerified:  cfg.Auth.RequireEmailVerified,
			RateLimitMax:          cfg.Auth.RateLimitLoginMax,
			RateLimitWindow:       cfg.Auth.RateLimitLoginWindow,
			PublicRateLimitMax:    cfg.Auth.RateLimitPublicMax,
			PublicRateLimitWindow: cfg.Auth.RateLimitPublicWindow,
			BaseURL:               cfg.BaseURL,
			AppName:               cfg.AppName,
			AssetsBaseURL:         cfg.AssetsBaseURLOrDefault(),
			DefaultCountryCode:    cfg.Auth.DefaultCountryCode,
			MaxActiveSessions:     cfg.Auth.MaxActiveSessions,
			DevMode:               devMode,
		},
	})

	homepageModule := homepage.New(homepage.Dependencies{
		Settings: homepage.Settings{
			AppName:       cfg.AppName,
			BaseURL:       cfg.BaseURL,
			AssetsBaseURL: cfg.AssetsBaseURLOrDefault(),
			Year:          clk.Now().Year(),
		},
	})

	// --- HTTP server ---------------------------------------------------------
	authorizer := rbacModule.Authorizer()
	platformhttp.SetTrustedProxies(cfg.TrustedProxies)

	router := platformhttp.NewRouter(platformhttp.RouterOptions{
		CORSAllowedOrigins: cfg.CORSAllowedOrigins,
	})
	platformhttp.MountStatic(router, cfg.PublicDir, "/assets")
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		platformhttp.WriteSuccess(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := db.PingContext(ctx); err != nil {
			platformhttp.WriteMappedError(w, err)
			return
		}
		if err := cacheClient.Ping(ctx); err != nil {
			platformhttp.WriteMappedError(w, err)
			return
		}
		platformhttp.WriteSuccess(w, http.StatusOK, map[string]string{"status": "ready"})
	})
	authModule.RegisterAPI(router)
	homepageModule.RegisterAPI(router)
	rbacModule.RegisterAPI(router, authModule.Authenticator(), authorizer)

	srv := platformhttp.NewServer(cfg.Port, router)

	errCh := make(chan error, 1)
	go func() {
		log.Info("api listening", "addr", srv.Addr, "env", cfg.Environment)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
			return
		}
		errCh <- nil
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
		return nil
	case sig := <-stop:
		log.Info("api shutting down", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	if err := <-errCh; err != nil {
		return err
	}
	log.Info("api stopped")
	return nil
}
