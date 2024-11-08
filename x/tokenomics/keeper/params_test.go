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
			desc:             "invalid MintAllocationDao (<0)",
			mintAllocatioDao: -0.1,
			expectedErr:      tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint allocation to DAO must be greater than or equal to 0: got %f", -0.1),
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

func TestParams_ValidateMintAllocationProposer(t *testing.T) {
	tests := []struct {
		desc                  string
		mintAllocatioProposer any
		expectedErr           error
	}{
		{
			desc:                  "invalid type",
			mintAllocatioProposer: "0",
			expectedErr:           tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:                  "invalid MintAllocationProposer (<0)",
			mintAllocatioProposer: -0.1,
			expectedErr:           tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint allocation to proposer must be greater than or equal to 0: got %f", -0.1),
		},
		{
			desc:                  "valid MintAllocationProposer",
			mintAllocatioProposer: tokenomicstypes.DefaultMintAllocationProposer,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintAllocationProposer(test.mintAllocatioProposer)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
