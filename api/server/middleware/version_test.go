package middleware

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/docker/docker/api/server/httputils"
	"github.com/docker/docker/pkg/version"
	"golang.org/x/net/context"
)

func TestVersionMiddleware(t *testing.T) {
	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		if httputils.VersionFromContext(ctx) == "" {
			t.Fatalf("Expected version, got empty string")
		}
		return nil
	}

	defaultVersion := version.Version("1.10.0")
	minVersion := version.Version("1.2.0")
	m := NewVersionMiddleware(defaultVersion, defaultVersion, minVersion)
	h := m.WrapHandler(handler)

	req, _ := http.NewRequest("GET", "/containers/json", nil)
	resp := httptest.NewRecorder()
	ctx := context.Background()
	if err := h(ctx, resp, req, map[string]string{}); err != nil {
		t.Fatal(err)
	}
}

func TestVersionMiddlewareWithErrors(t *testing.T) {
	handler := func(ctx context.Context, w http.ResponseWriter, r *http.Request, vars map[string]string) error {
		if httputils.VersionFromContext(ctx) == "" {
			t.Fatalf("Expected version, got empty string")
		}
		return nil
	}

	defaultVersion := version.Version("1.10.0")
	minVersion := version.Version("1.2.0")
	m := NewVersionMiddleware(defaultVersion, defaultVersion, minVersion)
	h := m.WrapHandler(handler)

	req, _ := http.NewRequest("GET", "/containers/json", nil)
	resp := httptest.NewRecorder()
	ctx := context.Background()

	vars := map[string]string{"version": "0.1"}
	err := h(ctx, resp, req, vars)

	if !strings.Contains(err.Error(), "client version 0.1 is too old. Minimum supported API version is 1.2.0") {
		t.Fatalf("Expected too old client error, got %v", err)
	}

	vars["version"] = "100000"
	err = h(ctx, resp, req, vars)
	if !strings.Contains(err.Error(), "client is newer than server") {
		t.Fatalf("Expected client newer than server error, got %v", err)
	}
}
