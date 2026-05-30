package gaggiuino

import (
	"testing"

	io_prometheus_client "github.com/prometheus/client_model/go"
)

func readLastShotGauge(t *testing.T) float64 {
	t.Helper()
	m := &io_prometheus_client.Metric{}
	if err := gaggiuinoLastShotID.Write(m); err != nil {
		t.Fatalf("failed to read last shot metric: %v", err)
	}
	if m.Gauge == nil {
		t.Fatalf("last shot metric is not a gauge")
	}
	return m.Gauge.GetValue()
}

func TestPrometheusBackendPassesThroughLastShotID(t *testing.T) {
	b := &prometheusBackend{}
	st := &status{Uptime: 1}
	id := int64(42)

	b.Record(backendState{Up: 1, Status: st, LastShotID: &id})

	if got := readLastShotGauge(t); got != 42 {
		t.Fatalf("expected last shot gauge 42, got %v", got)
	}

	id = 41
	b.Record(backendState{Up: 1, Status: st, LastShotID: &id})

	if got := readLastShotGauge(t); got != 41 {
		t.Fatalf("expected last shot gauge 41, got %v", got)
	}
}

func TestPrometheusBackendLeavesLastShotUnchangedWhenMissing(t *testing.T) {
	b := &prometheusBackend{}
	st := &status{Uptime: 1}
	id := int64(7)

	b.Record(backendState{Up: 1, Status: st, LastShotID: &id})
	b.Record(backendState{Up: 1, Status: st, LastShotID: nil})

	if got := readLastShotGauge(t); got != 7 {
		t.Fatalf("expected last shot gauge to stay at 7, got %v", got)
	}
}
