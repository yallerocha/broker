package utils

import (
	"fmt"
	"log/slog"
	"os"
	"runtime"
)

var logger *slog.Logger

// Configure the logger
func Configure_logger() {
	log_handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})
	logger = slog.New(log_handler)

	logger.With("app", "BROKER")
}

// Use the logger to show a message if an error exists.
// Finalizes the broker if it receives an error.
func Log_fatal(msg string, err error) {
	Log_err(msg, err)

	if err != nil {
		os.Exit(1)
	}
}

// Use the logger to show a message if an error exists.
func Log_err(msg string, err error) {
	if err != nil {
		_, file, line, ok := runtime.Caller(1)
		if !ok {
			file = "Unknown"
			line = 0
		}

		logger.Error("❌ "+msg+": "+err.Error(), slog.String("source", fmt.Sprintf("%s:%d", file, line)))
	}
}

// Use the logger to show a message.
func Log_info(msg string) {
	logger.Info(msg)
}
