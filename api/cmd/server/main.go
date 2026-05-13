// Command server runs the packcalc HTTP API.
//
// Configuration is read from environment variables exactly once at
// startup (see internal/config). The PackStore implementation is chosen
// based on REDIS_URL presence. The process traps SIGINT/SIGTERM and
// performs a graceful Fiber shutdown bounded by SHUTDOWN_TIMEOUT.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/redis/go-redis/v9"

	"github.com/oguzhantasimaz/packcalc/api/internal/config"
	"github.com/oguzhantasimaz/packcalc/api/internal/logging"
	"github.com/oguzhantasimaz/packcalc/api/internal/store"
	httptransport "github.com/oguzhantasimaz/packcalc/api/internal/transport/http"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("config: %w", err)
	}
	log := logging.New(cfg.Env, cfg.LogLevel)
	log.Info("startup", slog.String("config", cfg.String()))

	ps, redisClient, err := buildStore(context.Background(), cfg)
	if err != nil {
		return fmt.Errorf("store: %w", err)
	}
	if redisClient != nil {
		defer func() { _ = redisClient.Close() }()
	}

	handlers := httptransport.NewHandlers(ps, log)
	app := httptransport.NewRouter(handlers, cfg, log)

	errCh := make(chan error, 1)
	go func() {
		addr := fmt.Sprintf(":%d", cfg.Port)
		log.Info("listening", slog.String("addr", addr))
		if err := app.Listen(addr); err != nil {
			errCh <- err
		}
	}()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-sigCh:
		log.Info("shutdown requested", slog.String("signal", sig.String()))
	case err := <-errCh:
		return fmt.Errorf("listen: %w", err)
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()
	if err := app.ShutdownWithContext(shutdownCtx); err != nil {
		log.Error("shutdown failed", slog.String("err", err.Error()))
		return err
	}
	log.Info("shutdown complete")
	return nil
}

// buildStore selects the PackStore implementation. The redis.Client is
// returned alongside so main can close it on shutdown.
func buildStore(ctx context.Context, cfg config.Config) (store.PackStore, *redis.Client, error) {
	if cfg.RedisURL == "" {
		return store.NewMemory(), nil, nil
	}
	opts, err := redis.ParseURL(cfg.RedisURL)
	if err != nil {
		return nil, nil, fmt.Errorf("parse REDIS_URL: %w", err)
	}
	client := redis.NewClient(opts)
	s, err := store.NewRedis(ctx, client)
	if err != nil {
		return nil, nil, errors.Join(err, client.Close())
	}
	return s, client, nil
}
