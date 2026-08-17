// Package main is the scheduler composition root: it runs the periodic jobs
// registered by modules (e.g. the homepage "tick" demo) in their own binary,
// independent from the API and the queue worker.
package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	"github.com/fatkulnurk/go-project-starter/internal/platform/logger"
	"github.com/fatkulnurk/go-project-starter/internal/platform/schedule"
	_ "time/tzdata" // embed IANA timezone data so APP_TIMEZONE works anywhere
)

func main() {
	if err := run(); err != nil {
		slog.Error("scheduler", "err", err)
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

	clk := clock.Real{Loc: cfg.Location()}

	scheduler := schedule.New(log, cfg.Location())

	homepageModule := homepage.New(homepage.Dependencies{
		Settings: homepage.Settings{
			AppName:       cfg.AppName,
			BaseURL:       cfg.BaseURL,
			AssetsBaseURL: cfg.AssetsBaseURLOrDefault(),
			Year:          clk.Now().Year(),
		},
	})
	homepageModule.RegisterSchedule(scheduler)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		scheduler.Stop()
	}()

	log.Info("scheduler started", "env", cfg.Environment)
	if err := scheduler.Run(); err != nil {
		return err
	}
	log.Info("scheduler stopped")
	return nil
}
