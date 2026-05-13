// Package logging builds the slog logger used across the API. The handler
// shape is selected at startup based on the environment: a text handler
// for human-friendly local development, a JSON handler for production
// log aggregators.
package logging

import (
	"log/slog"
	"os"
	"strings"
)

// New returns a configured *slog.Logger.
//
//   - env == "dev" yields a TextHandler so log lines remain readable in
//     a terminal.
//   - Any other env yields a JSONHandler suitable for Loki, Datadog, etc.
//   - level accepts debug, info, warn (or warning), error — case
//     insensitive. Unknown values fall back to info.
func New(env, level string) *slog.Logger {
	opts := &slog.HandlerOptions{Level: parseLevel(level)}
	var h slog.Handler
	if strings.EqualFold(env, "dev") {
		h = slog.NewTextHandler(os.Stdout, opts)
	} else {
		h = slog.NewJSONHandler(os.Stdout, opts)
	}
	return slog.New(h)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
