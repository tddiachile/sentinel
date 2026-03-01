package middleware_test

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"

	"github.com/enunezf/sentinel/internal/middleware"
)

// newTestApp construye una app Fiber minima con el middleware RequestID montado
// y un handler que expone el valor de Locals("request_id") como header de respuesta
// para facilitar la verificacion en los tests.
func newTestApp() *fiber.App {
	app := fiber.New(fiber.Config{
		// Deshabilitar el prefork y otros features que interfieren en tests.
		DisableStartupMessage: true,
	})

	app.Use(middleware.RequestID())

	app.Get("/", func(c *fiber.Ctx) error {
		// Expone el valor del local para que los tests puedan inspeccionarlo.
		rid, _ := c.Locals(middleware.LocalRequestID).(string)
		c.Set("X-Local-Request-ID", rid)
		return c.SendStatus(fiber.StatusOK)
	})

	return app
}

// TestRequestID_Generated verifica que cuando el request NO trae el header
// X-Request-ID, el middleware genera un UUID valido (no vacio).
func TestRequestID_Generated(t *testing.T) {
	app := newTestApp()

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	rid := resp.Header.Get("X-Request-ID")
	if rid == "" {
		t.Fatal("expected X-Request-ID header to be set in response, got empty string")
	}

	// UUID v4 tiene 36 caracteres (8-4-4-4-12 con guiones).
	if len(rid) != 36 {
		t.Errorf("expected UUID v4 with 36 chars, got %q (len=%d)", rid, len(rid))
	}
}

// TestRequestID_Propagated verifica que cuando el request trae X-Request-ID,
// ese mismo valor se reutiliza (no se genera uno nuevo).
func TestRequestID_Propagated(t *testing.T) {
	app := newTestApp()

	incomingID := "my-custom-request-id-12345"
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", incomingID)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	rid := resp.Header.Get("X-Request-ID")
	if rid != incomingID {
		t.Errorf("expected X-Request-ID=%q (propagated), got %q", incomingID, rid)
	}
}

// TestRequestID_InResponse verifica que el header X-Request-ID siempre
// esta presente en la respuesta, tanto si se genero como si se propagó.
func TestRequestID_InResponse(t *testing.T) {
	app := newTestApp()

	tests := []struct {
		name       string
		headerVal  string
		wantHeader bool
	}{
		{"sin header entrante", "", true},
		{"con header entrante", "abc-123", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tc.headerVal != "" {
				req.Header.Set("X-Request-ID", tc.headerVal)
			}

			resp, err := app.Test(req)
			if err != nil {
				t.Fatalf("app.Test error: %v", err)
			}
			defer resp.Body.Close()

			rid := resp.Header.Get("X-Request-ID")
			if tc.wantHeader && rid == "" {
				t.Error("expected X-Request-ID header in response, got empty")
			}
		})
	}
}

// TestRequestID_LocalsSet verifica que el valor del request ID queda guardado
// en c.Locals(LocalRequestID) y es accesible por handlers posteriores.
func TestRequestID_LocalsSet(t *testing.T) {
	app := newTestApp()

	incomingID := "locals-check-id"
	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", incomingID)

	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	// El handler expone el valor del local en X-Local-Request-ID.
	localVal := resp.Header.Get("X-Local-Request-ID")
	if localVal != incomingID {
		t.Errorf("expected Locals(%q)=%q, got %q", middleware.LocalRequestID, incomingID, localVal)
	}
}

// TestRequestID_GeneratedIsValidUUIDFormat verifica que el UUID generado
// tiene el formato correcto: 8-4-4-4-12 separado por guiones.
func TestRequestID_GeneratedIsValidUUIDFormat(t *testing.T) {
	app := newTestApp()

	req := httptest.NewRequest("GET", "/", nil)
	resp, err := app.Test(req)
	if err != nil {
		t.Fatalf("app.Test error: %v", err)
	}
	defer resp.Body.Close()

	rid := resp.Header.Get("X-Request-ID")
	parts := splitByDash(rid)
	if len(parts) != 5 {
		t.Errorf("UUID should have 5 parts separated by '-', got %d parts in %q", len(parts), rid)
		return
	}

	expectedLengths := []int{8, 4, 4, 4, 12}
	for i, part := range parts {
		if len(part) != expectedLengths[i] {
			t.Errorf("UUID part %d should have length %d, got %d (%q)", i, expectedLengths[i], len(part), part)
		}
	}
}

// splitByDash divide un string por el caracter '-'.
func splitByDash(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '-' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}
