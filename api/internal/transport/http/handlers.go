package http

import (
	"context"
	"log/slog"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/oguzhantasimaz/packcalc/api/internal/packs"
	"github.com/oguzhantasimaz/packcalc/api/internal/store"
	"github.com/oguzhantasimaz/packcalc/api/internal/transport/types"
)

// Pinger is the optional interface a PackStore may satisfy to participate
// in the /readyz probe. Implementations whose health is implicit (e.g.
// the in-memory store) need not implement it.
type Pinger interface {
	Ping(ctx context.Context) error
}

// readyzPingTimeout caps the time spent verifying the store on /readyz.
// Matches the budget documented in the design spec.
const readyzPingTimeout = 500 * time.Millisecond

// Handlers groups the HTTP handlers and their dependencies. Dependencies
// are injected via NewHandlers; the handler set itself holds no state.
type Handlers struct {
	store store.PackStore
	log   *slog.Logger
}

// NewHandlers constructs a Handlers value.
func NewHandlers(s store.PackStore, log *slog.Logger) *Handlers {
	return &Handlers{store: s, log: log}
}

// Healthz responds to the liveness probe. It performs no I/O.
func (h *Handlers) Healthz(c *fiber.Ctx) error {
	return c.Status(fiber.StatusOK).JSON(types.HealthResponse{Status: "ok"})
}

// Readyz reports whether the API can serve traffic. If the underlying
// store implements Pinger, the store is pinged with a short timeout; a
// failed ping yields 503 with a NotReady envelope.
func (h *Handlers) Readyz(c *fiber.Ctx) error {
	if p, ok := h.store.(Pinger); ok {
		ctx, cancel := context.WithTimeout(c.UserContext(), readyzPingTimeout)
		defer cancel()
		if err := p.Ping(ctx); err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(types.ErrorResponse{
				Code:      CodeNotReady,
				Message:   err.Error(),
				RequestId: requestID(c),
			})
		}
	}
	return c.Status(fiber.StatusOK).JSON(types.HealthResponse{Status: "ready"})
}

// GetPacks returns the current pack-size set plus its version, mirroring
// the version in the ETag response header.
func (h *Handlers) GetPacks(c *fiber.Ctx) error {
	snap, err := h.store.Get(c.UserContext())
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderETag, snap.Version)
	return c.Status(fiber.StatusOK).JSON(types.PackSet{Sizes: snap.Sizes, Version: snap.Version})
}

// PutPacks atomically replaces the pack-size set, honoring an optional
// If-Match precondition. The response carries the new version both in
// the body and in the ETag header.
func (h *Handlers) PutPacks(c *fiber.Ctx) error {
	var req types.PutPacksRequest
	if err := c.BodyParser(&req); err != nil {
		return wrapBody(err)
	}
	ifMatch := c.Get(fiber.HeaderIfMatch)
	snap, err := h.store.Set(c.UserContext(), req.Sizes, ifMatch)
	if err != nil {
		return err
	}
	c.Set(fiber.HeaderETag, snap.Version)
	return c.Status(fiber.StatusOK).JSON(types.PackSet{Sizes: snap.Sizes, Version: snap.Version})
}

// Calculate computes the pack combination for the given order using the
// currently stored pack-size set.
func (h *Handlers) Calculate(c *fiber.Ctx) error {
	var req types.CalculateRequest
	if err := c.BodyParser(&req); err != nil {
		return wrapBody(err)
	}
	snap, err := h.store.Get(c.UserContext())
	if err != nil {
		return err
	}
	result, err := packs.Calculate(req.Order, snap.Sizes)
	if err != nil {
		return err
	}
	out := types.CalculateResponse{
		Packs:      make([]types.PackCount, 0, len(result.Packs)),
		TotalItems: result.TotalItems,
		TotalPacks: result.TotalPacks,
		Overshoot:  result.TotalItems - req.Order,
	}
	for _, p := range result.Packs {
		out.Packs = append(out.Packs, types.PackCount{Size: p.Size, Count: p.Count})
	}
	return c.Status(fiber.StatusOK).JSON(out)
}
