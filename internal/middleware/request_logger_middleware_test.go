package middleware_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/enunezf/sentinel/internal/config"
	"github.com/enunezf/sentinel/internal/logger"
	"github.com/enunezf/sentinel/internal/middleware"
)

// newLoggerApp construye una app Fiber con RequestID + RequestLogger, escribe
// los logs en el buffer provisto, y registra el handler que retorna el statusCode
// indicado. Nivel de log: DEBUG para capturar todos los mensajes en los tests.
func newLoggerApp(buf *bytes.Buffer, statusCode int) *fiber.App {
	cfg := config.LoggingConfig{
		Level:  "debug",
		Format: "json",
		Output: "stdout",
	}
	log := logger.NewWithWriter(cfg, buf)

	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	app.Use(middleware.RequestID())
	app.Use(middleware.RequestLogger(log))

	app.Get("/*", func(c *fiber.Ctx) error {
		return c.SendStatus(statusCode)
	})

	return app
}

// parseLastJSONLine analiza la ultima linea JSON del buffer de log.
func parseLastJSONLine(t *testing.T, buf *bytes.Buffer) map[string]any {
	t.Helper()
	output := strings.TrimSpace(buf.String())
	if output == "" {
		t.Fatal("log buffer is empty — no log entry was emitted")
	}
	lines := strings.Split(output, "\n")
	lastLine := lines[len(lines)-1]

	var record map[string]any
	if err := json.Unmarshal([]byte(lastLine), &record); err != nil {
		t.Fatalf("log output is not valid JSON: %v\nline: %s", err, lastLine)
	}
	return record
}

// TestRequestLogger_Fields verifica que el log emite los campos estructurados
// obligatorios: method, path, status, latency_ms, request_id.
func TestRequestLogger_Fields(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusOK)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Request-ID", "test-rid-001")
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	record := parseLastJSONLine(t, &buf)

	requiredFields := []string{"method", "path", "status", "latency_ms", "request_id"}
	for _, field := range requiredFields {
		if _, ok := record[field]; !ok {
			t.Errorf("expected field %q in log record, but it was absent\nrecord: %v", field, record)
		}
	}

	if record["method"] != "GET" {
		t.Errorf("expected method='GET', got %v", record["method"])
	}
	if record["path"] != "/api/test" {
		t.Errorf("expected path='/api/test', got %v", record["path"])
	}
	if record["request_id"] != "test-rid-001" {
		t.Errorf("expected request_id='test-rid-001', got %v", record["request_id"])
	}
	// status es float64 en JSON
	if record["status"] != float64(200) {
		t.Errorf("expected status=200, got %v", record["status"])
	}
}

// TestRequestLogger_ErrorLevel verifica que un status 500 genera un log con level ERROR.
func TestRequestLogger_ErrorLevel(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusInternalServerError)

	req := httptest.NewRequest("GET", "/fail", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	record := parseLastJSONLine(t, &buf)

	level, ok := record["level"].(string)
	if !ok {
		t.Fatalf("expected 'level' field to be a string, got %T: %v", record["level"], record["level"])
	}
	if strings.ToUpper(level) != "ERROR" {
		t.Errorf("expected level=ERROR for status 500, got %q", level)
	}
}

// TestRequestLogger_WarnLevel verifica que un status 400 genera un log con level WARN.
func TestRequestLogger_WarnLevel(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusBadRequest)

	req := httptest.NewRequest("GET", "/bad", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	record := parseLastJSONLine(t, &buf)

	level, ok := record["level"].(string)
	if !ok {
		t.Fatalf("expected 'level' field to be a string, got %T: %v", record["level"], record["level"])
	}
	if strings.ToUpper(level) != "WARN" {
		t.Errorf("expected level=WARN for status 400, got %q", level)
	}
}

