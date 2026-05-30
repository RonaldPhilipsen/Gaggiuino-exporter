package gaggiuino

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestNormalizeEndpointURL(t *testing.T) {
	tests := []struct {
		name        string
		endpoint    string
		expected    string
		errContains string
	}{
		{
			name:        "empty endpoint returns error",
			endpoint:    "",
			errContains: "cannot be empty",
		},
		{
			name:     "adds http scheme when missing",
			endpoint: "vmagent:8429/opentelemetry/v1/metrics",
			expected: "http://vmagent:8429/opentelemetry/v1/metrics",
		},
		{
			name:     "keeps existing scheme",
			endpoint: "https://vmagent:8429/opentelemetry/v1/metrics",
			expected: "https://vmagent:8429/opentelemetry/v1/metrics",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := normalizeEndpointURL(tt.endpoint)
			if tt.errContains != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.errContains)
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Fatalf("expected error containing %q, got %q", tt.errContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestNewOTLPMetrics(t *testing.T) {
	t.Run("returns error for invalid endpoint", func(t *testing.T) {
		t.Parallel()
		_, err := newOTLPMetrics(OTLPOptions{Endpoint: "://bad"})
		if err == nil {
			t.Fatalf("expected error, got nil")
		}
		if !strings.Contains(err.Error(), "invalid otlp endpoint") {
			t.Fatalf("expected invalid endpoint error, got %q", err.Error())
		}
	})

	t.Run("creates metrics and records values", func(t *testing.T) {
		t.Parallel()

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer srv.Close()

		m, err := newOTLPMetrics(OTLPOptions{
			Endpoint: srv.URL + "/opentelemetry/v1/metrics",
			Interval: 10 * time.Millisecond,
			Timeout:  2 * time.Second,
		})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		defer func() {
			_ = m.provider.Shutdown(context.Background())
		}()

		b := &otlpBackend{otlp: m}
		id := int64(3)
		st := status{
			Uptime:            12,
			ProfileID:         1,
			TargetTemperature: 95.0,
			Temperature:       93.3,
			Pressure:          8.9,
			WaterLevel:        20,
			Weight:            18.2,
		}

		b.Record(backendState{Up: 1, Status: &st, LastShotID: &id})
	})
}
