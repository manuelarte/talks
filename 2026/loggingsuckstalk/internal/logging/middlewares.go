package logging

import (
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5/middleware"
)

// AddLogger returns a middleware that adds a request-scoped logger to the context.
// The request-scoped logger includes the request ID if available.
func AddLogger(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())

			requestLogger := baseLogger
			if reqID != "" {
				requestLogger = baseLogger.With("requestId", reqID)
			}

			r = r.WithContext(withLogger(r.Context(), requestLogger))
			next.ServeHTTP(w, r)
		})
	}
}
