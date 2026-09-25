package netmon

import (
	"context"
	"fmt"
	"log"
	"runtime"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type Config struct {
	Endpoint string
	Interval time.Duration
}

type Monitor struct {
	Config Config
}

func (monitor Monitor) Run(ctx context.Context) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("network monitoring requires macOS nettop")
	}
	if monitor.Config.Interval <= 0 {
		return fmt.Errorf("interval must be positive")
	}

	exporter, err := otlpmetrichttp.New(ctx, otlpmetrichttp.WithEndpointURL(monitor.Config.Endpoint))
	if err != nil {
		return fmt.Errorf("create OTLP exporter: %w", err)
	}
	res, err := resource.New(ctx, resource.WithAttributes(attribute.String("service.name", "codex-netmon")))
	if err != nil {
		return fmt.Errorf("create OTel resource: %w", err)
	}
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(monitor.Config.Interval))),
	)
	defer func() {
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := provider.Shutdown(shutdownCtx); err != nil {
			log.Printf("OTLP shutdown: %v", err)
		}
	}()
	meter := provider.Meter("github.com/yudai2929/codex-netmon")
	received, err := meter.Int64Counter("codex.wifi.received", metric.WithUnit("By"), metric.WithDescription("Wi-Fi bytes received by Codex processes"))
	if err != nil {
		return err
	}
	sent, err := meter.Int64Counter("codex.wifi.sent", metric.WithUnit("By"), metric.WithDescription("Wi-Fi bytes sent by Codex processes"))
	if err != nil {
		return err
	}

	sampler := nettopSampler{timeout: 15 * time.Second}
	previous, err := sampler.read(ctx)
	if err != nil {
		return err
	}
	for _, sample := range previous.Samples {
		attrs := sampleAttributes(sample)
		received.Add(ctx, 0, attrs)
		sent.Add(ctx, 0, attrs)
	}
	log.Printf("monitoring Wi-Fi traffic for %d Codex and child processes", len(previous.Samples))
	ticker := time.NewTicker(monitor.Config.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			current, err := sampler.read(ctx)
			if err != nil {
				log.Printf("nettop sample: %v", err)
				continue
			}
			for pid, sample := range current.Samples {
				if old, found := previous.Samples[pid]; !found || old.Name != sample.Name || old.Kind != sample.Kind {
					attrs := sampleAttributes(sample)
					received.Add(ctx, 0, attrs)
					sent.Add(ctx, 0, attrs)
				}
			}
			for _, delta := range sampleDelta(previous, current) {
				attrs := sampleAttributes(delta)
				if delta.Received > 0 && delta.Received <= (1<<63-1) {
					received.Add(ctx, int64(delta.Received), attrs)
				}
				if delta.Sent > 0 && delta.Sent <= (1<<63-1) {
					sent.Add(ctx, int64(delta.Sent), attrs)
				}
			}
			previous = current
		}
	}
}

func sampleAttributes(sample processSample) metric.MeasurementOption {
	return metric.WithAttributes(
		attribute.String("process", sample.Name),
		attribute.String("process_kind", string(sample.Kind)),
	)
}
