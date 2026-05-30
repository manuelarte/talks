package services

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"

	"codeberg.org/manuelarte/loggingsuckstalk/internal/domain"
	"codeberg.org/manuelarte/loggingsuckstalk/internal/httperrors"
	"codeberg.org/manuelarte/loggingsuckstalk/internal/infrastructure/pub"
	"codeberg.org/manuelarte/loggingsuckstalk/internal/logging"
	"codeberg.org/manuelarte/loggingsuckstalk/internal/observability"
	"codeberg.org/manuelarte/loggingsuckstalk/internal/paymentgateway"
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

	logger := logging.FromContext(ctx).With(
		slog.String("idempotenceKey", idempotenceKey.String()),
		slog.String("giverID", giverID.String()),
		slog.String("receiverID", receiverID.String()),
		slog.String("amount", amount.String()),
	)

	if _, ok := s.cache[idempotenceKey]; ok {
		logger.InfoContext(ctx, "Money Transfer already processed")

		return nil
	}

	logger.InfoContext(ctx, "New money transfer request received")

	moneyTransfer := domain.NewMoneyTransfer(idempotenceKey, giverID, receiverID, amount)

	logger.InfoContext(ctx, "[PaymentGateway] Processing money transfer")

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

	pgElapsed := time.Since(pgStartTime)
	logging.AddField(ctx, "paymentGatewayElapsed", pgElapsed)

	if err != nil {
		logging.AddField(ctx, "paymentGateway", "error")

		switch {
		case errors.Is(err, paymentgateway.ErrNotEnoughSaldo):
			logger.WarnContext(ctx, "[PaymentGateway]: Validation error", slog.Any("err", err))
			logging.AddField(ctx, "paymentGatewayError", paymentgateway.ErrNotEnoughSaldo)

			return httperrors.ValidationError{
				Title:   "Saldo error",
				Message: "Not enough saldo",
			}
		default:
			logger.ErrorContext(ctx, "[PaymentGateway]: Internal server error", slog.Any("err", err))
			logging.AddField(ctx, "paymentGatewayError", err.Error())

			return httperrors.InternalServerError{
				Title:   "Payment gateway error",
				Message: err.Error(),
			}
		}
	}

	logging.AddField(ctx, "paymentGateway", "success")
	logger.InfoContext(ctx, "[PaymentGateway]: Money transfer processed")
	// sending sent
	err = s.pub.PublishMoneyTransfer(moneyTransfer)
	if err != nil {
		logger.ErrorContext(ctx, "Failed to publish money transfer event", slog.Any("err", err))
		logging.AddField(ctx, "kafkaEvent", "error")
		logging.AddField(ctx, "kafkaEventError", err.Error())
	} else {
		logger.InfoContext(ctx, "Money transferred event sent")
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
		logger.InfoContext(ctx, "Accounts updated")
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
