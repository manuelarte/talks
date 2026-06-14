package db

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/domain"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/infrastructure/db/sqlc"
	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/pagination"
)

var _ domain.AccountRepository = new(Repository)

type Repository struct {
	db      *sql.DB
	queries *sqlc.Queries
}

func NewRepository(db *sql.DB) Repository {
	return Repository{
		db:      db,
		queries: sqlc.New(db),
	}
}

func (r Repository) GetOne(ctx context.Context, id domain.AccountID) (domain.Account, error) {
	account, err := r.queries.GetAccount(ctx, uuid.UUID(id))
	if err != nil {
		return domain.Account{}, fmt.Errorf("failed to get account: %w", err)
	}

	return accountToDomain(account), nil
}

func (r Repository) GetAll(
	ctx context.Context,
	request pagination.PageRequest,
) (pagination.Page[domain.Account], error) {
	accounts, err := r.queries.GetAccounts(ctx, sqlc.GetAccountsParams{
		Limit:  int64(request.Size),
		Offset: int64(request.Number * request.Size),
	})
	if err != nil {
		return pagination.Page[domain.Account]{}, fmt.Errorf("failed to get accounts: %w", err)
	}

	count, err := r.queries.CountAccounts(ctx)
	if err != nil {
		return pagination.Page[domain.Account]{}, fmt.Errorf("failed to count accounts: %w", err)
	}

	return pagination.Page[domain.Account]{
		Content:    accountsToDomain(accounts),
		TotalCount: count,
		TotalPages: int(count / int64(request.Size)),
		Number:     request.Number,
		Size:       request.Size,
	}, nil
}

func (r Repository) SaveNewAmounts(
	ctx context.Context,
	giverID domain.GiverID,
	giverNewAmount domain.Money,
	receiverID domain.ReceiverID,
	receiverNewAmount domain.Money,
) error {
	giverAmountFloat, _ := strconv.ParseFloat(strings.TrimSpace(giverNewAmount.String()), 64)

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		_ = tx.Rollback()
	}()

	queryTx := r.queries.WithTx(tx)

	errGiver := queryTx.SaveAmount(ctx, sqlc.SaveAmountParams{
		Amount: giverAmountFloat,
		ID:     uuid.UUID(giverID),
	})
	if errGiver != nil {
		return fmt.Errorf("failed to save giver amount: %w", errGiver)
	}

	receiverAmountFloat, _ := strconv.ParseFloat(strings.TrimSpace(receiverNewAmount.String()), 64)

	errReceiver := queryTx.SaveAmount(ctx, sqlc.SaveAmountParams{
		Amount: receiverAmountFloat,
		ID:     uuid.UUID(receiverID),
	})
	if errReceiver != nil {
		return fmt.Errorf("failed to save receiver amount: %w", errReceiver)
	}

	err = tx.Commit()
	if err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func accountToDomain(account sqlc.Account) domain.Account {
	return domain.NewAccount(
		domain.AccountID(account.ID),
		domain.MustMoney(strconv.FormatFloat(account.Amount, 'f', 2, 64)),
	)
}

func accountsToDomain(accounts []sqlc.Account) []domain.Account {
	dtos := make([]domain.Account, len(accounts))
	for i, account := range accounts {
		dtos[i] = accountToDomain(account)
	}

	return dtos
}
