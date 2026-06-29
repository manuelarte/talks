package logging

import (
	"context"
	"log/slog"

	"github.com/manuelarte/logevent"
	"github.com/manuelarte/logevent/middlewares"
)

type (
	loggerKey struct{}

	GenericLogEvent struct {
		fields map[string]any
	}
)

func (le *GenericLogEvent) AddField(field string, value any) {
	if le.fields == nil {
		le.fields = make(map[string]any)
	}
	le.fields[field] = value
}

func (le *GenericLogEvent) Log(_ context.Context, li logevent.LogInterface) {
	if le.containsError() {
		//nolint:contextcheck // bug in contextcheck
		li.Error("Transfer failed", le.mapToArgs()...)

		return
	}
	if le.fields["paymentGatewayError"] != nil ||
		le.fields["kafkaEventError"] != nil ||
		le.fields["accountsUpdatedError"] != nil {
		li.Warn("Transfer completed with error", le.mapToArgs()...)
	} else {
		li.Info("Transfer completed", le.mapToArgs()...)
	}
}

func (le *GenericLogEvent) containsError() bool {
	return le.fields["error"] != nil
}

func (le *GenericLogEvent) mapToArgs() []any {
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
	_ = middlewares.UpdateLogEvent(ctx, func(le *GenericLogEvent) {
		le.AddField(key, value)
	})
}
