# Gaggiuino Prometheus exporter

Provides metrics for [Gaggiuino](https://gaggiuino.github.io/#/)-based espresso machines.
Interacts with the Gaggiuino API to scrape the current system status.

## Usage

```plaintext
Prometheus exporter for Gaggiuino espresso machine.

Usage:
  gaggiuino_exporter [flags]

Flags:
  --basic-auth-users stringToString       Basic Auth users and their passwords as bcrypt hashes (default [])
  -b, --bind-address string               Address to bind to (default "0.0.0.0:9995")
  -h, --help                              help for gaggiuino_exporter
  -m, --mode string                       Expose method - either 'state' or 'http' (default "http")
```

By default the collector sends requests to gaggiuino.local, as this is the default mdns name.
This can be overridden by setting the `GAGGIUINO_BASE_URL` env variable.

## Environment Variables

You can configure runtime behavior via environment variables:

- `MODE` (`http` or `state`)
- `BIND_ADDRESS` (default: `0.0.0.0:9995`)
- `GAGGIUINO_BASE_URL` (default: `http://gaggiuino.local`)
- `GAGGIUINO_HTTP_TIMEOUT` (default: `10s`, supports fractional durations like `0.5s`)

### OpenTelemetry OTLP Metrics Export

- `OTLP_ENABLED` (default: `false`)
- `OTLP_ENDPOINT` (default: `http://localhost:8429/opentelemetry/v1/metrics`)
- `OTLP_INTERVAL` (default: `15s`)
- `OTLP_TIMEOUT` (default: `10s`)
- `OTLP_HEADERS` (optional headers, supports JSON map or key=value CSV)

Accepted `OTLP_HEADERS` formats:

- `{"Authorization":"Bearer token","X-Scope-OrgID":"tenant-a"}`
- `Authorization=Bearer token,X-Scope-OrgID=tenant-a`

Example:

```bash
export OTLP_ENABLED=true
export OTLP_ENDPOINT=http://vmagent:8429/opentelemetry/v1/metrics
export OTLP_INTERVAL=15s
gaggiuino_exporter
```

When enabled, OTLP metrics are exported with `service.name=gaggiuino-exporter`.

## Example output

```plaintext
# HELP gaggiuino_up Whether the Gaggiuino API is reachable
# TYPE gaggiuino_up gauge
gaggiuino_up 1
# HELP gaggiuino_uptime_seconds Uptime of the espresso machine
# TYPE gaggiuino_uptime_seconds gauge
gaggiuino_uptime_seconds 24
# HELP gaggiuino_profile_id Current profile ID
# TYPE gaggiuino_profile_id gauge
gaggiuino_profile_id 10
# HELP gaggiuino_target_temperature Target temperature
# TYPE gaggiuino_target_temperature gauge
gaggiuino_target_temperature 1
# HELP gaggiuino_temperature Current temperature of the boiler
# TYPE gaggiuino_temperature gauge
gaggiuino_temperature 21.572617
# HELP gaggiuino_pressure_bar Current pressure measured at the boiler
# TYPE gaggiuino_pressure_bar gauge
gaggiuino_pressure_bar 0.248914
# HELP gaggiuino_water_level Current water level as measured by the ultrasonic sensor
# TYPE gaggiuino_water_level gauge
gaggiuino_water_level 4
# HELP gaggiuino_shot_weight Current weight of the espresso shot as measured by the scale
# TYPE gaggiuino_shot_weight gauge
gaggiuino_shot_weight 0.1
# HELP gaggiuino_last_shot_id Last shot ID reported by the Gaggiuino API
# TYPE gaggiuino_last_shot_id gauge
gaggiuino_last_shot_id 644
```

## Basic Auth

You can enable HTTP Basic authorization by setting the --basic-auth-users flag (password is "test"):

```bash
gaggiuino_exporter --basic-auth-users='admin=$2b$12$hNf2lSsxfm0.i4a.1kVpSOVyBCfIB51VRjgBUyv6kdnyTlgWj81Ay'
```
