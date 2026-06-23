//nolint:gosec // weak random generation is good enough
package pub

import (
	"errors"
	"math/rand/v2"

	"github.com/manuelarte/talks/2026/loggingsuckstalk/internal/domain"
)

type Pub struct{}

func (p Pub) PublishMoneyTransfer(_ domain.MoneyTransfer) error {
	n := rand.IntN(10)
	if n > 8 {
		return errors.New("connection timeout")
	}

	return nil
}
