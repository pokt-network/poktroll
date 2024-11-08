package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestParams_ValidateMintAllocationDao(t *testing.T) {
	tests := []struct {
		desc             string
		mintAllocatioDao any
		expectedErr      error
	}{
		{
			desc:             "invalid type",
			mintAllocatioDao: "0",
			expectedErr:      tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:             "valid MintAllocationDao",
			mintAllocatioDao: tokenomicstypes.DefaultMintAllocationDao,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintAllocationDao(test.mintAllocatioDao)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
