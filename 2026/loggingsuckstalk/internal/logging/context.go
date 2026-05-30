package logging

import (
	"context"
	"log/slog"
	"sync"
)

type loggerKey struct{}

type (
	logEventKey struct{}
	logEvent    struct {
		mu     sync.RWMutex
		fields map[string]any
	}
)

func (le *logEvent) addField(field string, value any) {
	le.mu.Lock()
	defer le.mu.Unlock()

	le.fields[field] = value
}

func (le *logEvent) containsError() bool {
	le.mu.RLock()
	defer le.mu.RUnlock()

	return le.fields["error"] != nil
}

func (le *logEvent) mapToArgs() []any {
	le.mu.RLock()
	defer le.mu.RUnlock()

	args := make([]any, 0, len(le.fields))
	for k, v := range le.fields {
		args = append(args, slog.Any(k, v))
	}

	return args
}

// FromContext returns the slog.Logger from the context.
// If no logger is found, it returns the default logger.
func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*slog.Logger); ok {
		return logger
	}

	return slog.Default()
}

// withLogger returns a new context with the given logger attached.
func withLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

func AddField(ctx context.Context, key string, value any) {
	le, ok := ctx.Value(logEventKey{}).(*logEvent)
	if !ok {
		return
	}

	le.addField(key, value)
}
