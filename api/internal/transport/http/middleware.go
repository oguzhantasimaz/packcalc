package http

import (
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	fiberrecover "github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/oklog/ulid/v2"
)

const (
	requestIDHeader = "X-Request-Id"
	requestIDKey    = "request_id"
	loggerKey       = "logger"
)

// requestIDMiddleware ensures every request has a stable identifier.
// Honors any caller-supplied X-Request-Id (e.g. from an upstream proxy)
// and otherwise mints a ULID. The id is stored in c.Locals and echoed
// in the response header.
func requestIDMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		rid := strings.TrimSpace(c.Get(requestIDHeader))
		if rid == "" {
			rid = ulid.Make().String()
		}
		c.Locals(requestIDKey, rid)
		c.Set(requestIDHeader, rid)
		return c.Next()
	}
}

// loggerMiddleware derives a request-scoped slog.Logger with request_id,
// method, and path baked in, stashes it on c.Locals, and logs one
// structured line per request on completion.
func loggerMiddleware(root *slog.Logger) fiber.Handler {
	return func(c *fiber.Ctx) error {
		rid, _ := c.Locals(requestIDKey).(string)
		log := root.With(
			slog.String("request_id", rid),
			slog.String("method", c.Method()),
			slog.String("path", c.Path()),
		)
		c.Locals(loggerKey, log)

		start := time.Now()
		err := c.Next()
		log.Info("request",
			slog.Int("status", c.Response().StatusCode()),
			slog.Int("bytes", len(c.Response().Body())),
			slog.Duration("duration", time.Since(start)),
		)
		return err
	}
}

// corsMiddleware configures CORS from the comma-separated origins list.
// An empty list or {"*"} yields a permissive wildcard configuration.
func corsMiddleware(origins []string) fiber.Handler {
	allow := "*"
	if len(origins) > 0 && !(len(origins) == 1 && origins[0] == "*") {
		allow = strings.Join(origins, ",")
	}
	return cors.New(cors.Config{
		AllowOrigins:  allow,
		AllowMethods:  "GET,PUT,POST,OPTIONS",
		AllowHeaders:  "Origin,Content-Type,If-Match,X-Request-Id",
		ExposeHeaders: "ETag,X-Request-Id",
	})
}

// recoverMiddleware turns panics into 500s via the central error handler.
func recoverMiddleware() fiber.Handler {
	return fiberrecover.New(fiberrecover.Config{EnableStackTrace: true})
}