// TestRequestLogger_InfoLevel verifica que un status 200 genera un log con level INFO.
func TestRequestLogger_InfoLevel(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusOK)

	req := httptest.NewRequest("GET", "/users", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	record := parseLastJSONLine(t, &buf)

	level, ok := record["level"].(string)
	if !ok {
		t.Fatalf("expected 'level' field to be a string, got %T: %v", record["level"], record["level"])
	}
	if strings.ToUpper(level) != "INFO" {
		t.Errorf("expected level=INFO for status 200, got %q", level)
	}
}

// TestRequestLogger_HealthDebug verifica que la ruta /health genera un log
// con level DEBUG (ruido bajo, no INFO).
func TestRequestLogger_HealthDebug(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusOK)

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	// Con nivel DEBUG el log debe aparecer.
	record := parseLastJSONLine(t, &buf)

	level, ok := record["level"].(string)
	if !ok {
		t.Fatalf("expected 'level' field to be a string, got %T: %v", record["level"], record["level"])
	}
	if strings.ToUpper(level) != "DEBUG" {
		t.Errorf("expected level=DEBUG for /health, got %q", level)
	}
}

// TestRequestLogger_SwaggerDebug verifica que rutas /swagger/* generan DEBUG.
func TestRequestLogger_SwaggerDebug(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusOK)

	paths := []string{"/swagger/index.html", "/swagger/doc.json", "/swagger/anything"}

	for _, path := range paths {
		buf.Reset()

		req := httptest.NewRequest("GET", path, nil)
		resp, err := app.Test(req)
		if err != nil {
			t.Fatalf("app.Test error for path %s: %v", path, err)
		}
		resp.Body.Close()

		record := parseLastJSONLine(t, &buf)

		level, ok := record["level"].(string)
		if !ok {
			t.Fatalf("path %s: expected 'level' field to be string, got %T", path, record["level"])
		}
		if strings.ToUpper(level) != "DEBUG" {
			t.Errorf("path %s: expected level=DEBUG for swagger route, got %q", path, level)
		}
	}
}

// TestRequestLogger_HealthFiltered verifica que /health NO aparece en el log
// cuando el nivel configurado es INFO (DEBUG se filtra).
func TestRequestLogger_HealthFiltered(t *testing.T) {
	var buf bytes.Buffer
	// Logger con nivel INFO — los mensajes DEBUG no deben aparecer.
	cfg := config.LoggingConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}
	log := logger.NewWithWriter(cfg, &buf)

	app := fiber.New(fiber.Config{DisableStartupMessage: true})
	app.Use(middleware.RequestID())
	app.Use(middleware.RequestLogger(log))
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.SendStatus(fiber.StatusOK)
	})

	req := httptest.NewRequest("GET", "/health", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	// Con nivel INFO, el log de /health (DEBUG) no debe emitirse.
	if buf.Len() > 0 {
		t.Errorf("expected no log output for /health with level=info, got: %s", buf.String())
	}
}

// TestRequestLogger_LatencyField verifica que el campo latency_ms es un numero
// positivo (float64).
func TestRequestLogger_LatencyField(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusOK)

	req := httptest.NewRequest("GET", "/ping", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	record := parseLastJSONLine(t, &buf)

	latency, ok := record["latency_ms"].(float64)
	if !ok {
		t.Fatalf("expected latency_ms to be float64, got %T: %v", record["latency_ms"], record["latency_ms"])
	}
	if latency < 0 {
		t.Errorf("expected latency_ms >= 0, got %f", latency)
	}
}

// TestRequestLogger_ComponentField verifica que el campo "component" es "http".
func TestRequestLogger_ComponentField(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusOK)

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	record := parseLastJSONLine(t, &buf)

	if record["component"] != "http" {
		t.Errorf("expected component='http', got %v", record["component"])
	}
}

