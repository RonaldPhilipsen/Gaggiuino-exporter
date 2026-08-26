package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/RonaldPhilipsen/gaggiuino-exporter/build"
	gaggiuino "github.com/RonaldPhilipsen/gaggiuino-exporter/gagguino"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

type ExporterOptions struct {
	OTLP gaggiuino.OTLPOptions
}

const defaultBaseURL = "http://gaggiuino.local"

var rootCmd = &cobra.Command{
	Use:   "gaggiuino_exporter",
	Short: "Prometheus exporter for Gaggiuino espresso machine",
	Long:  `Prometheus exporter for Gaggiuino espresso machine.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		printHeader()

		gaggiuino.SetDebug(viper.GetBool("debug"))

		baseURL, err := getBaseURL()
		if err != nil {
			return err
		}
		gaggiuino.Logger.Debug("resolved config",
			"mode", viper.GetString("mode"),
			"bind-address", viper.GetString("bind-address"),
			"basic-auth-users", len(viper.GetStringMapString("basic-auth-users")),
		)
		ctx := context.Background()

		if viper.GetString("mode") == "state" {
			lastShotID, err := gaggiuino.GetLastShotWithContext(ctx, baseURL)
			if err != nil {
				return fmt.Errorf("get last shot ID: %w", err)
			}
			state, err := gaggiuino.GetStateWithContext(ctx, baseURL)
			if err != nil {
				return fmt.Errorf("get state: %w", err)
			}
			stateJSON, err := json.Marshal(state)
			if err != nil {
				return fmt.Errorf("marshal state: %w", err)
			}
			gaggiuino.Logger.Info("current state", "state", string(stateJSON), "lastShotID", lastShotID)
			return nil
		}
		// TODO: Log all shots to Postgres
		exporterOptions := ExporterOptions{
			OTLP: gaggiuino.OTLPOptions{
				Enabled:  viper.GetBool("otlp-enabled"),
				Endpoint: viper.GetString("otlp-endpoint"),
				Interval: viper.GetDuration("otlp-interval"),
				Timeout:  viper.GetDuration("otlp-timeout"),
				Headers:  getOTLPHeaders(),
			},
		}

		e := gaggiuino.NewExporter(
			baseURL,
			viper.GetStringMapString("basic-auth-users"),
			exporterOptions.OTLP,
		)
		e.RunServer(viper.GetString("bind-address"))
		return nil
	},
}

func init() {
	cobra.OnInitialize(initConfig)
	flags := rootCmd.PersistentFlags()
	flags.StringP("mode", "m", "http", "Expose method - either 'state' or 'http'")
	flags.StringP("bind-address", "b", "0.0.0.0:9995", "Address to bind to")
	flags.StringToString("basic-auth-users", map[string]string{}, "Basic Auth users and their passwords as bcrypt hashes")
	flags.Bool("debug", false, "Enable verbose debug logging (websocket messages, polling, decoded fields)")

	flags.Bool("otlp-enabled", false, "Enable OpenTelemetry OTLP metrics export")
	flags.String("otlp-endpoint", "http://localhost:8429/opentelemetry/v1/metrics", "OTLP HTTP endpoint")
	flags.Duration("otlp-interval", 15*time.Second, "Interval for OTLP metric export")
	flags.Duration("otlp-timeout", 10*time.Second, "Timeout for OTLP export requests")
	flags.StringToString("otlp-headers", map[string]string{}, "Headers for OTLP export requests")

	err := viper.BindPFlags(flags)
	if err != nil {
		gaggiuino.Logger.Error("error binding flags", "error", err)
		os.Exit(1)
	}

	bindEnvOrDie("mode", "MODE")
	bindEnvOrDie("bind-address", "BIND_ADDRESS")
	bindEnvOrDie("debug", "DEBUG")
	bindEnvOrDie("otlp-enabled", "OTLP_ENABLED")
	bindEnvOrDie("otlp-endpoint", "OTLP_ENDPOINT")
	bindEnvOrDie("otlp-interval", "OTLP_INTERVAL")
	bindEnvOrDie("otlp-timeout", "OTLP_TIMEOUT")
	bindEnvOrDie("otlp-headers", "OTLP_HEADERS")
}

// initConfig reads in  ENV variables if set.
func initConfig() {
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_", ".", "_"))
	viper.AutomaticEnv() // read in environment variables that match
}

func bindEnvOrDie(key string, env ...string) {
	args := append([]string{key}, env...)
	if err := viper.BindEnv(args...); err != nil {
		gaggiuino.Logger.Error("error binding env", "key", key, "error", err)
		os.Exit(1)
	}
}

func getOTLPHeaders() map[string]string {
	headers := viper.GetStringMapString("otlp-headers")
	if len(headers) > 0 {
		gaggiuino.Logger.Debug("otlp headers from structured config", "count", len(headers))
		return headers
	}

	raw := strings.TrimSpace(viper.GetString("otlp-headers"))
	if raw == "" {
		return map[string]string{}
	}

	headers, err := parseOTLPHeaders(raw)
	if err != nil {
		gaggiuino.Logger.Warn("invalid otlp headers format", "error", err)
		return map[string]string{}
	}

	gaggiuino.Logger.Debug("otlp headers parsed from string", "count", len(headers))
	return headers
}

func parseOTLPHeaders(raw string) (map[string]string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return map[string]string{}, nil
	}

	if strings.HasPrefix(trimmed, "{") {
		headers := map[string]string{}
		if err := json.Unmarshal([]byte(trimmed), &headers); err != nil {
			return nil, fmt.Errorf("expected JSON object or key=value pairs: %w", err)
		}
		return headers, nil
	}

	headers := map[string]string{}
	for _, part := range strings.Split(trimmed, ",") {
		kv := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(kv) != 2 {
			return nil, fmt.Errorf("invalid header entry %q", part)
		}

		key := strings.TrimSpace(kv[0])
		value := strings.TrimSpace(kv[1])
		if key == "" || value == "" {
			return nil, fmt.Errorf("invalid header entry %q", part)
		}

		headers[key] = value
	}

	return headers, nil
}

func getBaseURL() (string, error) {
	rawBaseURL := os.Getenv("GAGGIUINO_BASE_URL")
	if rawBaseURL == "" {
		gaggiuino.Logger.Info("using default base URL", "baseURL", defaultBaseURL)
		return defaultBaseURL, nil
	}

	parsedURL, err := url.Parse(rawBaseURL)
	if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
		return "", fmt.Errorf("invalid GAGGIUINO_BASE_URL %q", rawBaseURL)
	}

	gaggiuino.Logger.Info("using base URL from env", "scheme", parsedURL.Scheme, "host", parsedURL.Host)
	return parsedURL.String(), nil
}

func printHeader() {
	gaggiuino.Logger.Info("gaggiuino-exporter",
		"version", build.BuildVersion,
		"sha1", build.BuildCommitSha,
		"go", runtime.Version(),
		"goos", runtime.GOOS,
		"goarch", runtime.GOARCH,
	)
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
