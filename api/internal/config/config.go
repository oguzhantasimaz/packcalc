// Package config loads runtime configuration from environment variables
// exactly once at program start. Nothing else in the application reads
// os.Environ; downstream packages take a Config value via constructor
// injection.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the resolved runtime configuration.
type Config struct {
	Port            int
	Env             string
	LogLevel        string
	RedisURL        string
	CorsOrigins     []string
	ShutdownTimeout time.Duration
}

// Load reads environment variables, applying defaults where unset.
// Currently always returns nil error; the signature reserves room for
// future required-variable validation.
func Load() (Config, error) {
	return Config{
		Port:            getInt("PORT", 8080),
		Env:             getString("ENV", "dev"),
		LogLevel:        getString("LOG_LEVEL", "info"),
		RedisURL:        getString("REDIS_URL", ""),
		CorsOrigins:     splitCSV(getString("CORS_ORIGINS", "*")),
		ShutdownTimeout: getDuration("SHUTDOWN_TIMEOUT", 30*time.Second),
	}, nil
}

// String returns a compact, secret-free representation suitable for
// logging at startup. REDIS_URL is intentionally redacted to "redis" so
// credentials never leak into logs.
func (c Config) String() string {
	store := "memory"
	if c.RedisURL != "" {
		store = "redis"
	}
	return fmt.Sprintf("port=%d env=%s log_level=%s store=%s cors=%v shutdown=%s",
		c.Port, c.Env, c.LogLevel, store, c.CorsOrigins, c.ShutdownTimeout)
}

func getString(k, def string) string {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		return v
	}
	return def
}

func getInt(k string, def int) int {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getDuration(k string, def time.Duration) time.Duration {
	if v, ok := os.LookupEnv(k); ok && v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}
