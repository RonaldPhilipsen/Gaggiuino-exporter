# Gaggiuino Prometheus exporter

Provides metrics for [Gaggiuino](https://gaggiuino.github.io/#/)-based espresso machines.
Interacts with the Gaggiuino API to scrape the current system status.

## Usage

```plaintext
Prometheus exporter for Gaggiuino espresso machine.

Usage:
  gaggiuino_exporter [flags]

Flags:
      --basic-auth-users stringToString   Basic Auth users and their passwords as bcypt hashes (default [])
  -b, --bind-address string               Address to bind to (default "0.0.0.0:9995")
  -h, --help                              help for gaggiuino_exporter
  -m, --mode string                       Expose method - either 'state' or 'http' (default "http")
```

## Example output

```plaintext
# HELP Pressure Current pressure measured at the boiler
# TYPE Pressure gauge
Pressure 0.248914
# HELP ProfileId Current profile ID
# TYPE ProfileId gauge
ProfileId 10
# HELP TargetTemperature Target temperature
# TYPE TargetTemperature gauge
TargetTemperature 1
# HELP Temperature Current temperature of the boiler
# TYPE Temperature gauge
Temperature 21.572617
# HELP Uptime Uptime of the espresso machine
# TYPE Uptime gauge
Uptime 24
# HELP WaterLevel Current water level as measured by the ultrasonic sensor
# TYPE WaterLevel gauge
WaterLevel 4
# HELP Weight Current weight of on the scale of the espresso machine
# TYPE Weight gauge
Weight 0.1
# HELP gaggiuino_up Whether the Gaggiuino API is reachable
# TYPE gaggiuino_up gauge
gaggiuino_up 1
```

## Basic Auth

You can enable HTTP Basic authorization by setting the --basic-auth-users flag (password is "test"):

```bash
gaggiuino-exporter --basic-auth-users='admin=$2b$12$hNf2lSsxfm0.i4a.1kVpSOVyBCfIB51VRjgBUyv6kdnyTlgWj81Ay'
```
