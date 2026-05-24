package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/parisnakitakejser/ironroot/internal/config"
	"github.com/parisnakitakejser/ironroot/internal/telemetry"
)

func TestMetricsEndpointExposesAPIMetrics(t *testing.T) {
	cfg := config.Default()
	cfg.Telemetry.Enabled = false
	cfg.Telemetry.Prometheus.Enabled = true
	cfg.Telemetry.Prometheus.Path = "/metrics"

	shutdown, err := telemetry.Configure(context.Background(), cfg.Telemetry, "ironroot-test")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = shutdown(context.Background())
	})

	router := NewRouter(Dependencies{Config: cfg})
	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, "/metrics", nil))
	if rr.Code != http.StatusOK {
		t.Fatalf("metrics endpoint status = %d", rr.Code)
	}
	body := rr.Body.String()
	if !strings.Contains(body, "pki_api_requests") || !strings.Contains(body, `method="GET"`) {
		t.Fatalf("expected API metrics with method label, got:\n%s", body)
	}
}