// TestRequestLogger_NoSensitiveData verifica que el log NO contiene headers
// sensibles como Authorization o X-App-Key.
func TestRequestLogger_NoSensitiveData(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusOK)

	req := httptest.NewRequest("GET", "/secure", nil)
	req.Header.Set("Authorization", "Bearer super-secret-token-value")
	req.Header.Set("X-App-Key", "my-very-secret-app-key")

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	output := buf.String()

	sensitiveValues := []string{
		"super-secret-token-value",
		"my-very-secret-app-key",
	}

	for _, sensitive := range sensitiveValues {
		if strings.Contains(output, sensitive) {
			t.Errorf("log output should NOT contain sensitive value %q, but it does\noutput: %s", sensitive, output)
		}
	}
}

// TestResolveLevel_StatusCodes verifica los distintos rangos de status code
// a traves de requests reales.
func TestResolveLevel_StatusCodes(t *testing.T) {
	tests := []struct {
		name          string
		path          string
		status        int
		expectedLevel string
	}{
		{"200 OK", "/ok", 200, "INFO"},
		{"201 Created", "/created", 201, "INFO"},
		{"204 No Content", "/no-content", 204, "INFO"},
		{"301 Redirect", "/redirect", 301, "INFO"},
		{"400 Bad Request", "/bad", 400, "WARN"},
		{"401 Unauthorized", "/unauth", 401, "WARN"},
		{"403 Forbidden", "/forbidden", 403, "WARN"},
		{"404 Not Found", "/missing", 404, "WARN"},
		{"500 Internal Error", "/error", 500, "ERROR"},
		{"503 Unavailable", "/unavail", 503, "ERROR"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			var buf bytes.Buffer

			cfg := config.LoggingConfig{
				Level:  "debug",
				Format: "json",
				Output: "stdout",
			}
			log := logger.NewWithWriter(cfg, &buf)

			app := fiber.New(fiber.Config{DisableStartupMessage: true})
			app.Use(middleware.RequestID())
			app.Use(middleware.RequestLogger(log))
			app.Get(tc.path, func(c *fiber.Ctx) error {
				return c.SendStatus(tc.status)
			})

			req := httptest.NewRequest("GET", tc.path, nil)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test error: %v", err)
			}
			resp.Body.Close()

			record := parseLastJSONLine(t, &buf)

			level, ok := record["level"].(string)
			if !ok {
				t.Fatalf("expected 'level' to be string, got %T", record["level"])
			}
			if strings.ToUpper(level) != tc.expectedLevel {
				t.Errorf("status %d: expected level=%s, got %q", tc.status, tc.expectedLevel, level)
			}
		})
	}
}

// TestRequestLogger_ClientIP_ForwardedFor verifica que el campo "ip" en el log
// usa el primer valor de X-Forwarded-For cuando esta presente.
func TestRequestLogger_ClientIP_ForwardedFor(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusOK)

	tests := []struct {
		name      string
		xffHeader string
		wantIP    string
	}{
		{
			name:      "ip simple",
			xffHeader: "10.0.0.1",
			wantIP:    "10.0.0.1",
		},
		{
			name:      "lista con comas — toma el primero",
			xffHeader: "10.0.0.1, 10.0.0.2, 10.0.0.3",
			wantIP:    "10.0.0.1",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			buf.Reset()
			req := httptest.NewRequest("GET", "/ip-test", nil)
			req.Header.Set("X-Forwarded-For", tc.xffHeader)
			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test error: %v", err)
			}
			resp.Body.Close()

			record := parseLastJSONLine(t, &buf)
			if record["ip"] != tc.wantIP {
				t.Errorf("expected ip=%q, got %v", tc.wantIP, record["ip"])
			}
		})
	}
}

// TestRequestLogger_Message verifica que el campo msg es "HTTP request".
func TestRequestLogger_Message(t *testing.T) {
	var buf bytes.Buffer
	app := newLoggerApp(&buf, fiber.StatusOK)

	req := httptest.NewRequest("GET", "/anything", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	record := parseLastJSONLine(t, &buf)

	if record["msg"] != "HTTP request" {
		t.Errorf("expected msg='HTTP request', got %v", record["msg"])
	}
}

// Verificacion estatica: slog.Level es comparable con constantes slog.
var _ slog.Level = slog.LevelDebug
