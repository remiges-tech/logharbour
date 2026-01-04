package logharbour

import (
	"net"
	"net/http"
	"strings"

	"github.com/segmentio/ksuid"
)

// HeaderTraceID is the HTTP header name for trace ID.
const HeaderTraceID = "X-Trace-ID"

// WithHTTPRequest returns a new Logger with trace ID and client IP extracted
// from the HTTP request.
//
// Trace ID is extracted from the X-Trace-ID header. If not present, a new KSUID
// is generated. This allows distributed tracing across services when the upstream
// service passes along the trace ID.
//
// Client IP is extracted in this order:
//   - X-Forwarded-For header (first IP) - standard header set by proxies/load balancers
//   - X-Real-IP header - set by Nginx
//   - RemoteAddr - direct TCP connection address
//
// When behind a reverse proxy, configure the proxy to set X-Forwarded-For or X-Real-IP,
// otherwise RemoteAddr will contain the proxy's IP instead of the client's IP.
func (l *Logger) WithHTTPRequest(r *http.Request) *Logger {
	return l.
		WithTraceID(extractOrGenerateTraceID(r)).
		WithRemoteIP(extractClientIP(r))
}

// extractOrGenerateTraceID gets trace ID from X-Trace-ID header.
// If no trace ID is present, generates a new KSUID.
func extractOrGenerateTraceID(r *http.Request) string {
	if traceID := r.Header.Get(HeaderTraceID); traceID != "" {
		return traceID
	}
	return ksuid.New().String()
}

// extractClientIP gets client IP from HTTP request headers or RemoteAddr.
// Priority: X-Forwarded-For (first IP) > X-Real-IP > RemoteAddr.
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (set by proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs: "client, proxy1, proxy2"
		// The first one is the original client
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			ip := strings.TrimSpace(ips[0])
			if ip != "" {
				return ip
			}
		}
	}

	// Check X-Real-IP header (set by nginx)
	if realIP := r.Header.Get("X-Real-IP"); realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr (direct connection)
	// RemoteAddr is in format "IP:port", so we need to extract just the IP
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// If SplitHostPort fails, RemoteAddr might be just an IP (no port)
		return r.RemoteAddr
	}
	return host
}
