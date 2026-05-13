// Package apidocs embeds the hand-written OpenAPI 3.1 spec and the
// vendored Swagger UI distribution into the server binary. Embedding
// means the running image is fully self-contained: no volume mounts, no
// ConfigMap-as-file, no init container. The package exports two values
// that the HTTP transport serves directly.
package apidocs

import "embed"

// OpenAPI is the raw bytes of the OpenAPI 3.1 specification. Served at
// GET /openapi.yaml with content-type application/yaml.
//
//go:embed openapi.yaml
var OpenAPI []byte

// SwaggerUI is the vendored Swagger UI dist plus a hand-written index.html.
// Served at /docs via Fiber's filesystem middleware.
//
//go:embed swagger-ui
var SwaggerUI embed.FS
