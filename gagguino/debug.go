package gaggiuino

import (
	"log/slog"
	"os"
)

var logLevel = new(slog.LevelVar)

// Logger is the package-wide structured logger. Its level is controlled by SetDebug.
var Logger = slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: logLevel}))

// SetDebug enables or disables debug-level logging across the package.
func SetDebug(enabled bool) {
	if enabled {
		logLevel.Set(slog.LevelDebug)
	} else {
		logLevel.Set(slog.LevelInfo)
	}
}
