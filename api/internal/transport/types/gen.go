// Package types holds request/response models generated from the
// hand-written OpenAPI 3.1 spec at api/api/openapi.yaml. Server stubs are
// intentionally not generated; handlers are hand-written against Fiber's
// idioms and consume these types directly.
package types

//go:generate go run github.com/oapi-codegen/oapi-codegen/v2/cmd/oapi-codegen -config cfg.yaml ../../../api/openapi.yaml
