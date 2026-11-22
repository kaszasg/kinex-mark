package app

import (
	"io"
	"log/slog"

	"github.com/kaszasg/kinex-mark/internal/config"
)

// NewLogger returns a text or JSON slog logger writing to w at cfg.LogLevel.
func NewLogger(cfg config.Config, w io.Writer) *slog.Logger {
	opts := &slog.HandlerOptions{Level: cfg.LogLevel}
	if cfg.LogFormat == "json" {
		return slog.New(slog.NewJSONHandler(w, opts))
	}
	return slog.New(slog.NewTextHandler(w, opts))
}
