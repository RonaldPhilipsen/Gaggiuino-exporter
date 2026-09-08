package gaggiuino

import (
	"context"
	"fmt"
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
			Logger.Error("failed to initialize OTLP metrics exporter", "error", err)
		} else {
			e.otlp = otlp
			e.backends = append(e.backends, &otlpBackend{otlp: otlp})
			Logger.Info("OTLP metrics export enabled", "endpoint", e.otlpOptions.Endpoint, "interval", e.otlpOptions.Interval)
		}
	}

	return e
}

func (e *Exporter) runOTLPPolling() {
	interval := e.otlpOptions.Interval
	if interval <= 0 {
		interval = 1 * time.Second
	}
	Logger.Debug("starting otlp polling loop", "interval", interval)

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

// runOTLPStream registers callbacks on the WebSocket client so metrics are
// published as soon as new state arrives, instead of on a fixed ticker.
func (e *Exporter) runOTLPStream() {
	Logger.Debug("starting otlp websocket stream mode")

	e.ws.onStateUpdate(func(st status) {
		e.handleStateTransition(nil, "otlp-stream")
		e.publishState(backendState{Up: 1, Status: &st})
	})

	e.ws.onConnected(func(connected bool) {
		if connected {
			return
		}
		e.handleStateTransition(fmt.Errorf("websocket connection lost"), "otlp-stream")
		e.publishState(backendState{Up: 0})
	})
}

// Start begins background metric collection: the WebSocket connection and,
// if enabled, OTLP publishing (either ticker-polled or streamed from the
// WebSocket). It is independent of serving Prometheus scrape requests, so
// callers that only need scraping can still call it to populate metrics.
func (e *Exporter) Start(ctx context.Context) {
	if e.otlp != nil {
		if e.otlpOptions.Mode == OTLPModeImmediate {
			e.runOTLPStream()
		} else {
			go e.runOTLPPolling()
		}
	}

	if e.ws != nil {
		go e.ws.run(ctx)
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
			Logger.Debug("publishing state", "source", "websocket", "state", st)
			e.publishState(backendState{Up: 1, Status: &st})
			return nil
		}
		Logger.Debug("websocket connected but no state received yet, falling back to REST")
	}

	ctx := context.Background()
	state, err := GetStateWithContext(ctx, e.baseURL)
	if err != nil {
		return err
	}
	Logger.Debug("publishing state", "source", "rest", "state", state)

	var lastShotIDPtr *int64

	lastShotID, err := GetLastShotWithContext(ctx, e.baseURL)
	if err != nil {
		Logger.Warn("failed to get last shot", "baseURL", e.baseURL, "error", err)
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
				Logger.Error("failed to get state", "baseURL", e.baseURL, "error", err)
			} else {
				Logger.Error("failed to get state", "baseURL", e.baseURL, "source", source, "error", err)
			}
		} else {
			Logger.Debug("still failing to get state", "baseURL", e.baseURL, "source", source, "error", err)
		}
		e.publishState(backendState{Up: 0})
		return
	}

	if e.logOnStateChange(true) {
		if source == "" {
			Logger.Info("recovered connectivity", "baseURL", e.baseURL)
		} else {
			Logger.Info("recovered connectivity", "baseURL", e.baseURL, "source", source)
		}
	}
}
