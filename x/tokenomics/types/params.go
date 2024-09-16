package types

import (
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

	"github.com/pokt-network/poktroll/app/volatile"
)

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	KeyComputeUnitsToTokensMultiplier            = []byte("ComputeUnitsToTokensMultiplier")
	ParamComputeUnitsToTokensMultiplier          = "compute_units_to_tokens_multiplier"
	DefaultComputeUnitsToTokensMultiplier uint64 = 42 // TODO_MAINNET: Determine the default value.
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(computeUnitsToTokensMultiplier uint64) Params {
	return Params{
		ComputeUnitsToTokensMultiplier: computeUnitsToTokensMultiplier,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(
		DefaultComputeUnitsToTokensMultiplier,
	)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyComputeUnitsToTokensMultiplier,
			&p.ComputeUnitsToTokensMultiplier,
			ValidateComputeUnitsToTokensMultiplier,
		),
	}
}

// ValidateBasic does a sanity check on the provided params.
func (params *Params) ValidateBasic() error {
	// Validate the ComputeUnitsToTokensMultiplier
	if err := ValidateComputeUnitsToTokensMultiplier(params.ComputeUnitsToTokensMultiplier); err != nil {
		return err
	}

	return nil
}

// NumComputeUnitsToCoin calculates the amount of uPOKT to mint based on the number of compute units.
func (params *Params) NumComputeUnitsToCoin(numClaimComputeUnits uint64) (sdk.Coin, error) {
	// CUPR is a LOCAL service specific parameter
	upoktAmount := math.NewInt(int64(numClaimComputeUnits * params.ComputeUnitsToTokensMultiplier))
	if upoktAmount.IsNegative() {
		return sdk.Coin{}, ErrTokenomicsRootHashInvalid.Wrap("sum * compute_units_to_tokens_multiplier is negative")
	}

	return sdk.NewCoin(volatile.DenomuPOKT, upoktAmount), nil
}

// ValidateComputeUnitsToTokensMultiplier validates the ComputeUnitsToTokensMultiplier governance parameter.
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateComputeUnitsToTokensMultiplier(v interface{}) error {
	computeUnitsToTokensMultiplier, ok := v.(uint64)
	if !ok {
		return ErrTokenomicsParamsInvalid.Wrapf("invalid parameter type: %T", v)
	}

	if computeUnitsToTokensMultiplier <= 0 {
		return ErrTokenomicsParamsInvalid.Wrapf("invalid ComputeUnitsToTokensMultiplier: (%v)", computeUnitsToTokensMultiplier)
	}

	return nil
}
