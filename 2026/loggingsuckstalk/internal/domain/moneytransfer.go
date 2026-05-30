package domain

import (
	"github.com/google/uuid"
)

type (
	IdempotenceKey uuid.UUID

	GiverID    AccountID
	ReceiverID AccountID

	MoneyTransfer struct {
		idempotenceKey IdempotenceKey
		giverID        GiverID
		receiverID     ReceiverID
		amount         Money
	}
)

func (i IdempotenceKey) String() string {
	return uuid.UUID(i).String()
}

func (g GiverID) String() string {
	return uuid.UUID(g).String()
}

func (r ReceiverID) String() string {
	return uuid.UUID(r).String()
}

func NewMoneyTransfer(
	idempotenceKey IdempotenceKey,
	giverID GiverID,
	receiverID ReceiverID,
	amount Money,
) MoneyTransfer {
	return MoneyTransfer{
		idempotenceKey: idempotenceKey,
		giverID:        giverID,
		receiverID:     receiverID,
		amount:         amount,
	}
}

func (m MoneyTransfer) IdempotenceKey() IdempotenceKey {
	return m.idempotenceKey
}

func (m MoneyTransfer) GiverID() GiverID {
	return m.giverID
}

func (m MoneyTransfer) ReceiverID() ReceiverID {
	return m.receiverID
}

func (m MoneyTransfer) Amount() Money {
	return m.amount
}
