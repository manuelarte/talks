package rest

import (
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/domain"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/infrastructure/pub"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/paymentgateway"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/services"
)

type Rest struct {
	accountsHandler
}

func Create(r chi.Router, swaggerUI fs.FS, openAPI []byte, accountsRepository domain.AccountRepository) {
	// Prometheus
	r.Handle("/metrics", promhttp.Handler())

	// Swagger
	sfs, _ := fs.Sub(swaggerUI, "static/swagger-ui")
	r.Handle("/swagger/*", http.StripPrefix("/swagger/", http.FileServer(http.FS(sfs))))

	r.Get("/docs", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(openAPI)
	})

	paymentClient := paymentgateway.NewClient(accountsRepository)
	moneyTransferService := services.NewMoneyTransferService(accountsRepository, paymentClient, pub.Pub{})
	restAPI := Rest{accountsHandler: accountsHandler{moneyTransferService, accountsRepository}}
	ssi := NewStrictHandlerWithOptions(restAPI, nil, StrictHTTPServerOptions{})
	HandlerWithOptions(ssi, ChiServerOptions{
		BaseRouter: r,
	})
}

func getOrDefault[T any](x *T, y T) T {
	if x != nil {
		return *x
	}

	return y
}
