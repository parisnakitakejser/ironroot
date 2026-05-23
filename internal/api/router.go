package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"

	"github.com/ironroot/ironroot/internal/audit"
	"github.com/ironroot/ironroot/internal/ca"
	"github.com/ironroot/ironroot/internal/config"
	"github.com/ironroot/ironroot/internal/db"
	"github.com/ironroot/ironroot/internal/telemetry"
)

type Dependencies struct {
	Config    config.Config
	Store     db.Store
	Authority ca.Authority
	Audit     audit.Logger
	Logger    *slog.Logger
}

func NewRouter(dep Dependencies) http.Handler {
	h := handler{dep: dep}
	r := chi.NewRouter()
	r.Use(metricsMiddleware)
	r.Use(otelhttp.NewMiddleware("ironroot-api"))
	r.Get("/healthz", h.healthz)
	r.Get("/readyz", h.readyz)
	r.Get("/v1/ca/root", h.root)
	r.Get("/v1/ca/chain", h.chain)
	r.Post("/v1/enroll", h.enroll)
	r.Post("/v1/certificates/request", h.requestCertificate)
	r.Post("/v1/certificates/renew", h.renewCertificate)
	r.Post("/v1/certificates/revoke", h.revokeCertificate)
	r.Get("/v1/certificates/status/{serial}", h.certificateStatus)
	r.Get("/v1/audit", h.audit)
	return r
}

func metricsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &statusWriter{ResponseWriter: w, status: 200}
		next.ServeHTTP(rw, r)
		metrics := telemetry.Instruments()
		attrs := metricAttrs(r, rw.status)
		metrics.APIRequests.Add(r.Context(), 1, metric.WithAttributes(attrs...))
		metrics.APIRequestDuration.Record(r.Context(), time.Since(start).Seconds(), metric.WithAttributes(attrs...))
	})
}

func metricAttrs(r *http.Request, status int) []attribute.KeyValue {
	return []attribute.KeyValue{
		attribute.String("http.method", r.Method),
		attribute.String("http.route", r.URL.Path),
		attribute.Int("http.status_code", status),
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}
