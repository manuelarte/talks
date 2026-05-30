//nolint:gosec // weak random generation is good enough
package pub

import (
	"errors"
	"math/rand/v2"

	"codeberg.org/manuelarte/loggingsuckstalk/internal/domain"
)

type Pub struct{}

func (p Pub) PublishMoneyTransfer(_ domain.MoneyTransfer) error {
	n := rand.IntN(10)
	if n > 3 {
		return errors.New("failed to publish money transfer")
	}

	return nil
}
