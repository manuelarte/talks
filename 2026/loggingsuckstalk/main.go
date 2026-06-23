package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/riandyrn/otelchi"
	otelchimetric "github.com/riandyrn/otelchi/metric"
	"go.opentelemetry.io/contrib/bridges/otelslog"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/log/global"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/log"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"

	"github.com/manuelarte/talks/2026/loggingsuckstalk/info"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/config"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/infrastructure/api/rest"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/infrastructure/db"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/logging"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/observability"
)

func main() {
	if err := run(); err != nil {
		//nolint:sloglint // only logging in default this error
		slog.Error("Application error", "error", err)
	}
}

func run() error {
	ctx := context.Background()

	tracer := otel.Tracer(info.AppName)

	otelShutdown, mp, lp, err := setupOTelSDK(ctx, "localhost:4317")
	if err != nil {
		return fmt.Errorf("error setting open telemetry: %w", err)
	}

	defer func() {
		err = errors.Join(err, otelShutdown(context.Background()))
	}()

	// Create a bridged slog logger
	logger := otelslog.NewLogger(info.AppName, otelslog.WithLoggerProvider(lp))

	logger.InfoContext(ctx, "Starting database migration")

	dbConn, err := config.Migrate(ResourcesFolder)
	if err != nil {
		return fmt.Errorf("failed to migrate the database: %w", err)
	}
	defer dbConn.Close()

	// define base config for metric middlewares
	baseCfg := otelchimetric.NewBaseConfig(info.AppName, otelchimetric.WithMeterProvider(mp))
	r := chi.NewRouter()

	//nolint:mnd // guess
	headerTimeout := 10 * time.Second
	r.Use(
		middleware.Logger,
		middleware.Recoverer,
		otelchi.Middleware(info.AppName, otelchi.WithChiRoutes(r)),
		otelchimetric.NewRequestDurationMillis(baseCfg),
		otelchimetric.NewRequestInFlight(baseCfg),
		otelchimetric.NewResponseSizeBytes(baseCfg),
		middleware.RequestID,
		middleware.RealIP,
		logging.AddLogger(slog.Default()), // change for slog.Default() to not to send, or logger to send logs.
		logging.AddLogEvent(logger),       // change for slog.Default() to not to send, or logger to send logs.
		middleware.Timeout(headerTimeout),
	)

	repository := db.NewRepository(dbConn)
	rest.Create(r, SwaggerUI, OpenAPI, repository)

	srvErr := make(chan error, 1)

	srv := &http.Server{
		Addr:              ":8080",
		Handler:           r,
		ReadHeaderTimeout: headerTimeout, // Prevent G112 (CWE-400)
		BaseContext: func(net.Listener) context.Context {
			return observability.AddContext(ctx, tracer)
		},
	}

	srvErr <- srv.ListenAndServe()

	return nil
}

func newPropagator() propagation.TextMapPropagator {
	return propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	)
}

func setupOTelSDK(
	ctx context.Context,
	otelGRPCEndpoint string,
) (func(context.Context) error, *sdkmetric.MeterProvider, *log.LoggerProvider, error) {
	hostname := "manuelarte-ING"
	shutdownFuncs := make([]func(context.Context) error, 3)

	var err error

	// shutdown calls cleanup functions registered via shutdownFuncs.
	// The errors from the calls are joined.
	// Each registered cleanup will be invoked once.
	shutdown := func(ctx context.Context) error {
		var shutdownErr error

		for _, fn := range shutdownFuncs {
			if fn != nil {
				shutdownErr = errors.Join(shutdownErr, fn(ctx))
			}
		}

		shutdownFuncs = nil

		return shutdownErr
	}

	// handleErr calls shutdown for cleanup and makes sure that all errors are returned.
	handleErr := func(inErr error) {
		err = errors.Join(inErr, shutdown(ctx))
	}

	// Set up propagator.
	prop := newPropagator()
	otel.SetTextMapPropagator(prop)

	tp, err := observability.InitTracerProvider(ctx, otelGRPCEndpoint, hostname)
	if err != nil {
		handleErr(err)

		return shutdown, nil, nil, fmt.Errorf("error initializing trace provider: %w", err)
	}

	shutdownFuncs[0] = tp.Shutdown
	otel.SetTracerProvider(tp)

	mp, err := observability.InitMeterProvider(ctx, otelGRPCEndpoint, hostname)
	if err != nil {
		handleErr(err)

		return shutdown, nil, nil, fmt.Errorf("failed to initialize meter provider: %w", err)
	}

	shutdownFuncs[1] = mp.Shutdown
	otel.SetMeterProvider(mp)

	loggerProvider, err := observability.InitLoggingProvider(ctx, otelGRPCEndpoint, hostname)
	if err != nil {
		handleErr(err)

		return shutdown, nil, nil, fmt.Errorf("failed to initialize logger provider: %w", err)
	}

	shutdownFuncs[2] = loggerProvider.Shutdown
	global.SetLoggerProvider(loggerProvider)

	return shutdown, mp, loggerProvider, nil
}
