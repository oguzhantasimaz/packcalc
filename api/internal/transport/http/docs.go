package http

import (
	stdhttp "net/http"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/filesystem"

	apidocs "github.com/oguzhantasimaz/packcalc/api/api"
)

// mountDocs wires the OpenAPI spec and the embedded Swagger UI:
//
//   - GET /openapi.yaml — raw spec, served as application/yaml.
//   - GET /docs and /docs/<asset> — Swagger UI index plus its static
//     assets, served from the embedded FS.
//
// Both are served from the same origin as the API to avoid CORS friction
// when the UI fetches the spec.
func mountDocs(app *fiber.App) {
	app.Get("/openapi.yaml", func(c *fiber.Ctx) error {
		c.Set(fiber.HeaderContentType, "application/yaml; charset=utf-8")
		return c.Send(apidocs.OpenAPI)
	})

	app.Use("/docs", filesystem.New(filesystem.Config{
		Root:       stdhttp.FS(apidocs.SwaggerUI),
		PathPrefix: "swagger-ui",
		Index:      "index.html",
		Browse:     false,
	}))
}
