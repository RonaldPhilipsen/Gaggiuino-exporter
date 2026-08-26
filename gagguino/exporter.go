package gaggiuino

import (
	"context"
	"log"
	"net/http"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type backendState struct {
	Up         float64
	Status     *status
	LastShotID *int64
}

type metricsBackend interface {
	Record(backendState)
}

func NewExporter(baseURL string, basicAuth map[string]string, otlpOptions OTLPOptions) *Exporter {
	e := &Exporter{
		baseURL:     baseURL,
		basicAuth:   basicAuth,
		otlpOptions: otlpOptions,
		handler:     promhttp.Handler(),
		backends:    []metricsBackend{&prometheusBackend{}},
		lastUpState: -1,
		ws:          newWSClient(baseURL),
	}

	if e.otlpOptions.Enabled {
		otlp, err := newOTLPMetrics(e.otlpOptions)
		if err != nil {
			log.Printf("failed to initialize OTLP metrics exporter: %v", err)
		} else {
			e.otlp = otlp
			e.backends = append(e.backends, &otlpBackend{otlp: otlp})
			log.Printf("OTLP metrics export enabled: endpoint=%s interval=%s", e.otlpOptions.Endpoint, e.otlpOptions.Interval)
		}
	}

	return e
}

func (e *Exporter) runOTLPPolling() {
	interval := e.otlpOptions.Interval
	if interval <= 0 {
		interval = 1 * time.Second
	}

	if err := e.updateMachineMetrics(); err != nil {
		e.handleStateTransition(err, "otlp-polling")
	} else {
		e.handleStateTransition(nil, "otlp-polling")
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for range ticker.C {
		if err := e.updateMachineMetrics(); err != nil {
			e.handleStateTransition(err, "otlp-polling")
			continue
		}
		e.handleStateTransition(nil, "otlp-polling")
	}
}

// Exporter is the type to be used to start HTTP server and run the analysis
type Exporter struct {
	baseURL     string
	basicAuth   map[string]string
	otlpOptions OTLPOptions
	handler     http.Handler
	otlp        *otlpMetrics
	backends    []metricsBackend
	lastUpState int32
	ws          *wsClient
}

// updateMachineMetrics refreshes and publishes the current machine state. It
// prefers the live WebSocket state when connected, falling back to scraping
// the REST API otherwise.
func (e *Exporter) updateMachineMetrics() error {
	if e.ws != nil && e.ws.isConnected() {
		if st, ok := e.ws.snapshot(); ok {
			e.publishState(backendState{Up: 1, Status: &st})
			return nil
		}
	}

	ctx := context.Background()
	state, err := GetStateWithContext(ctx, e.baseURL)
	if err != nil {
		return err
	}

	var lastShotIDPtr *int64

	lastShotID, err := GetLastShotWithContext(ctx, e.baseURL)
	if err != nil {
		log.Printf("failed to get last shot from %s: %v", e.baseURL, err)
	} else {
		id := int64(lastShotID)
		lastShotIDPtr = &id
	}

	e.publishState(backendState{
		Up:         1,
		Status:     &state,
		LastShotID: lastShotIDPtr,
	})

	return nil
}

func (e *Exporter) publishState(state backendState) {
	for _, backend := range e.backends {
		backend.Record(state)
	}
}

func (e *Exporter) logOnStateChange(up bool) bool {
	newState := int32(0)
	if up {
		newState = 1
	}

	oldState := atomic.LoadInt32(&e.lastUpState)
	if oldState == newState {
		return false
	}

	atomic.StoreInt32(&e.lastUpState, newState)
	if oldState == -1 && up {
		return false
	}

	return true
}

func (e *Exporter) handleStateTransition(err error, source string) {
	if err != nil {
		if e.logOnStateChange(false) {
			if source == "" {
				log.Printf("failed to get state from %s: %v", e.baseURL, err)
			} else {
				log.Printf("failed to get state from %s (%s): %v", e.baseURL, source, err)
			}
		}
		e.publishState(backendState{Up: 0})
		return
	}

	if e.logOnStateChange(true) {
		if source == "" {
			log.Printf("recovered connectivity to %s", e.baseURL)
		} else {
			log.Printf("recovered connectivity to %s (%s)", e.baseURL, source)
		}
	}
}
