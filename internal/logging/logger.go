package logging

import (
	"log/slog"
	"os"
)

// SetUpLogging sets up the logging for the application based on the log level provided
// logLevel: int - the log level to set the logger to, defaults to error
// returns: *slog.Logger - the logger to be used for logging
// sets the logger as slog.Default
func SetUpLogging(logLevel int) *slog.Logger {

	var lvl slog.Level

	switch {
	case logLevel < int(slog.LevelInfo):
		lvl = slog.LevelDebug
	case logLevel < int(slog.LevelWarn):
		lvl = slog.LevelInfo
	case logLevel < int(slog.LevelError):
		lvl = slog.LevelWarn
	default:
		lvl = slog.LevelError
	}

	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: lvl,
	}))

	slog.SetDefault(logger)
	return logger
}
