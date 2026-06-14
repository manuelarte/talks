package rest

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"

	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/domain"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/httperrors"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/logging"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/observability"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/pagination"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/services"
)

// accountsHandler handle the account endpoints.
type accountsHandler struct {
	moneyTransferService services.MoneyTransferService
	accountRepository    domain.AccountRepository
}

func (a accountsHandler) TransferMoney(
	ctx context.Context,
	request TransferMoneyRequestObject,
) (TransferMoneyResponseObject, error) {
	ctx, span := observability.StartSpan(
		ctx, "accountsHandler.GetAccounts",
		trace.WithAttributes(
			attribute.String("accountGiver", request.AccountGiver.String()),
			attribute.String("accountReceiver", request.AccountReceiver.String()),
			attribute.String("amount", request.Body.Amount),
		))
	defer span.End()

	if request.Params.IdempotencyKey == nil {
		msg := "Idempotency key is required"
		logging.AddField(ctx, "error", msg)
		logging.AddField(ctx, "statusCode", http.StatusBadRequest)
		span.SetStatus(codes.Error, msg)

		return TransferMoney400ApplicationProblemPlusJSONResponse(
			ErrorResponse{
				Detail: msg,
				Status: http.StatusBadRequest,
				Title:  "Validation error",
			}), nil
	}

	idempotenceKey := *request.Params.IdempotencyKey

	logging.AddField(ctx, "idempotenceKey", idempotenceKey.String())
	logging.AddField(ctx, "accountGiver", request.AccountGiver.String())
	logging.AddField(ctx, "accountReceiver", request.AccountReceiver.String())
	logging.AddField(ctx, "amount", request.Body.Amount)

	err := a.moneyTransferService.Transfer(
		ctx,
		domain.IdempotenceKey(idempotenceKey),
		domain.GiverID(request.AccountGiver),
		domain.ReceiverID(request.AccountReceiver),
		domain.MustMoney(request.Body.Amount),
	)
	if err != nil {
		if ve, ok := errors.AsType[httperrors.ValidationError](err); ok {
			logging.AddField(ctx, "error", err.Error())
			logging.AddField(ctx, "statusCode", http.StatusBadRequest)
			span.SetStatus(codes.Error, err.Error())

			return TransferMoney400ApplicationProblemPlusJSONResponse(
				ErrorResponse{
					Detail: ve.Message,
					Status: http.StatusBadRequest,
					Title:  ve.Title,
				}), nil
		}

		if ise, ok := errors.AsType[httperrors.InternalServerError](err); ok {
			logging.AddField(ctx, "error", err.Error())
			logging.AddField(ctx, "statusCode", http.StatusInternalServerError)
			span.SetStatus(codes.Error, ise.Message)

			return TransferMoney400ApplicationProblemPlusJSONResponse(
				ErrorResponse{
					Detail: ise.Message,
					Status: 500,
					Title:  ise.Title,
				}), nil
		}
	}

	logging.AddField(ctx, "statusCode", http.StatusOK)

	return TransferMoney200JSONResponse{
		AccountGiver:    request.AccountGiver,
		AccountReceiver: request.AccountReceiver,
		Amount:          request.Body.Amount,
	}, nil
}

func (a accountsHandler) GetAccounts(
	ctx context.Context,
	request GetAccountsRequestObject,
) (GetAccountsResponseObject, error) {
	ctx, span := observability.StartSpan(
		ctx, "accountsHandler.GetAccounts",
	)
	defer span.End()

	size := getOrDefault(request.Params.Size, 10)
	number := getOrDefault(request.Params.Page, 0)
	pageRequest := pagination.PageRequest{
		Number: number,
		Size:   size,
	}

	pageAccounts, err := a.accountRepository.GetAll(ctx, pageRequest)
	if err != nil {
		return GetAccounts500ApplicationProblemPlusJSONResponse{
			Detail: err.Error(),
			Status: 500,
			Title:  "Can't get accounts",
		}, fmt.Errorf("can't get accounts: %w", err)
	}

	next := Paths{}.GetAccountsEndpoint.Path(GetAccountsEndpointQueryParams{
		Page: strconv.Itoa(number + 1),
		Size: strconv.Itoa(size),
	})

	prev := Paths{}.GetAccountsEndpoint.Path(GetAccountsEndpointQueryParams{
		Page: strconv.Itoa(number - 1),
		Size: strconv.Itoa(size),
	})
	if number == 0 {
		prev = ""
	}

	if number == pageAccounts.TotalPages-1 {
		next = ""
	}

	paginateMetadata := Page{
		Next:   next,
		Number: number,
		Prev:   prev,
		Self: Paths{}.GetAccountsEndpoint.Path(GetAccountsEndpointQueryParams{
			Page: strconv.Itoa(number),
			Size: strconv.Itoa(size),
		}),
		Size:       size,
		TotalCount: pageAccounts.TotalCount,
		TotalPages: pageAccounts.TotalPages,
	}

	return GetAccounts200JSONResponse{
		Accounts: accountsToDto(pageAccounts.Content),
		Page:     paginateMetadata,
	}, nil
}

func (a accountsHandler) GetAccount(
	ctx context.Context,
	request GetAccountRequestObject,
) (GetAccountResponseObject, error) {
	ctx, span := observability.StartSpan(
		ctx, "accountsHandler.GetOne",
	)
	defer span.End()

	account, err := a.accountRepository.GetOne(ctx, domain.AccountID(request.Id))
	if err != nil {
		return GetAccount500ApplicationProblemPlusJSONResponse{
			Detail: err.Error(),
			Status: 500,
			Title:  "Can't get account",
		}, fmt.Errorf("can't get account: %w", err)
	}

	return GetAccount200JSONResponse(accountToDto(account)), nil
}

func accountToDto(account domain.Account) Account {
	return Account{
		Id:     uuid.UUID(account.ID()),
		Amount: account.Amount().String(),
		Self:   Paths{}.GetAccountEndpoint.Path(account.ID().String()),
	}
}

func accountsToDto(accounts []domain.Account) []Account {
	dtos := make([]Account, len(accounts))
	for i, account := range accounts {
		dtos[i] = accountToDto(account)
	}

	return dtos
}
