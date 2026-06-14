package logging

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

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

// AddLogEvent returns a middleware that adds a request-scoped logger to the context.
// The request-scoped logger includes the request ID if available.
func AddLogEvent(baseLogger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqID := middleware.GetReqID(r.Context())
			le := &logEvent{
				fields: make(map[string]any),
			}
			le.addField("requestId", reqID)
			r = r.WithContext(context.WithValue(r.Context(), logEventKey{}, le))
			next.ServeHTTP(w, r)

			// TODO, THIS IS DIRTY, THINK OF middleware per endpoint
			if strings.Contains(r.URL.Path, "/transfers/") {
				if le.containsError() {
					//nolint:contextcheck // bug in contextcheck
					baseLogger.ErrorContext(r.Context(), "Transfer failed", le.mapToArgs()...)

					return
				}

				if le.fields["paymentGatewayError"] != nil ||
					le.fields["kafkaEventError"] != nil ||
					le.fields["accountsUpdatedError"] != nil {
					baseLogger.WarnContext(r.Context(), "Transfer completed with error", le.mapToArgs()...)
				} else {
					baseLogger.InfoContext(r.Context(), "Transfer completed", le.mapToArgs()...)
				}

				// baseLogger.InfoContext(r.Context(), "Transfer completed", le.mapToArgs()...)
			}
		})
	}
}
