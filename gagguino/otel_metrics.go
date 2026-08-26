package gaggiuino

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
)

type OTLPOptions struct {
	Enabled  bool
	Endpoint string
	Interval time.Duration
	Timeout  time.Duration
	Headers  map[string]string
}

type otlpMetrics struct {
	provider          *sdkmetric.MeterProvider
	up                metric.Float64Gauge
	uptimeSeconds     metric.Float64Gauge
	profileID         metric.Float64Gauge
	targetTemperature metric.Float64Gauge
	temperature       metric.Float64Gauge
	pressureBar       metric.Float64Gauge
	waterLevel        metric.Float64Gauge
	shotWeight        metric.Float64Gauge
	lastShotID        metric.Int64Gauge
}

type otlpBackend struct {
	otlp *otlpMetrics
}

func (b *otlpBackend) Record(state backendState) {
	if b.otlp == nil {
		return
	}

	ctx := context.Background()
	b.otlp.up.Record(ctx, state.Up)
	if state.Up <= 0 || state.Status == nil {
		return
	}

	b.otlp.uptimeSeconds.Record(ctx, float64(state.Status.Uptime))
	b.otlp.profileID.Record(ctx, float64(state.Status.ProfileID))
	b.otlp.targetTemperature.Record(ctx, state.Status.TargetTemperature)
	b.otlp.temperature.Record(ctx, state.Status.Temperature)
	b.otlp.pressureBar.Record(ctx, state.Status.Pressure)
	b.otlp.waterLevel.Record(ctx, float64(state.Status.WaterLevel))
	b.otlp.shotWeight.Record(ctx, state.Status.Weight)

	if state.LastShotID != nil {
		b.otlp.lastShotID.Record(ctx, *state.LastShotID)
	}
}

func newOTLPMetrics(opts OTLPOptions) (*otlpMetrics, error) {
	endpointURL, err := normalizeEndpointURL(opts.Endpoint)
	if err != nil {
		return nil, err
	}

	u, err := url.Parse(endpointURL)
	if err != nil {
		return nil, fmt.Errorf("invalid otlp endpoint %q: %w", opts.Endpoint, err)
	}

	if opts.Interval <= 0 {
		opts.Interval = 15 * time.Second
	}
	if opts.Timeout <= 0 {
		opts.Timeout = 10 * time.Second
	}

	exporterOpts := []otlpmetrichttp.Option{
		otlpmetrichttp.WithEndpoint(u.Host),
		otlpmetrichttp.WithURLPath(u.EscapedPath()),
		otlpmetrichttp.WithTimeout(opts.Timeout),
	}
	if strings.EqualFold(u.Scheme, "http") {
		exporterOpts = append(exporterOpts, otlpmetrichttp.WithInsecure())
	}
	if len(opts.Headers) > 0 {
		exporterOpts = append(exporterOpts, otlpmetrichttp.WithHeaders(opts.Headers))
	}

	ctx, cancel := context.WithTimeout(context.Background(), opts.Timeout)
	defer cancel()

	exporter, err := otlpmetrichttp.New(ctx, exporterOpts...)
	if err != nil {
		return nil, err
	}

	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			attribute.String("service.name", "gaggiuino-exporter"),
			attribute.String("exporter", "gaggiuino"),
		),
	)
	if err != nil {
		return nil, err
	}

	reader := sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(opts.Interval))
	provider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(reader),
		sdkmetric.WithResource(res),
	)
	meter := provider.Meter("github.com/RonaldPhilipsen/gaggiuino-exporter")

	up, err := meter.Float64Gauge("gaggiuino_up")
	if err != nil {
		return nil, err
	}
	uptimeSeconds, err := meter.Float64Gauge("gaggiuino_uptime_seconds")
	if err != nil {
		return nil, err
	}
	profileID, err := meter.Float64Gauge("gaggiuino_profile_id")
	if err != nil {
		return nil, err
	}
	targetTemperature, err := meter.Float64Gauge("gaggiuino_target_temperature")
	if err != nil {
		return nil, err
	}
	temperature, err := meter.Float64Gauge("gaggiuino_temperature")
	if err != nil {
		return nil, err
	}
	pressureBar, err := meter.Float64Gauge("gaggiuino_pressure_bar")
	if err != nil {
		return nil, err
	}
	waterLevel, err := meter.Float64Gauge("gaggiuino_water_level")
	if err != nil {
		return nil, err
	}
	shotWeight, err := meter.Float64Gauge("gaggiuino_shot_weight")
	if err != nil {
		return nil, err
	}
	lastShotID, err := meter.Int64Gauge("gaggiuino_last_shot_id")
	if err != nil {
		return nil, err
	}

	return &otlpMetrics{
		provider:          provider,
		up:                up,
		uptimeSeconds:     uptimeSeconds,
		profileID:         profileID,
		targetTemperature: targetTemperature,
		temperature:       temperature,
		pressureBar:       pressureBar,
		waterLevel:        waterLevel,
		shotWeight:        shotWeight,
		lastShotID:        lastShotID,
	}, nil
}

func normalizeEndpointURL(endpoint string) (string, error) {
	if endpoint == "" {
		return "", fmt.Errorf("otlp endpoint cannot be empty")
	}

	if strings.Contains(endpoint, "://") {
		return endpoint, nil
	}

	return "http://" + endpoint, nil
}
