package http

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/oguzhantasimaz/packcalc/api/internal/config"
)

// NewRouter wires middleware and routes against the supplied handlers,
// returning a configured *fiber.App ready to listen. The middleware
// chain order is significant — recover wraps everything; request_id
// runs before the logger so log lines carry the id.
func NewRouter(h *Handlers, cfg config.Config, log *slog.Logger) *fiber.App {
	app := fiber.New(fiber.Config{
		AppName:               "packcalc",
		DisableStartupMessage: true,
		ErrorHandler:          errorHandler(log),
		// 16 KB request header buffer. Fiber/fasthttp's default is 4 KB,
		// which rejects browsers with large cookie sets (the 431
		// "Request Header Fields Too Large" path). 16 KB matches
		// Cloudflare's default and covers any reasonable client while
		// still rejecting genuinely abusive requests.
		ReadBufferSize: 16384,
	})

	app.Use(recoverMiddleware())
	app.Use(requestIDMiddleware())
	app.Use(loggerMiddleware(log))
	app.Use(corsMiddleware(cfg.CorsOrigins))

	app.Get("/healthz", h.Healthz)
	app.Get("/readyz", h.Readyz)

	mountDocs(app)

	v1 := app.Group("/api/v1")
	v1.Get("/packs", h.GetPacks)
	v1.Put("/packs", h.PutPacks)
	v1.Post("/calculate", h.Calculate)

	return app
}
