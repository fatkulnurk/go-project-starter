// Package main is the subscriber composition root: it runs the pub/sub
// subscribers registered by modules (e.g. the homepage "app.demo.ping" demo)
// in their own binary, independent from the API and the queue worker. Unlike
// the worker it needs no database — every pub/sub broker is external.
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
	"github.com/fatkulnurk/go-project-starter/internal/platform/pubsub"
	_ "time/tzdata" // embed IANA timezone data so APP_TIMEZONE works anywhere
)

func main() {
	if err := run(); err != nil {
		slog.Error("subscriber", "err", err)
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

	subscriber, err := pubsub.NewServer(cfg.PubSub, log)
	if err != nil {
		return err
	}

	clk := clock.Real{Loc: cfg.Location()}

	homepageModule := homepage.New(homepage.Dependencies{
		Settings: homepage.Settings{
			AppName:       cfg.AppName,
			BaseURL:       cfg.BaseURL,
			AssetsBaseURL: cfg.AssetsBaseURLOrDefault(),
			Year:          clk.Now().Year(),
		},
	})
	homepageModule.RegisterPubSub(subscriber)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-stop
		subscriber.Stop()
	}()

	log.Info("subscriber started", "env", cfg.Environment, "driver", cfg.PubSub.Driver)
	if err := subscriber.Run(); err != nil {
		return err
	}
	log.Info("subscriber stopped")
	return nil
}
