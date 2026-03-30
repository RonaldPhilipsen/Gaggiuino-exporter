package gaggiuino

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"golang.org/x/crypto/bcrypt"
)

var (
	gaggiuino_up = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_up",
			Help: "Whether the Gaggiuino API is reachable",
		},
	)
	gaggiuino_uptime_seconds = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_uptime_seconds",
			Help: "Uptime of the espresso machine",
		},
	)
	gaggiuino_profile_id = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_profile_id",
			Help: "Current profile ID",
		},
	)
	gaggiuino_target_temperature = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_target_temperature",
			Help: "Target temperature",
		},
	)
	gaggiuino_temperature = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_temperature",
			Help: "Current temperature of the boiler",
		},
	)
	gaggiuino_pressure_bar = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_pressure_bar",
			Help: "Current pressure measured at the boiler",
		},
	)
	gaggiuino_water_level = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_water_level",
			Help: "Current water level as measured by the ultrasonic sensor",
		},
	)
	gaggiuino_shot_weight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "gaggiuino_shot_weight",
			Help: "Current weight of the espresso shot as measured by the scale",
		},
	)
	collectors = []prometheus.Collector{
		gaggiuino_up,
		gaggiuino_uptime_seconds,
		gaggiuino_profile_id,
		gaggiuino_target_temperature,
		gaggiuino_temperature,
		gaggiuino_pressure_bar,
		gaggiuino_water_level,
		gaggiuino_shot_weight,
	}
)

func init() {
	prometheus.MustRegister(collectors...)
}

func NewExporter(baseURL string, basicAuth map[string]string) *Exporter {
	return &Exporter{
		baseURL: baseURL,
		basicAuth: basicAuth,
	}
}

// Exporter is the type to be used to start HTTP server and run the analysis
type Exporter struct {
	baseURL		string
	basicAuth   map[string]string
}


func (e *Exporter) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	if len(e.basicAuth) > 0 {
		if ok := e.authorizeReq(w, req); !ok {
			return
		}
	}

	state, err := GetState(e.baseURL)
	if err != nil {
		gaggiuino_up.Set(0)
		log.Printf("failed to get state from %s: %v", e.baseURL, err)
		promhttp.Handler().ServeHTTP(w, req)
		return
	}
	gaggiuino_up.Set(1)
	gaggiuino_uptime_seconds.Set(float64(state.Uptime))
	gaggiuino_profile_id.Set(float64(state.ProfileId))
	gaggiuino_target_temperature.Set(state.TargetTemperature)
	gaggiuino_temperature.Set(state.Temperature)
	gaggiuino_pressure_bar.Set(state.Pressure)
	gaggiuino_water_level.Set(float64(state.WaterLevel))
	gaggiuino_shot_weight.Set(state.Weight)

	promhttp.Handler().ServeHTTP(w, req)
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

	w.Header().Add("WWW-Authenticate", "Basic realm=\"Access to Gaggiuino exporter\"")
	w.WriteHeader(401)
	return false
}

// RunServer starts HTTP server loop
func (e *Exporter) RunServer(addr string) {
	http.Handle("/", http.HandlerFunc(ServeIndex))
	http.Handle("/metrics", e)

	log.Printf("Providing metrics at http://%s/metrics", addr)
	err := http.ListenAndServe(addr, nil)
	if err != nil {
		log.Fatal("ListenAndServe:", err)
	}
}

// ServeIndex serves index page
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