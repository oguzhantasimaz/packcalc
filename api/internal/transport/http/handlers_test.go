package http

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"reflect"
	"sort"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/oguzhantasimaz/packcalc/api/internal/config"
	"github.com/oguzhantasimaz/packcalc/api/internal/store"
	"github.com/oguzhantasimaz/packcalc/api/internal/transport/types"
)

// newApp builds a fresh app + memory store + discard logger for a test.
func newApp(t *testing.T) (*fiber.App, *store.Memory) {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	mem := store.NewMemory()
	h := NewHandlers(mem, log)
	cfg := config.Config{CorsOrigins: []string{"*"}}
	return NewRouter(h, cfg, log), mem
}

// newAppWithStore wires an arbitrary PackStore (handy for stubs).
func newAppWithStore(t *testing.T, s store.PackStore) *fiber.App {
	t.Helper()
	log := slog.New(slog.NewTextHandler(io.Discard, nil))
	h := NewHandlers(s, log)
	cfg := config.Config{CorsOrigins: []string{"*"}}
	return NewRouter(h, cfg, log)
}

// do issues a request through fiber's in-process test harness.
func do(t *testing.T, app *fiber.App, method, path string, body any, headers map[string]string) *http.Response {
	t.Helper()
	var r io.Reader
	if body != nil {
		switch b := body.(type) {
		case string:
			r = bytes.NewBufferString(b)
		default:
			buf, err := json.Marshal(body)
			if err != nil {
				t.Fatalf("marshal body: %v", err)
			}
			r = bytes.NewBuffer(buf)
		}
	}
	req := httptest.NewRequest(method, path, r)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	res, err := app.Test(req, -1)
	if err != nil {
		t.Fatalf("app.Test: %v", err)
	}
	return res
}

func decode[T any](t *testing.T, res *http.Response) T {
	t.Helper()
	var v T
	if err := json.NewDecoder(res.Body).Decode(&v); err != nil {
		t.Fatalf("decode: %v", err)
	}
	return v
}

func TestHealthz(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodGet, "/healthz", nil, nil)
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	got := decode[types.HealthResponse](t, res)
	if got.Status != "ok" {
		t.Errorf("status = %q, want ok", got.Status)
	}
}

func TestReadyz_MemoryStoreReturnsReady(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodGet, "/readyz", nil, nil)
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	got := decode[types.HealthResponse](t, res)
	if got.Status != "ready" {
		t.Errorf("status = %q, want ready", got.Status)
	}
}

// brokenStore implements PackStore + Pinger and always reports failure.
type brokenStore struct{}

func (brokenStore) Get(context.Context) (store.Snapshot, error) {
	return store.Snapshot{Sizes: store.DefaultSizes, Version: "v"}, nil
}
func (brokenStore) Set(context.Context, []int, string) (store.Snapshot, error) {
	return store.Snapshot{}, errors.New("not allowed")
}
func (brokenStore) Ping(context.Context) error { return errors.New("redis down") }

func TestReadyz_BrokenStoreReturns503(t *testing.T) {
	app := newAppWithStore(t, brokenStore{})
	res := do(t, app, http.MethodGet, "/readyz", nil, nil)
	if res.StatusCode != 503 {
		t.Fatalf("status = %d, want 503", res.StatusCode)
	}
	got := decode[types.ErrorResponse](t, res)
	if got.Code != CodeNotReady {
		t.Errorf("code = %q, want %q", got.Code, CodeNotReady)
	}
	if got.RequestId == "" {
		t.Errorf("request_id missing")
	}
}

func TestGetPacks_ReturnsDefaultsAndETag(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodGet, "/api/v1/packs", nil, nil)
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	etag := res.Header.Get(fiber.HeaderETag)
	if etag == "" {
		t.Errorf("ETag header missing")
	}
	got := decode[types.PackSet](t, res)
	if got.Version != etag {
		t.Errorf("body version %q != ETag %q", got.Version, etag)
	}
	// Defaults sorted DESC.
	want := append([]int(nil), store.DefaultSizes...)
	sort.Sort(sort.Reverse(sort.IntSlice(want)))
	if !reflect.DeepEqual(got.Sizes, want) {
		t.Errorf("sizes = %v, want %v", got.Sizes, want)
	}
}

