package logharbour

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestExtractOrGenerateTraceID(t *testing.T) {
	tests := []struct {
		name          string
		headerValue   string
		headerPresent bool
		wantGenerated bool
	}{
		{
			name:          "extracts trace ID from header",
			headerValue:   "abc123",
			wantGenerated: false,
		},
		{
			name:          "generates trace ID when header is empty",
			headerValue:   "",
			wantGenerated: true,
		},
		{
			name:           "generates trace ID when header value is empty string",
			headerValue:    "",
			headerPresent:  true,
			wantGenerated:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.headerValue != "" || tt.headerPresent {
				req.Header.Set(HeaderTraceID, tt.headerValue)
			}

			got := extractOrGenerateTraceID(req)

			if tt.wantGenerated {
				if got == "" {
					t.Error("expected generated trace ID, got empty string")
				}
				if len(got) != 27 {
					t.Errorf("expected KSUID length 27, got %d", len(got))
				}
			} else {
				if got != tt.headerValue {
					t.Errorf("expected %q, got %q", tt.headerValue, got)
				}
			}
		})
	}
}

func TestExtractClientIP(t *testing.T) {
	tests := []struct {
		name       string
		headers    map[string]string
		remoteAddr string
		want       string
	}{
		{
			name:       "extracts from X-Forwarded-For single IP",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1"},
			remoteAddr: "10.0.0.1:12345",
			want:       "192.168.1.1",
		},
		{
			name:       "extracts first IP from X-Forwarded-For multiple IPs",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.2, 10.0.0.3"},
			remoteAddr: "10.0.0.1:12345",
			want:       "192.168.1.1",
		},
		{
			name:       "extracts from X-Real-IP when X-Forwarded-For is absent",
			headers:    map[string]string{"X-Real-IP": "172.16.0.1"},
			remoteAddr: "10.0.0.1:12345",
			want:       "172.16.0.1",
		},
		{
			name:       "X-Forwarded-For takes priority over X-Real-IP",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1", "X-Real-IP": "172.16.0.1"},
			remoteAddr: "10.0.0.1:12345",
			want:       "192.168.1.1",
		},
		{
			name:       "falls back to RemoteAddr when no headers",
			headers:    map[string]string{},
			remoteAddr: "10.0.0.1:12345",
			want:       "10.0.0.1",
		},
		{
			name:       "handles RemoteAddr without port",
			headers:    map[string]string{},
			remoteAddr: "10.0.0.1",
			want:       "10.0.0.1",
		},
		{
			name:       "handles IPv6 RemoteAddr with port",
			headers:    map[string]string{},
			remoteAddr: "[::1]:12345",
			want:       "::1",
		},
		{
			name:       "handles empty X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": ""},
			remoteAddr: "10.0.0.1:12345",
			want:       "10.0.0.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			got := extractClientIP(req)

			if got != tt.want {
				t.Errorf("expected %q, got %q", tt.want, got)
			}
		})
	}
}

