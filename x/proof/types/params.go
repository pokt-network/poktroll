package types

import paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

var (
	_ paramtypes.ParamSet = (*Params)(nil)

	KeyMinRelayDifficultyBits            = []byte("MinRelayDifficultyBits")
	ParamMinRelayDifficultyBits          = "min_relay_difficulty_bits"
	DefaultMinRelayDifficultyBits uint64 = 0 // TODO_MAINNET(#142, #401): Determine the default value.
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(minRelayDifficultyBits uint64) Params {
	return Params{
		MinRelayDifficultyBits: minRelayDifficultyBits,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultMinRelayDifficultyBits)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyMinRelayDifficultyBits,
			&p.MinRelayDifficultyBits,
			ValidateMinRelayDifficultyBits,
		),
	}
}

// ValidateBasic does a sanity check on the provided params.
func (params *Params) ValidateBasic() error {
	// Validate the ComputeUnitsToTokensMultiplier
	if err := ValidateMinRelayDifficultyBits(params.MinRelayDifficultyBits); err != nil {
		return err
	}

	return nil
}

// validateMinRelayDifficultyBits validates the MinRelayDifficultyBits param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateMinRelayDifficultyBits(v interface{}) error {
	difficulty, ok := v.(uint64)
	if !ok {
		return ErrProofParamInvalid.Wrapf("invalid parameter type: %T", v)
	}

	if difficulty < 0 {
		return ErrProofParamInvalid.Wrapf("invalid MinRelayDifficultyBits: (%v)", difficulty)
	}

	return nil
}
