package telemetry

import (
	"context"
	"testing"

	"github.com/ironroot/ironroot/internal/config"
)

func TestConfigureDisabled(t *testing.T) {
	shutdown, err := Configure(context.Background(), config.TelemetryConfig{Enabled: false}, "test")
	if err != nil {
		t.Fatal(err)
	}
	if err := shutdown(context.Background()); err != nil {
		t.Fatal(err)
	}
}
