package gaggiuino

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"golang.org/x/crypto/bcrypt"
)

var (
	gaggiuinoUp = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_up",
			Help: "Whether the Gaggiuino API is reachable",
		},
	)
	gaggiuinoUptimeSeconds = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_uptime_seconds",
			Help: "Uptime of the espresso machine",
		},
	)
	gaggiuinoProfileID = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_profile_id",
			Help: "Current profile ID",
		},
	)
	gaggiuinoTargetTemperature = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_target_temperature",
			Help: "Target temperature",
		},
	)
	gaggiuinoTemperature = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_temperature",
			Help: "Current temperature of the boiler",
		},
	)
	gaggiuinoPressureBar = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_pressure_bar",
			Help: "Current pressure measured at the boiler",
		},
	)
	gaggiuinoWaterLevel = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_water_level",
			Help: "Current water level as measured by the ultrasonic sensor",
		},
	)
	gaggiuinoShotWeight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_shot_weight",
			Help: "Current weight of the espresso shot as measured by the scale",
		},
	)
	gaggiuinoLastShotID = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_last_shot_id",
			Help: "Last shot ID reported by the Gaggiuino API",
		},
	)
	prometheusCollectors = []prometheus.Collector{
		gaggiuinoUp,
		gaggiuinoUptimeSeconds,
		gaggiuinoProfileID,
		gaggiuinoTargetTemperature,
		gaggiuinoTemperature,
		gaggiuinoPressureBar,
		gaggiuinoWaterLevel,
		gaggiuinoShotWeight,
		gaggiuinoLastShotID,
	}
)

func init() {
	prometheus.MustRegister(prometheusCollectors...)
}

type prometheusBackend struct{}

func (p *prometheusBackend) Record(state backendState) {
	gaggiuinoUp.Set(state.Up)
	if state.Up <= 0 || state.Status == nil {
		Logger.Debug("prometheus record", "up", state.Up)
		return
	}

	gaggiuinoUptimeSeconds.Set(float64(state.Status.Uptime))
	gaggiuinoProfileID.Set(float64(state.Status.ProfileID))
	gaggiuinoTargetTemperature.Set(state.Status.TargetTemperature)
	gaggiuinoTemperature.Set(state.Status.Temperature)
	gaggiuinoPressureBar.Set(state.Status.Pressure)
	gaggiuinoWaterLevel.Set(float64(state.Status.WaterLevel))
	gaggiuinoShotWeight.Set(state.Status.Weight)

	if state.LastShotID != nil {
		gaggiuinoLastShotID.Set(float64(*state.LastShotID))
	}
	Logger.Debug("prometheus record", "up", state.Up, "status", state.Status, "lastShotID", state.LastShotID)
}

// ServeHTTP handles Prometheus scrape requests at /metrics.
func (e *Exporter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	Logger.Debug("scrape request", "remoteAddr", req.RemoteAddr)
	if len(e.basicAuth) > 0 {
		if ok := e.authorizeReq(w, req); !ok {
			return
		}
	}

	if err := e.updateMachineMetrics(); err != nil {
		e.handleStateTransition(err, "prometheus-scrape")
		e.handler.ServeHTTP(w, req)
		return
	}

	e.handleStateTransition(nil, "prometheus-scrape")

	e.handler.ServeHTTP(w, req)
}

func (e *Exporter) authorizeReq(w http.ResponseWriter, req *http.Request) bool {
	user, pass, ok := req.BasicAuth()
	if ok {
		if hashed, found := e.basicAuth[user]; found {
			err := bcrypt.CompareHashAndPassword([]byte(hashed), []byte(pass))
			if err == nil {
				return true
			}
		}
	}

	Logger.Debug("unauthorized scrape request", "remoteAddr", req.RemoteAddr, "user", user)
	w.Header().Add("WWW-Authenticate", "Basic realm=\"Access to Gaggiuino exporter\"")
	w.WriteHeader(http.StatusUnauthorized)
	return false
}

// RunServer starts HTTP server loop.
func (e *Exporter) RunServer(addr string) {
	if e.otlp != nil {
		go e.runOTLPPolling()
	}

	if e.ws != nil {
		go e.ws.run(context.Background())
	}

	http.Handle("/", http.HandlerFunc(ServeIndex))
	http.Handle("/metrics", e)

	Logger.Info("providing metrics", "addr", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		Logger.Error("ListenAndServe failed", "error", err)
		os.Exit(1)
	}
}

// ServeIndex serves index page.
func ServeIndex(w http.ResponseWriter, req *http.Request) {
	w.Header().Set("Content-type", "text/html")
	res := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="utf-8">
	<meta name="viewport" content="width=device-width">
	<title>Gaggiuino Prometheus Exporter</title>
</head>
<body>
<h1>Gaggiuino Prometheus Exporter</h1>
<p>
	<a href="/metrics">Metrics</a>
</p>
<p>
	<a href="https://github.com/RonaldPhilipsen/Gaggiuino_exporter">Homepage</a>
</p>
</body>
</html>
`
	fmt.Fprint(w, res)
}
