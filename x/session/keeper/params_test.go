package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func TestGetParams(t *testing.T) {
	tests := []struct {
		desc                   string
		numSuppliersPerSession any
		expectedErr            error
	}{
		{
			desc:                   "invalid type",
			numSuppliersPerSession: "420",
			expectedErr:            sessiontypes.ErrSessionParamInvalid.Wrapf("invalid parameter type: string"),
		},
		{
			desc:                   "invalid NumSuppliersPerSession (<1)",
			numSuppliersPerSession: uint64(0),
			expectedErr:            sessiontypes.ErrSessionParamInvalid.Wrapf("number of suppliers per session (%d) MUST be greater than 1", 0),
		},
		{
			desc:                   "valid NumSuppliersPerSession",
			numSuppliersPerSession: uint64(420),
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := sessiontypes.ValidateNumSuppliersPerSession(test.numSuppliersPerSession)
			if test.expectedErr != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
