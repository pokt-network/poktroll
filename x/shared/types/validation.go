package types

import (
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/pocket"
)

// ValidatePositiveuPOKT performs the following validation steps:
//   - Validates the coin string is a valid cosmos coin
//   - Validates the coin is positive (amount > 0)
//   - Validates the coin denom is upokt
func ValidatePositiveuPOKT(coinStr string) error {
	coin, err := cosmostypes.ParseCoinNormalized(coinStr)
	if err != nil {
		return fmt.Errorf("while parsing coin %s; (%s)", coinStr, err)
	}
	if !coin.IsValid() {
		return fmt.Errorf("invalid coin %s; (%s)", coinStr, coin.Validate())
	}
	if coin.IsZero() || coin.IsNegative() {
		return fmt.Errorf("invalid coin amount: %s <= 0", coinStr)
	}
	if coin.Denom != pocket.DenomuPOKT {
		return fmt.Errorf("invalid coin denom: expected %s, got %s", pocket.DenomuPOKT, coin.Denom)
	}
	return nil
}
