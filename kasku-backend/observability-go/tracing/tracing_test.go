package tracing_test

import (
	"context"
	"testing"
	"time"

	"github.com/TubagusAldiMY/kasku/observability-go/tracing"
	"go.opentelemetry.io/otel"
)

func TestInitTracer_NoopWhenEndpointEmpty(t *testing.T) {
	t.Parallel()
	shutdown, err := tracing.InitTracer("test-service", tracing.Config{
		OTLPEndpoint: "",
	})
	if err != nil {
		t.Fatalf("InitTracer noop harus tidak error: %v", err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	// Tracer global harus ter-set (noop provider)
	tracer := otel.GetTracerProvider().Tracer("test")
	if tracer == nil {
		t.Fatal("tracer global tidak ter-set")
	}
}

func TestInitTracer_FailsGracefullyOnBadEndpoint(t *testing.T) {
	t.Parallel()
	// Endpoint invalid — harus return error tapi tidak panic.
	// OTLP gRPC exporter adalah async; koneksi tidak dibuat saat New() dipanggil,
	// sehingga err bisa nil meski endpoint tidak reachable. Yang penting: tidak
	// panic dan shutdown bisa dipanggil dengan aman.
	shutdown, err := tracing.InitTracer("test-service", tracing.Config{
		OTLPEndpoint:   "invalid-host:9999",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	})
	if shutdown == nil {
		t.Fatal("shutdown func harus tidak nil")
	}
	_ = err // error boleh nil (async exporter) atau non-nil
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	_ = shutdown(ctx)
}

func TestConfigFromEnv_DefaultValues(t *testing.T) {
	// t.Setenv tidak boleh dipakai bersama t.Parallel() — env test dijalankan serial
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "")
	t.Setenv("SERVICE_VERSION", "")
	t.Setenv("APP_ENV", "")

	cfg := tracing.ConfigFromEnv()
	if cfg.OTLPEndpoint != "" {
		t.Errorf("endpoint harus kosong, dapat: %q", cfg.OTLPEndpoint)
	}
	if cfg.ServiceVersion != "1.0.0" {
		t.Errorf("default version harus 1.0.0, dapat: %q", cfg.ServiceVersion)
	}
	if cfg.Environment != "production" {
		t.Errorf("default env harus production, dapat: %q", cfg.Environment)
	}
}

func TestConfigFromEnv_CustomValues(t *testing.T) {
	// t.Setenv tidak boleh dipakai bersama t.Parallel() — env test dijalankan serial
	t.Setenv("OTEL_EXPORTER_OTLP_ENDPOINT", "otel-collector:4317")
	t.Setenv("SERVICE_VERSION", "2.3.1")
	t.Setenv("APP_ENV", "staging")

	cfg := tracing.ConfigFromEnv()
	if cfg.OTLPEndpoint != "otel-collector:4317" {
		t.Errorf("endpoint salah, dapat: %q", cfg.OTLPEndpoint)
	}
	if cfg.ServiceVersion != "2.3.1" {
		t.Errorf("version salah, dapat: %q", cfg.ServiceVersion)
	}
	if cfg.Environment != "staging" {
		t.Errorf("env salah, dapat: %q", cfg.Environment)
	}
}

func TestTracer_ReturnsNonNil(t *testing.T) {
	t.Parallel()
	// Pastikan Tracer() helper tidak panic dan return non-nil
	tr := tracing.Tracer("my-service")
	if tr == nil {
		t.Fatal("Tracer() harus mengembalikan non-nil tracer")
	}
}
