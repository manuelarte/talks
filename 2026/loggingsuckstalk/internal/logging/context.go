package logging

import (
	"context"
	"log/slog"

	"github.com/manuelarte/logevent"
)

var _ logevent.LogEvent[*slog.Logger] = new(TransferLogEvent)

type (
	loggerKey struct{}

	TransferLogEvent struct {
		fields map[string]any
	}
)

func (le *TransferLogEvent) AddField(field string, value any) {
	if le.fields == nil {
		le.fields = make(map[string]any)
	}

	le.fields[field] = value
}

func (le *TransferLogEvent) Log(ctx context.Context, logger *slog.Logger) {
	if le.containsError() {
		logger.ErrorContext(ctx, "Transfer failed", le.mapToArgs()...)

		return
	}

	if le.fields["paymentGatewayError"] != nil ||
		le.fields["kafkaEventError"] != nil ||
		le.fields["accountsUpdatedError"] != nil {
		logger.WarnContext(ctx, "Transfer completed with error", le.mapToArgs()...)
	} else {
		logger.InfoContext(ctx, "Transfer completed", le.mapToArgs()...)
	}
}

func (le *TransferLogEvent) containsError() bool {
	return le.fields["error"] != nil
}

func (le *TransferLogEvent) mapToArgs() []any {
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
	_ = logevent.UpdateLogEvent(ctx, func(le *TransferLogEvent) {
		le.AddField(key, value)
	})
}
