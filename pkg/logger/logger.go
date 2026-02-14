package logger

import (
	"log/slog"
	"os"

	"github.com/abdulshakoor02/goCrmBackend/config"
)

var Log *slog.Logger

func InitLogger(cfg *config.Config) {
	var handler slog.Handler

	opts := &slog.HandlerOptions{
		Level: parseLevel(cfg.LogLevel),
	}

	if cfg.ServerEnv == "production" || cfg.ServerEnv == "staging" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}

	Log = slog.New(handler)
	slog.SetDefault(Log)
}

func parseLevel(level string) slog.Level {
	switch level {
	case "debug":
		return slog.LevelDebug
	case "info":
		return slog.LevelInfo
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