func TestWithHTTPRequest(t *testing.T) {
	ctx := NewLoggerContext(Info)
	logger := NewLogger(ctx, "test-app", nil)

	t.Run("extracts trace ID and IP from request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(HeaderTraceID, "trace-123")
		req.Header.Set("X-Forwarded-For", "192.168.1.100")

		newLogger := logger.WithHTTPRequest(req)

		if newLogger.traceId != "trace-123" {
			t.Errorf("expected traceId %q, got %q", "trace-123", newLogger.traceId)
		}
		if newLogger.remoteIP != "192.168.1.100" {
			t.Errorf("expected remoteIP %q, got %q", "192.168.1.100", newLogger.remoteIP)
		}
	})

	t.Run("generates trace ID when not in header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.RemoteAddr = "10.0.0.1:8080"

		newLogger := logger.WithHTTPRequest(req)

		if newLogger.traceId == "" {
			t.Error("expected generated traceId, got empty string")
		}
		if len(newLogger.traceId) != 27 {
			t.Errorf("expected KSUID length 27, got %d", len(newLogger.traceId))
		}
		if newLogger.remoteIP != "10.0.0.1" {
			t.Errorf("expected remoteIP %q, got %q", "10.0.0.1", newLogger.remoteIP)
		}
	})

	t.Run("returns new logger without modifying original", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set(HeaderTraceID, "trace-456")
		req.Header.Set("X-Real-IP", "172.16.0.50")

		newLogger := logger.WithHTTPRequest(req)

		if logger.traceId != "" {
			t.Errorf("original logger traceId should be empty, got %q", logger.traceId)
		}
		if logger.remoteIP != "" {
			t.Errorf("original logger remoteIP should be empty, got %q", logger.remoteIP)
		}
		if newLogger.traceId != "trace-456" {
			t.Errorf("new logger traceId should be %q, got %q", "trace-456", newLogger.traceId)
		}
		if newLogger.remoteIP != "172.16.0.50" {
			t.Errorf("new logger remoteIP should be %q, got %q", "172.16.0.50", newLogger.remoteIP)
		}
	})
}

func TestWithTraceID(t *testing.T) {
	ctx := NewLoggerContext(Info)
	logger := NewLogger(ctx, "test-app", nil)

	t.Run("sets trace ID", func(t *testing.T) {
		newLogger := logger.WithTraceID("my-trace-id")

		if newLogger.traceId != "my-trace-id" {
			t.Errorf("expected traceId %q, got %q", "my-trace-id", newLogger.traceId)
		}
	})

	t.Run("returns new logger without modifying original", func(t *testing.T) {
		newLogger := logger.WithTraceID("another-trace")

		if logger.traceId != "" {
			t.Errorf("original logger traceId should be empty, got %q", logger.traceId)
		}
		if newLogger.traceId != "another-trace" {
			t.Errorf("new logger traceId should be %q, got %q", "another-trace", newLogger.traceId)
		}
	})

	t.Run("chains with other With methods", func(t *testing.T) {
		newLogger := logger.
			WithTraceID("chained-trace").
			WithModule("test-module").
			WithRemoteIP("1.2.3.4")

		if newLogger.traceId != "chained-trace" {
			t.Errorf("expected traceId %q, got %q", "chained-trace", newLogger.traceId)
		}
		if newLogger.module != "test-module" {
			t.Errorf("expected module %q, got %q", "test-module", newLogger.module)
		}
		if newLogger.remoteIP != "1.2.3.4" {
			t.Errorf("expected remoteIP %q, got %q", "1.2.3.4", newLogger.remoteIP)
		}
	})
}

func TestTraceIDInLogEntry(t *testing.T) {
	ctx := NewLoggerContext(Info)
	var buf strings.Builder
	logger := NewLogger(ctx, "test-app", &buf)

	logger.WithTraceID("entry-trace-123").Log("test message")

	output := buf.String()
	if !strings.Contains(output, `"trace_id":"entry-trace-123"`) {
		t.Errorf("expected trace_id in output, got: %s", output)
	}
}

// Example usage in an HTTP handler.
// For Gin, use c.Request to get the *http.Request.
func ExampleLogger_WithHTTPRequest() {
	ctx := NewLoggerContext(Info)
	var buf strings.Builder
	logger := NewLogger(ctx, "my-app", &buf)

	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("X-Trace-ID", "abc-123")
	req.Header.Set("X-Forwarded-For", "192.168.1.1")

	logger.WithHTTPRequest(req).Log("processing request")

	output := buf.String()
	fmt.Println("has trace_id:", strings.Contains(output, `"trace_id":"abc-123"`))
	fmt.Println("has remote_ip:", strings.Contains(output, `"remote_ip":"192.168.1.1"`))
	// Output:
	// has trace_id: true
	// has remote_ip: true
}
