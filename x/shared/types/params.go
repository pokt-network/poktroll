package types

import (
	paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"
)

const (
	DefaultNumBlocksPerSession = 4
	ParamNumBlocksPerSession   = "num_blocks_per_session"
)

var (
	_                      paramtypes.ParamSet = (*Params)(nil)
	KeyNumBlocksPerSession                     = []byte("NumBlocksPerSession")
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams() Params {
	return Params{
		NumBlocksPerSession: DefaultNumBlocksPerSession,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams()
}

// ParamSetPairs get the params.ParamSet
func (params *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyNumBlocksPerSession,
			&params.NumBlocksPerSession,
			ValidateNumBlocksPerSession,
		),
	}
}

// Validate validates the set of params
func (params *Params) ValidateBasic() error {
	if err := ValidateNumBlocksPerSession(params.NumBlocksPerSession); err != nil {
		return err
	}

	return nil
}

// ValidateNumBlocksPerSession validates the NumBlocksPerSession param
// NB: The argument is an interface type to satisfy the ParamSetPair function signature.
func ValidateNumBlocksPerSession(v interface{}) error {
	numBlocksPerSession, ok := v.(uint64)
	if !ok {
		return ErrSessionParamInvalid.Wrapf("invalid parameter type: %T", v)
	}

	if numBlocksPerSession < 1 {
		return ErrSessionParamInvalid.Wrapf("invalid NumBlocksPerSession: (%v)", numBlocksPerSession)
	}

	return nil
}
