package domain

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"codeberg.org/manuelarte/loggingsuckstalk/internal/pagination"
)

var _ Money = new(moneyDecimal)

type (
	AccountID uuid.UUID

	//godddlint:valueObject
	Money interface {
		GreaterThan(other Money) bool
		String() string
	}

	moneyDecimal struct {
		dec decimal.Decimal
	}

	//godddlint:entity
	Account struct {
		id     AccountID
		amount Money
	}

	AccountRepository interface {
		GetOne(ctx context.Context, id AccountID) (Account, error)
		GetAll(ctx context.Context, request pagination.PageRequest) (pagination.Page[Account], error)
		SaveNewAmounts(
			ctx context.Context,
			giverID GiverID,
			giverNewAmount Money,
			receiverID ReceiverID,
			receiverNewAmount Money,
		) error
	}

	MoneyParserError struct {
		err error
	}
)

func NewMoney(amount string) (Money, error) {
	dec, err := decimal.NewFromString(amount)
	if err != nil {
		return nil, MoneyParserError{err: err}
	}

	dec = dec.Round(2)

	return moneyDecimal{dec: dec}, nil
}

func MustMoney(amount string) Money {
	money, err := NewMoney(amount)
	if err != nil {
		panic(err)
	}

	return money
}

func NewAccount(id AccountID, amount Money) Account {
	return Account{
		id:     id,
		amount: amount,
	}
}

func (a AccountID) String() string {
	return uuid.UUID(a).String()
}

func (m moneyDecimal) String() string {
	return m.dec.String()
}

func (m moneyDecimal) GreaterThan(other Money) bool {
	amount, _ := decimal.NewFromString(other.String())

	return m.dec.GreaterThan(amount)
}

func (a *Account) ID() AccountID {
	return a.id
}

func (a *Account) Amount() Money {
	return a.amount
}

func (a *Account) CanTransfer(amount Money) bool {
	return a.amount.GreaterThan(amount)
}

func (e MoneyParserError) Error() string {
	return fmt.Sprintf("Money Parser Error: %v", e.err)
}
