package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/domain"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/httperrors"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/infrastructure/pub"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/logging"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/observability"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/paymentgateway"
)

type MoneyTransferService struct {
	cache         map[domain.IdempotenceKey]struct{}
	paymentClient *paymentgateway.Client
	repo          domain.AccountRepository
	pub           pub.Pub
}

func NewMoneyTransferService(
	repo domain.AccountRepository,
	paymentClient *paymentgateway.Client,
	pub pub.Pub,
) MoneyTransferService {
	return MoneyTransferService{
		cache:         make(map[domain.IdempotenceKey]struct{}),
		repo:          repo,
		paymentClient: paymentClient,
		pub:           pub,
	}
}

// Transfer money from one account to another.
// It does the following:
//   - Send the transfer request to a payment gateway.
//   - After the payment goes through, it sends an event.
//   - It updates the accounts.
//
// Note: In a real production environment this would not be the way to implement it,
// since you need to take into account that each steps can fail. Then patterns like the Outpux Pattern
// could be implemented to make sure that the event is at least once delivered.
// and the same with the account update.
func (s *MoneyTransferService) Transfer(
	ctx context.Context,
	idempotenceKey domain.IdempotenceKey,
	giverID domain.GiverID,
	receiverID domain.ReceiverID,
	amount domain.Money,
) error {
	ctx, span := observability.StartSpan(
		ctx, "MoneyTransferService.Transfer",
		trace.WithAttributes(
			attribute.String("idempotenceKey", idempotenceKey.String()),
		))
	defer span.End()

	logger := logging.FromContext(ctx)

	if _, ok := s.cache[idempotenceKey]; ok {
		logger.InfoContext(ctx, fmt.Sprintf("Money Transfer (%q) already processed", idempotenceKey))

		return nil
	}

	logger.InfoContext(ctx,
		fmt.Sprintf("New money transfer from %q to %q with idempotenceKey %s request received",
			giverID, receiverID, idempotenceKey),
	)

	moneyTransfer := domain.NewMoneyTransfer(idempotenceKey, giverID, receiverID, amount)

	logger.InfoContext(ctx, fmt.Sprintf("[PaymentGateway]: Processing money transfer, key=%q", idempotenceKey))

	ctx, cancelFn := context.WithTimeout(ctx, 3*time.Second)
	defer cancelFn()

	type transferResult struct {
		response paymentgateway.TransferResponse
		err      error
	}

	resultChan := make(chan transferResult, 1)

	pgStartTime := time.Now()

	go func() {
		response, err := s.paymentClient.Transfer(ctx, moneyTransferToTransferRequest(moneyTransfer))
		resultChan <- transferResult{response, err}
	}()

	errTimeout := errors.New("payment gateway timeout")

	var (
		response paymentgateway.TransferResponse
		err      error
	)

	select {
	case result := <-resultChan:
		response = result.response
		err = result.err
	case <-ctx.Done():
		err = errTimeout
	}

	pgElapsedMs := time.Since(pgStartTime) / 1_000_000
	logging.AddField(ctx, "paymentGatewayElapsed_ms", pgElapsedMs)

	if err != nil {
		logging.AddField(ctx, "paymentGateway", "error")

		switch {
		case errors.Is(err, paymentgateway.ErrNotEnoughSaldo):
			logger.WarnContext(
				ctx,
				fmt.Sprintf("[PaymentGateway]: Validation error, key=%q, err=%q", idempotenceKey, err),
			)
			logging.AddField(ctx, "paymentGatewayError", paymentgateway.ErrNotEnoughSaldo)

			return httperrors.ValidationError{
				Title:   "Saldo error",
				Message: "Not enough saldo",
			}
		default:
			logger.ErrorContext(
				ctx,
				fmt.Sprintf("[PaymentGateway]: error %q, key=%q", err, idempotenceKey),
			)
			logging.AddField(ctx, "paymentGatewayError", err.Error())

			return httperrors.InternalServerError{
				Title:   "Payment gateway error",
				Message: err.Error(),
			}
		}
	}

	logging.AddField(ctx, "paymentGateway", "success")
	logger.InfoContext(
		ctx,
		fmt.Sprintf("[PaymentGateway]: Money transfer processed, key=%q", idempotenceKey),
	)

	err = s.pub.PublishMoneyTransfer(moneyTransfer)
	if err != nil {
		logger.ErrorContext(
			ctx,
			fmt.Sprintf("Failed to publish money transfer event, key=%q", idempotenceKey),
			slog.Any("err", err),
		)
		logging.AddField(ctx, "kafkaEvent", "error")
		logging.AddField(ctx, "kafkaEventError", err.Error())
	} else {
		logger.InfoContext(ctx, fmt.Sprintf("Money transferred event sent, key=%q", idempotenceKey))
		logging.AddField(ctx, "kafkaEvent", "success")
	}

	errUpdateAmounts := s.repo.SaveNewAmounts(
		ctx,
		giverID,
		domain.MustMoney(response.GiverAmount.String()),
		receiverID,
		domain.MustMoney(response.ReceiverAmount.String()),
	)
	if errUpdateAmounts != nil {
		logger.WarnContext(ctx, "Failed to update accounts amounts", slog.Any("err", errUpdateAmounts))
		logging.AddField(ctx, "accountsUpdated", "error")
		logging.AddField(ctx, "accountsUpdatedError", errUpdateAmounts.Error())
	} else {
		logger.InfoContext(ctx, fmt.Sprintf("Accounts updated for key=%q", idempotenceKey))
		logging.AddField(ctx, "accountsUpdated", "success")
	}

	s.cache[idempotenceKey] = struct{}{}

	return nil
}

func moneyTransferToTransferRequest(moneyTransfer domain.MoneyTransfer) paymentgateway.TransferRequest {
	decimalAmount, _ := decimal.NewFromString(moneyTransfer.Amount().String())

	return paymentgateway.TransferRequest{
		IdempotenceKey: uuid.UUID(moneyTransfer.IdempotenceKey()),
		GiverID:        uuid.UUID(moneyTransfer.GiverID()),
		ReceiverID:     uuid.UUID(moneyTransfer.ReceiverID()),
		Amount:         decimalAmount,
	}
}
