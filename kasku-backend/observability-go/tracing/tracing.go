// Package tracing menyediakan OpenTelemetry tracer provider setup yang dipakai
// bersama oleh semua service Go di KasKu.
//
// Pattern: panggil InitTracer("service-name", cfg) di main.go segera setelah
// config diload. Simpan returned ShutdownFunc dan panggil saat graceful shutdown.
// Jika OTLPEndpoint kosong, tracing didisable (noop) — tidak pernah crash.
package tracing

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
	"go.opentelemetry.io/otel/trace"
	nooptrace "go.opentelemetry.io/otel/trace/noop"
)

// Config menyimpan konfigurasi OTel yang dibaca dari env vars oleh service.
type Config struct {
	// OTLPEndpoint adalah gRPC endpoint OTEL Collector, mis. "otel-collector:4317".
	// Jika kosong, tracing didisable (noop tracer).
	OTLPEndpoint string
	// ServiceVersion diisi dari env SERVICE_VERSION.
	ServiceVersion string
	// Environment diisi dari env APP_ENV.
	Environment string
}

// ShutdownFunc adalah fungsi yang harus dipanggil saat service shutdown
// untuk flush pending spans sebelum proses berakhir.
type ShutdownFunc func(ctx context.Context) error

// InitTracer menginisialisasi OTel tracer provider global dan propagator.
// Mengembalikan ShutdownFunc yang harus dipanggil saat graceful shutdown.
//
// Jika cfg.OTLPEndpoint kosong, dipasang noop tracer — service berjalan normal
// tanpa mengirim data tracing. Tidak pernah return error fatal; error exporter
// init dikembalikan agar pemanggil bisa log warning dan lanjut.
func InitTracer(serviceName string, cfg Config) (ShutdownFunc, error) {
	noopShutdown := func(ctx context.Context) error { return nil }

	if cfg.OTLPEndpoint == "" {
		setNoop()
		return noopShutdown, nil
	}

	ctx := context.Background()
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(),
	)
	if err != nil {
		// Gagal buat exporter — fallback ke noop agar service tetap jalan
		setNoop()
		return noopShutdown, fmt.Errorf("gagal membuat OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(serviceName),
			semconv.ServiceVersion(cfg.ServiceVersion),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		res = resource.Default()
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp,
			sdktrace.WithBatchTimeout(5*time.Second),
			sdktrace.WithMaxExportBatchSize(512),
		),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return func(ctx context.Context) error {
		return tp.Shutdown(ctx)
	}, nil
}

// setNoop memasang noop TracerProvider global supaya kode downstream
// (otelgin, otelgrpc) tetap bisa dipanggil tanpa panic.
func setNoop() {
	otel.SetTracerProvider(nooptrace.NewTracerProvider())
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))
}

// Tracer mengembalikan named tracer dari global provider.
// Dipakai di usecase/handler jika perlu membuat child span manual.
func Tracer(name string) trace.Tracer {
	return otel.GetTracerProvider().Tracer(name)
}

// ConfigFromEnv membaca konfigurasi OTel dari standard env vars.
// Dipanggil dari masing-masing service di configs/config.go.
func ConfigFromEnv() Config {
	return Config{
		OTLPEndpoint:   os.Getenv("OTEL_EXPORTER_OTLP_ENDPOINT"),
		ServiceVersion: envOrDefault("SERVICE_VERSION", "1.0.0"),
		Environment:    envOrDefault("APP_ENV", "production"),
	}
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
