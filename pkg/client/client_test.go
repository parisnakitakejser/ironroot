package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

func TestTraceContextPropagation(t *testing.T) {
	otel.SetTextMapPropagator(propagation.TraceContext{})
	tp := sdktrace.NewTracerProvider(sdktrace.WithSampler(sdktrace.AlwaysSample()))
	defer func() { _ = tp.Shutdown(context.Background()) }()
	otel.SetTracerProvider(tp)
	seen := ""
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Header.Get("traceparent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"enrollment_id":"enr"}`))
	}))
	defer srv.Close()

	ctx, span := otel.Tracer("test").Start(context.Background(), "command")
	defer span.End()
	_, err := New(srv.URL).Enroll(ctx, EnrollmentRequest{Token: "t", Hostname: "h", MachineID: "m"})
	if err != nil {
		t.Fatal(err)
	}
	if seen == "" {
		t.Fatal("missing traceparent header")
	}
}
