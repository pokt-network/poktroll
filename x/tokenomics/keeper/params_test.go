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

func TestParams_ValidateMintAllocationSupplier(t *testing.T) {
	tests := []struct {
		desc                  string
		mintAllocatioSupplier any
		expectedErr           error
	}{
		{
			desc:                  "invalid type",
			mintAllocatioSupplier: "0",
			expectedErr:           tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:                  "invalid MintAllocationSupplier (<0)",
			mintAllocatioSupplier: -0.1,
			expectedErr:           tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint allocation to supplier must be greater than or equal to 0: got %f", -0.1),
		},
		{
			desc:                  "valid MintAllocationSupplier",
			mintAllocatioSupplier: tokenomicstypes.DefaultMintAllocationSupplier,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintAllocationSupplier(test.mintAllocatioSupplier)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateMintAllocationSourceOwner(t *testing.T) {
	tests := []struct {
		desc                     string
		mintAllocatioSourceOwner any
		expectedErr              error
	}{
		{
			desc:                     "invalid type",
			mintAllocatioSourceOwner: "0",
			expectedErr:              tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:                     "invalid MintAllocationSourceOwner (<0)",
			mintAllocatioSourceOwner: -0.1,
			expectedErr:              tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint allocation to source owner must be greater than or equal to 0: got %f", -0.1),
		},
		{
			desc:                     "valid MintAllocationSourceOwner",
			mintAllocatioSourceOwner: tokenomicstypes.DefaultMintAllocationSourceOwner,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintAllocationSourceOwner(test.mintAllocatioSourceOwner)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateMintAllocationApplication(t *testing.T) {
	tests := []struct {
		desc                     string
		mintAllocatioApplication any
		expectedErr              error
	}{
		{
			desc:                     "invalid type",
			mintAllocatioApplication: "0",
			expectedErr:              tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:                     "invalid MintAllocationApplication (<0)",
			mintAllocatioApplication: -0.1,
			expectedErr:              tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint allocation to application must be greater than or equal to 0: got %f", -0.1),
		},
		{
			desc:                     "valid MintAllocationApplication",
			mintAllocatioApplication: tokenomicstypes.DefaultMintAllocationApplication,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintAllocationApplication(test.mintAllocatioApplication)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateDaoRewardAddress(t *testing.T) {
	tests := []struct {
		desc             string
		daoRewardAddress any
		expectedErr      error
	}{
		{
			desc:             "invalid type",
			daoRewardAddress: int64(0),
			expectedErr:      tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: int64"),
		},
		{
			desc:             "invalid bech32 DaoRewardAddress",
			daoRewardAddress: "not_a_bech32",
			expectedErr:      tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("invalid dao reward address %q: decoding bech32 failed: invalid separator index -1", "not_a_bech32"),
		},
		{
			desc:             "valid DaoRewardAddress",
			daoRewardAddress: tokenomicstypes.DefaultDaoRewardAddress,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateDaoRewardAddress(test.daoRewardAddress)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
