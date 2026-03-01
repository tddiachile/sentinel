package logger

import (
	"io"
	"log/slog"
	"os"
	"strings"

	"github.com/enunezf/sentinel/internal/config"
)

// New crea un *slog.Logger configurado segun LoggingConfig.
// format "json" -> slog.NewJSONHandler(output, opts)
// format "text" -> slog.NewTextHandler(output, opts)
// output "stderr" -> os.Stderr; cualquier otro valor -> os.Stdout (default).
// level se mapea a slog.Level mediante ParseLevel.
func New(cfg config.LoggingConfig) *slog.Logger {
	var w io.Writer
	if strings.ToLower(cfg.Output) == "stderr" {
		w = os.Stderr
	} else {
		w = os.Stdout
	}
	return newWithWriter(cfg, w)
}

// NewWithWriter crea un *slog.Logger que escribe en el writer provisto.
// Util para testing: permite capturar el output en un bytes.Buffer sin
// necesidad de redirigir os.Stdout.
func NewWithWriter(cfg config.LoggingConfig, w io.Writer) *slog.Logger {
	return newWithWriter(cfg, w)
}

// newWithWriter es la implementacion interna compartida por New y NewWithWriter.
func newWithWriter(cfg config.LoggingConfig, w io.Writer) *slog.Logger {
	level := ParseLevel(cfg.Level)

	opts := &slog.HandlerOptions{
		Level: level,
	}

	var handler slog.Handler

	if strings.ToLower(cfg.Format) == "text" {
		handler = slog.NewTextHandler(w, opts)
	} else {
		// Default: JSON. Handles "json" and any unrecognised format.
		handler = slog.NewJSONHandler(w, opts)
	}

	return slog.New(handler)
}

// ParseLevel convierte un string "debug"/"info"/"warn"/"error" a slog.Level.
// Valor desconocido -> slog.LevelInfo.
func ParseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// WithComponent retorna un logger hijo con el campo "component" fijo.
func WithComponent(logger *slog.Logger, component string) *slog.Logger {
	return logger.With("component", component)
}
