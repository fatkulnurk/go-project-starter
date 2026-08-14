// Package main is the web composition root. It serves the public web module
// (homepage landing page) independently from the API, so a frontend can be
// deployed and scaled on its own port.
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

	"github.com/fatkulnurk/go-project-starter/internal/modules/homepage"
	"github.com/fatkulnurk/go-project-starter/internal/platform/clock"
	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
	platformhttp "github.com/fatkulnurk/go-project-starter/internal/platform/http"
	"github.com/fatkulnurk/go-project-starter/internal/platform/logger"
	_ "time/tzdata" // embed IANA timezone data so APP_TIMEZONE works anywhere
)

func main() {
	if err := run(); err != nil {
		slog.Error("web", "err", err)
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

	homepageModule := homepage.New(homepage.Dependencies{
		Settings: homepage.Settings{
			AppName:       cfg.AppName,
			BaseURL:       cfg.BaseURL,
			AssetsBaseURL: cfg.AssetsBaseURLOrDefault(),
			Year:          clk.Now().Year(),
		},
	})

	router := platformhttp.NewRouter()
	platformhttp.MountStatic(router, cfg.PublicDir, "/assets")
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		platformhttp.WriteSuccess(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	homepageModule.RegisterWeb(router)

	srv := platformhttp.NewServer(cfg.WebPort, router)

	errCh := make(chan error, 1)
	go func() {
		log.Info("web listening", "addr", srv.Addr, "env", cfg.Environment)
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
		log.Info("web shutting down", "signal", sig.String())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		return err
	}
	if err := <-errCh; err != nil {
		return err
	}
	log.Info("web stopped")
	return nil
}
