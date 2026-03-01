package logger_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"strings"
	"testing"

	"github.com/enunezf/sentinel/internal/config"
	"github.com/enunezf/sentinel/internal/logger"
)

// TestParseLevel verifica que todos los niveles validos se parsean correctamente
// y que un valor invalido retorna slog.LevelInfo.
func TestParseLevel(t *testing.T) {
	tests := []struct {
		input    string
		expected slog.Level
	}{
		{"debug", slog.LevelDebug},
		{"DEBUG", slog.LevelDebug},
		{"info", slog.LevelInfo},
		{"INFO", slog.LevelInfo},
		{"warn", slog.LevelWarn},
		{"WARN", slog.LevelWarn},
		{"warning", slog.LevelWarn},
		{"WARNING", slog.LevelWarn},
		{"error", slog.LevelError},
		{"ERROR", slog.LevelError},
		// Valores invalidos deben retornar INFO.
		{"", slog.LevelInfo},
		{"trace", slog.LevelInfo},
		{"fatal", slog.LevelInfo},
		{"unknown", slog.LevelInfo},
		{"  info  ", slog.LevelInfo}, // con espacios -> trim -> "info" -> INFO
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := logger.ParseLevel(tc.input)
			if got != tc.expected {
				t.Errorf("ParseLevel(%q) = %v; want %v", tc.input, got, tc.expected)
			}
		})
	}
}

// TestNew_JSON verifica que NewWithWriter con format "json" emite JSON valido.
func TestNew_JSON(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	log := logger.NewWithWriter(cfg, &buf)
	log.Info("test message", "key", "value")

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got empty string")
	}

	// Debe ser JSON valido.
	var record map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &record); err != nil {
		t.Fatalf("output is not valid JSON: %v\noutput: %s", err, output)
	}

	// Verificar campos obligatorios.
	if record["msg"] != "test message" {
		t.Errorf("expected msg='test message', got %v", record["msg"])
	}
	if record["key"] != "value" {
		t.Errorf("expected key='value', got %v", record["key"])
	}
	if _, ok := record["time"]; !ok {
		t.Error("expected 'time' field in JSON output")
	}
	if _, ok := record["level"]; !ok {
		t.Error("expected 'level' field in JSON output")
	}
}

// TestNew_Text verifica que NewWithWriter con format "text" emite texto (no JSON).
func TestNew_Text(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "text",
		Output: "stdout",
	}

	log := logger.NewWithWriter(cfg, &buf)
	log.Info("test text message")

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got empty string")
	}

	// El output de texto NO es JSON valido.
	var record map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &record); err == nil {
		t.Error("expected text format output to NOT be valid JSON, but it was")
	}

	// El output de texto debe contener el mensaje.
	if !strings.Contains(output, "test text message") {
		t.Errorf("expected output to contain 'test text message', got: %s", output)
	}
}

// TestLevelFiltering verifica que un logger configurado con nivel "warn"
// descarta mensajes de nivel "info" y los de nivel "warn" los emite.
func TestLevelFiltering(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.LoggingConfig{
		Level:  "warn",
		Format: "json",
		Output: "stdout",
	}

	log := logger.NewWithWriter(cfg, &buf)

	// Este mensaje NO debe aparecer (INFO < WARN).
	log.Info("should be filtered")

	if buf.Len() > 0 {
		t.Errorf("INFO message should be filtered when level=warn, got: %s", buf.String())
	}

	// Este mensaje SI debe aparecer (WARN >= WARN).
	log.Warn("should appear")

	if buf.Len() == 0 {
		t.Error("WARN message should appear when level=warn, got empty output")
	}

	if !strings.Contains(buf.String(), "should appear") {
		t.Errorf("expected 'should appear' in output, got: %s", buf.String())
	}
}

// TestLevelFiltering_Debug verifica que DEBUG se filtra cuando el nivel es INFO.
func TestLevelFiltering_Debug(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	log := logger.NewWithWriter(cfg, &buf)
	log.Debug("debug message should be filtered")

	if buf.Len() > 0 {
		t.Errorf("DEBUG message should be filtered when level=info, got: %s", buf.String())
	}
}

// TestWithComponent verifica que WithComponent agrega el campo "component"
// en todos los logs del logger hijo.
func TestWithComponent(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	base := logger.NewWithWriter(cfg, &buf)
	child := logger.WithComponent(base, "auth-service")

	child.Info("login attempt")

	output := buf.String()
	if output == "" {
		t.Fatal("expected log output, got empty string")
	}

	var record map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(output)), &record); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if record["component"] != "auth-service" {
		t.Errorf("expected component='auth-service', got %v", record["component"])
	}
}

// TestWithComponent_MultipleMessages verifica que el campo "component" persiste
// en todos los mensajes del logger hijo, no solo en el primero.
func TestWithComponent_MultipleMessages(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}

	base := logger.NewWithWriter(cfg, &buf)
	child := logger.WithComponent(base, "bootstrap")

	child.Info("message one")
	child.Info("message two")
	child.Warn("message three")

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 log lines, got %d: %s", len(lines), buf.String())
	}

	for i, line := range lines {
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("line %d is not valid JSON: %v", i+1, err)
		}
		if record["component"] != "bootstrap" {
			t.Errorf("line %d: expected component='bootstrap', got %v", i+1, record["component"])
		}
	}
}

// TestNew_ReturnsLogger verifica que New() retorna un *slog.Logger funcional
// (no nil) tanto para output "stdout" como "stderr".
func TestNew_ReturnsLogger(t *testing.T) {
	cfgs := []config.LoggingConfig{
		{Level: "info", Format: "json", Output: "stdout"},
		{Level: "info", Format: "json", Output: "stderr"},
		{Level: "info", Format: "json", Output: ""}, // default a stdout
		{Level: "info", Format: "text", Output: "stdout"},
	}

	for _, cfg := range cfgs {
		log := logger.New(cfg)
		if log == nil {
			t.Errorf("New(%+v) returned nil, expected *slog.Logger", cfg)
		}
	}
}

// TestNew_UnknownFormat verifica que un formato desconocido usa JSON como default.
func TestNew_UnknownFormat(t *testing.T) {
	var buf bytes.Buffer
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "xml", // formato desconocido
		Output: "stdout",
	}

	log := logger.NewWithWriter(cfg, &buf)
	log.Info("default format test")

	output := strings.TrimSpace(buf.String())
	var record map[string]any
	if err := json.Unmarshal([]byte(output), &record); err != nil {
		t.Errorf("unknown format should fall back to JSON, got non-JSON output: %s", output)
	}
}