func TestPutPacks_HappyPath(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodPut, "/api/v1/packs",
		map[string]any{"sizes": []int{100, 50, 200}}, nil)
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if res.Header.Get(fiber.HeaderETag) == "" {
		t.Errorf("ETag missing")
	}
	got := decode[types.PackSet](t, res)
	if !reflect.DeepEqual(got.Sizes, []int{200, 100, 50}) {
		t.Errorf("sizes = %v, want [200 100 50]", got.Sizes)
	}
}

func TestPutPacks_StaleIfMatchReturns412(t *testing.T) {
	app, _ := newApp(t)
	// Read current version.
	res := do(t, app, http.MethodGet, "/api/v1/packs", nil, nil)
	stale := decode[types.PackSet](t, res).Version

	// Bump.
	_ = do(t, app, http.MethodPut, "/api/v1/packs",
		map[string]any{"sizes": []int{1, 2, 3}}, nil)

	// Stale If-Match.
	res = do(t, app, http.MethodPut, "/api/v1/packs",
		map[string]any{"sizes": []int{9}},
		map[string]string{"If-Match": stale})
	if res.StatusCode != 412 {
		t.Fatalf("status = %d, want 412", res.StatusCode)
	}
	got := decode[types.ErrorResponse](t, res)
	if got.Code != CodeVersionMismatch {
		t.Errorf("code = %q, want %q", got.Code, CodeVersionMismatch)
	}
}

func TestPutPacks_CurrentIfMatchSucceeds(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodGet, "/api/v1/packs", nil, nil)
	current := decode[types.PackSet](t, res).Version

	res = do(t, app, http.MethodPut, "/api/v1/packs",
		map[string]any{"sizes": []int{42}},
		map[string]string{"If-Match": current})
	if res.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", res.StatusCode)
	}
}

func TestPutPacks_ValidationErrors(t *testing.T) {
	cases := []struct {
		name       string
		body       any
		wantStatus int
		wantCode   string
	}{
		{"empty sizes", map[string]any{"sizes": []int{}}, 400, CodeInvalidRequest},
		{"zero size", map[string]any{"sizes": []int{0, 100}}, 400, CodeInvalidRequest},
		{"negative size", map[string]any{"sizes": []int{-1, 100}}, 400, CodeInvalidRequest},
		{"duplicate", map[string]any{"sizes": []int{250, 250}}, 400, CodeInvalidRequest},
		{"size over limit", map[string]any{"sizes": []int{20_000_000}}, 422, CodeLimitExceeded},
		{"malformed json", "{not json", 400, CodeInvalidRequest},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			app, _ := newApp(t)
			res := do(t, app, http.MethodPut, "/api/v1/packs", c.body, nil)
			if res.StatusCode != c.wantStatus {
				t.Fatalf("status = %d, want %d", res.StatusCode, c.wantStatus)
			}
			got := decode[types.ErrorResponse](t, res)
			if got.Code != c.wantCode {
				t.Errorf("code = %q, want %q", got.Code, c.wantCode)
			}
			if got.RequestId == "" {
				t.Errorf("request_id missing")
			}
		})
	}
}

