package types

import paramtypes "github.com/cosmos/cosmos-sdk/x/params/types"

var (
	KeyNumSuppliersPerSession            = []byte("NumSuppliersPerSession")
	ParamNumSuppliersPerSession          = "num_suppliers_per_session"
	DefaultNumSuppliersPerSession uint64 = 15

	_ paramtypes.ParamSet = (*Params)(nil)
)

// ParamKeyTable the param key table for launch module
func ParamKeyTable() paramtypes.KeyTable {
	return paramtypes.NewKeyTable().RegisterParamSet(&Params{})
}

// NewParams creates a new Params instance
func NewParams(numSuppliersPerSession uint64) Params {
	return Params{
		NumSuppliersPerSession: numSuppliersPerSession,
	}
}

// DefaultParams returns a default set of parameters
func DefaultParams() Params {
	return NewParams(DefaultNumSuppliersPerSession)
}

// ParamSetPairs get the params.ParamSet
func (p *Params) ParamSetPairs() paramtypes.ParamSetPairs {
	return paramtypes.ParamSetPairs{
		paramtypes.NewParamSetPair(
			KeyNumSuppliersPerSession,
			&p.NumSuppliersPerSession,
			ValidateNumSuppliersPerSession,
		),
	}
}

// ValidateBasic does a sanity check on the provided params.
func (p Params) ValidateBasic() error {
	if err := ValidateNumSuppliersPerSession(p.NumSuppliersPerSession); err != nil {
		return err
	}

	return nil
}

// ValidateNumSuppliersPerSession validates the NumSuppliersPerSession param.
func ValidateNumSuppliersPerSession(numSuppliersPerSessionAny any) error {
	numSuppliersPerSession, ok := numSuppliersPerSessionAny.(uint64)
	if !ok {
		return ErrSessionParamInvalid.Wrapf("invalid parameter type: %T", numSuppliersPerSessionAny)
	}

	if numSuppliersPerSession < 1 {
		return ErrSessionParamInvalid.Wrapf("number of suppliers per session (%d) MUST be greater than or equal to 1", numSuppliersPerSession)
	}

	return nil
}
