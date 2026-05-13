package http

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"github.com/oguzhantasimaz/packcalc/api/internal/packs"
	"github.com/oguzhantasimaz/packcalc/api/internal/store"
	"github.com/oguzhantasimaz/packcalc/api/internal/transport/types"
)

// Stable machine-readable error slugs returned in ErrorResponse.code.
const (
	CodeInvalidRequest  = "invalid_request"
	CodeVersionMismatch = "version_mismatch"
	CodeLimitExceeded   = "limit_exceeded"
	CodeNotReady        = "not_ready"
	CodeInternal        = "internal"
)

// ErrBodyDecode wraps a malformed request body so that decode failures
// map cleanly onto 400 / invalid_request through statusForError.
var ErrBodyDecode = errors.New("malformed request body")

// wrapBody decorates a decode error so the central error handler maps it
// to 400 invalid_request.
func wrapBody(err error) error {
	return fmt.Errorf("%w: %s", ErrBodyDecode, err)
}

// statusForError maps a domain error to its HTTP status and slug. Returns
// 500/internal as the catch-all so the error handler can decide whether
// to further refine based on *fiber.Error.
func statusForError(err error) (int, string) {
	switch {
	case errors.Is(err, ErrBodyDecode),
		errors.Is(err, packs.ErrInvalidOrder),
		errors.Is(err, packs.ErrEmptySizes),
		errors.Is(err, packs.ErrInvalidSize),
		errors.Is(err, packs.ErrDuplicateSize):
		return fiber.StatusBadRequest, CodeInvalidRequest
	case errors.Is(err, packs.ErrLimitExceeded):
		return fiber.StatusUnprocessableEntity, CodeLimitExceeded
	case errors.Is(err, store.ErrVersionMismatch):
		return fiber.StatusPreconditionFailed, CodeVersionMismatch
	}
	return fiber.StatusInternalServerError, CodeInternal
}

// errorHandler returns Fiber's app-wide error handler. Every error in the
// application leaves through this funnel so the response envelope is
// identical across endpoints and middleware.
func errorHandler(root *slog.Logger) fiber.ErrorHandler {
	return func(c *fiber.Ctx, err error) error {
		status, code := statusForError(err)
		// If the error chain carries a *fiber.Error (e.g. 404 from
		// routing, payload-too-large from body limit), use its status.
		var fe *fiber.Error
		if status == fiber.StatusInternalServerError && errors.As(err, &fe) {
			status = fe.Code
			if status >= 400 && status < 500 {
				code = CodeInvalidRequest
			}
		}

		log := getLogger(c, root)
		if status >= 500 {
			log.Error("request failed", slog.Int("status", status), slog.String("err", err.Error()))
		} else {
			log.Info("request rejected",
				slog.Int("status", status),
				slog.String("code", code),
				slog.String("err", err.Error()),
			)
		}

		return c.Status(status).JSON(types.ErrorResponse{
			Code:      code,
			Message:   err.Error(),
			RequestId: requestID(c),
		})
	}
}

// requestID returns the request-scoped id stamped by the request-id
// middleware, or the empty string if not set (which only happens before
// middleware runs).
func requestID(c *fiber.Ctx) string {
	if v, ok := c.Locals(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func getLogger(c *fiber.Ctx, root *slog.Logger) *slog.Logger {
	if v, ok := c.Locals(loggerKey).(*slog.Logger); ok {
		return v
	}
	return root
}