func TestCalculate_ReviewerExamples(t *testing.T) {
	cases := []struct {
		order      int
		wantItems  int
		wantPacks  int
		wantCounts map[int]int
	}{
		{1, 250, 1, map[int]int{250: 1}},
		{250, 250, 1, map[int]int{250: 1}},
		{251, 500, 1, map[int]int{500: 1}},
		{501, 750, 2, map[int]int{500: 1, 250: 1}},
		{12001, 12250, 4, map[int]int{5000: 2, 2000: 1, 250: 1}},
	}
	for _, c := range cases {
		c := c
		t.Run("", func(t *testing.T) {
			app, _ := newApp(t)
			res := do(t, app, http.MethodPost, "/api/v1/calculate",
				map[string]any{"order": c.order}, nil)
			if res.StatusCode != 200 {
				t.Fatalf("status = %d", res.StatusCode)
			}
			got := decode[types.CalculateResponse](t, res)
			if got.TotalItems != c.wantItems {
				t.Errorf("total_items = %d, want %d", got.TotalItems, c.wantItems)
			}
			if got.TotalPacks != c.wantPacks {
				t.Errorf("total_packs = %d, want %d", got.TotalPacks, c.wantPacks)
			}
			if got.Overshoot != c.wantItems-c.order {
				t.Errorf("overshoot = %d, want %d", got.Overshoot, c.wantItems-c.order)
			}
			gotCounts := map[int]int{}
			for _, p := range got.Packs {
				gotCounts[p.Size] = p.Count
			}
			if !reflect.DeepEqual(gotCounts, c.wantCounts) {
				t.Errorf("packs = %v, want %v", gotCounts, c.wantCounts)
			}
		})
	}
}

func TestCalculate_ValidationErrors(t *testing.T) {
	cases := []struct {
		name       string
		body       any
		wantStatus int
		wantCode   string
	}{
		{"negative order", map[string]any{"order": -1}, 400, CodeInvalidRequest},
		{"order over limit", map[string]any{"order": 20_000_000}, 422, CodeLimitExceeded},
		{"malformed json", "{bad", 400, CodeInvalidRequest},
	}
	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			app, _ := newApp(t)
			res := do(t, app, http.MethodPost, "/api/v1/calculate", c.body, nil)
			if res.StatusCode != c.wantStatus {
				t.Fatalf("status = %d, want %d", res.StatusCode, c.wantStatus)
			}
			got := decode[types.ErrorResponse](t, res)
			if got.Code != c.wantCode {
				t.Errorf("code = %q, want %q", got.Code, c.wantCode)
			}
		})
	}
}

func TestCalculate_ZeroOrderReturnsEmptyResult(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodPost, "/api/v1/calculate",
		map[string]any{"order": 0}, nil)
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	got := decode[types.CalculateResponse](t, res)
	if got.TotalItems != 0 || got.TotalPacks != 0 || got.Overshoot != 0 || len(got.Packs) != 0 {
		t.Errorf("zero order should yield empty result, got %+v", got)
	}
}

func TestRequestID_GeneratedWhenAbsent(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodGet, "/healthz", nil, nil)
	rid := res.Header.Get("X-Request-Id")
	if rid == "" {
		t.Fatalf("server did not stamp X-Request-Id")
	}
}

func TestRequestID_HonoredWhenSupplied(t *testing.T) {
	app, _ := newApp(t)
	want := "test-rid-abc"
	res := do(t, app, http.MethodGet, "/healthz", nil,
		map[string]string{"X-Request-Id": want})
	if got := res.Header.Get("X-Request-Id"); got != want {
		t.Errorf("server did not echo client-supplied X-Request-Id: got %q", got)
	}
}

func TestErrorResponse_CarriesRequestID(t *testing.T) {
	app, _ := newApp(t)
	want := "rid-error-roundtrip"
	res := do(t, app, http.MethodPost, "/api/v1/calculate",
		map[string]any{"order": -1},
		map[string]string{"X-Request-Id": want})
	if res.StatusCode != 400 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	got := decode[types.ErrorResponse](t, res)
	if got.RequestId != want {
		t.Errorf("request_id = %q, want %q", got.RequestId, want)
	}
}

func TestUnknownRoute_Returns404Envelope(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodGet, "/nope", nil, nil)
	if res.StatusCode != 404 {
		t.Fatalf("status = %d, want 404", res.StatusCode)
	}
	got := decode[types.ErrorResponse](t, res)
	if got.Code != CodeInvalidRequest {
		t.Errorf("code = %q, want %q", got.Code, CodeInvalidRequest)
	}
	if got.RequestId == "" {
		t.Errorf("request_id missing on 404")
	}
}
