package rest

import (
	"github.com/go-chi/chi/v5"
	"github.com/manuelarte/embeddedswagger"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/domain"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/infrastructure/pub"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/paymentgateway"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/services"
)

type Rest struct {
	accountsHandler
}

func Create(r chi.Router, openAPI []byte, accountsRepository domain.AccountRepository) {
	r.Handle("/metrics", promhttp.Handler())
	_ = embeddedswagger.Add(embeddedswagger.DefaultConfig(openAPI), r)

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
