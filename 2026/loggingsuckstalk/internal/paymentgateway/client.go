//nolint:gosec // weak random generation is good enough
package paymentgateway

import (
	"context"
	"errors"
	"fmt"
	"math/rand/v2"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/domain"
)

var (
	ErrNotEnoughSaldo = errors.New("not enough saldo")
	ErrInternalError  = errors.New("internal error")
)

type (
	// Client is simulating a 3rd party payment gateway.
	// We are going to inject a repository just to be able to query the accounts to check the saldo.
	Client struct {
		cache    cache
		mockRepo domain.AccountRepository
	}

	TransferRequest struct {
		IdempotenceKey uuid.UUID
		GiverID        uuid.UUID
		ReceiverID     uuid.UUID
		Amount         decimal.Decimal
	}

	TransferResponse struct {
		GiverAmount       decimal.Decimal
		ReceiverAmount    decimal.Decimal
		AmountTransferred decimal.Decimal
	}

	cache struct {
		mu sync.RWMutex
		c  map[uuid.UUID]TransferResponse
	}
)

func (c *cache) get(key uuid.UUID) (TransferResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	val, ok := c.c[key]

	return val, ok
}

func (c *cache) set(key uuid.UUID, val TransferResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.c[key] = val
}

func NewClient(repo domain.AccountRepository) *Client {
	return &Client{
		cache: cache{
			c: make(map[uuid.UUID]TransferResponse),
		},
		mockRepo: repo,
	}
}

func (c *Client) Transfer(ctx context.Context, request TransferRequest) (TransferResponse, error) {
	timeToReplyMS := normalBetween(500, 6000, 2000, 1000)
	delay := time.NewTimer(time.Duration(timeToReplyMS * float64(time.Millisecond)))
	defer delay.Stop()

	for {
		select {
		case <-ctx.Done():
			return TransferResponse{}, fmt.Errorf("timeout: %w", ctx.Err())
		case <-delay.C:
			if response, ok := c.cache.get(request.IdempotenceKey); ok {
				return response, nil
			}

			giver, err := c.mockRepo.GetOne(ctx, domain.AccountID(request.GiverID))
			if err != nil {
				return TransferResponse{}, ErrInternalError
			}

			receiver, err := c.mockRepo.GetOne(ctx, domain.AccountID(request.ReceiverID))
			if err != nil {
				return TransferResponse{}, ErrInternalError
			}

			if !giver.CanTransfer(domain.MustMoney(request.Amount.String())) {
				return TransferResponse{}, ErrNotEnoughSaldo
			}

			n := rand.IntN(10)
			if n > 7 {
				return TransferResponse{}, ErrInternalError
			}

			tr := TransferResponse{
				GiverAmount:       decimal.RequireFromString(giver.Amount().String()).Sub(request.Amount),
				ReceiverAmount:    decimal.RequireFromString(receiver.Amount().String()).Add(request.Amount),
				AmountTransferred: decimal.RequireFromString(request.Amount.String()),
			}
			c.cache.set(request.IdempotenceKey, tr)

			return tr, nil
		}
	}
}

func normalBetween(minValue, maxValue, mean, stddev float64) float64 {
	for {
		v := rand.NormFloat64()*stddev + mean
		if v >= minValue && v <= maxValue {
			return v
		}
	}
}
