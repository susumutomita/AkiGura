package srv

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestServerSetupAndHandlers(t *testing.T) {
	tempDB := filepath.Join(t.TempDir(), "test_server.sqlite3")
	t.Cleanup(func() { os.Remove(tempDB) })

	server, err := New(tempDB, "test-hostname")
	if err != nil {
		t.Fatalf("failed to create server: %v", err)
	}

	// Test root endpoint returns 200 and HTML content
	t.Run("root endpoint returns HTML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()

		server.HandleRoot(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/html; charset=utf-8" {
			t.Errorf("expected Content-Type text/html; charset=utf-8, got %s", contentType)
		}
	})

	// Test user page endpoint
	t.Run("user page returns HTML", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/user", nil)
		w := httptest.NewRecorder()

		server.HandleUserPage(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", w.Code)
		}

		contentType := w.Header().Get("Content-Type")
		if contentType != "text/html; charset=utf-8" {
			t.Errorf("expected Content-Type text/html; charset=utf-8, got %s", contentType)
		}
	})
}

func TestUtilityFunctions(t *testing.T) {
	t.Run("mainDomainFromHost function", func(t *testing.T) {
		tests := []struct {
			input    string
			expected string
		}{
			{"example.exe.cloud:8080", "exe.cloud:8080"},
			{"example.exe.dev", "exe.dev"},
			{"example.exe.cloud", "exe.cloud"},
		}

		for _, test := range tests {
			result := mainDomainFromHost(test.input)
			if result != test.expected {
				t.Errorf("mainDomainFromHost(%q) = %q, expected %q", test.input, result, test.expected)
			}
		}
	})
}
