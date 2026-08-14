// Package logger wraps log/slog with context-aware request logging.
package logger

import (
	"context"
	"log/slog"
	"os"

	"github.com/fatkulnurk/go-project-starter/internal/platform/config"
)

// New builds the application-wide logger from an env string
// ("development" | "production"). Production emits JSON to stdout for easy
// ingestion; development uses human-readable text at debug level.
func New(environment string) *slog.Logger {
	level := slog.LevelInfo
	if environment == config.EnvironmentDevelopment {
		level = slog.LevelDebug
	}
	if environment == config.EnvironmentProduction {
		return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
	}
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: level}))
}

type ctxKey struct{}

// With stores the logger inside ctx.
func With(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// From returns the logger stored in ctx, or a fallback logger.
func From(ctx context.Context) *slog.Logger {
	if l, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok {
		return l
	}
	return slog.Default()
}
