package logging

import (
	"net/http"
	"time"

	"github.com/google/uuid"
)

// RequestIDMiddleware adds a request ID to each HTTP request and logs request/response
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Generate or extract request ID
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		// Add to context
		ctx := WithRequestID(r.Context(), requestID)
		r = r.WithContext(ctx)

		// Add to response header
		w.Header().Set("X-Request-ID", requestID)

		// Wrap response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Log request start
		start := time.Now()
		InfoContext(ctx, "request started",
			"method", r.Method,
			"path", r.URL.Path,
			"remoteAddr", r.RemoteAddr,
		)

		// Handle request
		next.ServeHTTP(wrapped, r)

		// Log request completion
		duration := time.Since(start)
		if wrapped.statusCode >= 400 {
			ErrorContext(ctx, "request failed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"durationMs", duration.Milliseconds(),
			)
		} else {
			InfoContext(ctx, "request completed",
				"method", r.Method,
				"path", r.URL.Path,
				"status", wrapped.statusCode,
				"durationMs", duration.Milliseconds(),
			)
		}
	})
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Flush implements http.Flusher for SSE support
func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}
