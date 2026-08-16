// Package main is the worker composition root: it processes asynq tasks
// (email/SMS delivery) enqueued by the API.
package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	appid "github.com/fatkulnurk/go-project-starter/internal/application/id"
	"github.com/fatkulnurk/go-project-starter/internal/modules/auth"
	"github.com/fatkulnurk/go-project-starter/internal/platform/cache"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/fatkulnurk/go-project-starter/internal/platform/database"
	"github.com/fatkulnurk/go-project-starter/internal/platform/hash"
	"github.com/fatkulnurk/go-project-starter/internal/platform/id"
	"github.com/fatkulnurk/go-project-starter/internal/platform/logger"
	"github.com/fatkulnurk/go-project-starter/internal/platform/mailer"
	"github.com/fatkulnurk/go-project-starter/internal/platform/queue"
	"github.com/fatkulnurk/go-project-starter/internal/platform/sms"
	"github.com/fatkulnurk/go-project-starter/internal/platform/token"
	_ "time/tzdata" // embed IANA timezone data so APP_TIMEZONE works anywhere
)

func main() {
	if err := run(); err != nil {
		slog.Error("worker", "err", err)
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
	appid.SetDefault(id.Generator{})

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

	mailSender, err := mailer.New(cfg.Mail)
	if err != nil {
		return err
	}
	defer mailSender.Close()

	smsSender, err := sms.New(cfg.SMS)
	if err != nil {
		return err
	}

	queueServer, err := queue.NewServer(cfg.Queue, log, db, cfg.Database.Driver)
	if err != nil {
		return err
	}

	queueClient, err := queue.NewClient(cfg.Queue, db, cfg.Database.Driver)
	if err != nil {
		return err
	}
	defer queueClient.Close()

	tokenManager := token.NewManager(cfg.Auth.JWTSecret, cfg.Auth.JWTIssuer, cfg.Auth.JWTAudience)
	hasher := hash.NewBCrypt(0)
	clk := clock.Real{Loc: cfg.Location()}
	devMode := cfg.Environment != config.EnvironmentProduction

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

	authModule.RegisterQueue(queueServer)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		queueServer.Stop()
	}()

	log.Info("worker started", "env", cfg.Environment)
	if err := queueServer.Run(); err != nil {
		return err
	}
	log.Info("worker stopped")
	return nil
}
