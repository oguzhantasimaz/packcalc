package http

import (
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestOpenAPISpecServed(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodGet, "/openapi.yaml", nil, nil)
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	if ct := res.Header.Get("Content-Type"); !strings.Contains(ct, "yaml") {
		t.Errorf("content-type = %q, want yaml-ish", ct)
	}
	body, err := io.ReadAll(res.Body)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(body), "openapi: 3.1.0") {
		t.Errorf("body does not contain openapi marker; first bytes: %q", trimFirst(body, 80))
	}
}

func TestSwaggerUIIndexServed(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodGet, "/docs/", nil, nil)
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	got := string(body)
	if !strings.Contains(got, "swagger-ui") {
		t.Errorf("index missing swagger-ui marker; first bytes: %q", trimFirst(body, 120))
	}
	if !strings.Contains(got, "/openapi.yaml") {
		t.Errorf("index does not reference /openapi.yaml")
	}
}

func TestSwaggerUIAssetServed(t *testing.T) {
	app, _ := newApp(t)
	res := do(t, app, http.MethodGet, "/docs/swagger-ui.css", nil, nil)
	if res.StatusCode != 200 {
		t.Fatalf("status = %d", res.StatusCode)
	}
	body, _ := io.ReadAll(res.Body)
	if len(body) < 1024 {
		t.Errorf("css too small: %d bytes", len(body))
	}
}

func trimFirst(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n])
}
