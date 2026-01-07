package keeper_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

func TestParams_ValidateMintAllocationDao(t *testing.T) {
	tests := []struct {
		desc              string
		mintAllocationDao any
		expectedErr       error
	}{
		{
			desc:              "invalid type",
			mintAllocationDao: "0",
			expectedErr:       tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:              "invalid MintAllocationDao (<0)",
			mintAllocationDao: -0.1,
			expectedErr:       tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint allocation to DAO must be greater than or equal to 0: got %f", -0.1),
		},
		{
			desc:              "valid MintAllocationDao",
			mintAllocationDao: tokenomicstypes.DefaultMintAllocationPercentages.Dao,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintAllocationDao(test.mintAllocationDao)
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
		desc                   string
		mintAllocationProposer any
		expectedErr            error
	}{
		{
			desc:                   "invalid type",
			mintAllocationProposer: "0",
			expectedErr:            tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:                   "invalid MintAllocationProposer (<0)",
			mintAllocationProposer: -0.1,
			expectedErr:            tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint allocation to proposer must be greater than or equal to 0: got %f", -0.1),
		},
		{
			desc:                   "valid MintAllocationProposer",
			mintAllocationProposer: tokenomicstypes.DefaultMintAllocationPercentages.Proposer,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintAllocationProposer(test.mintAllocationProposer)
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
		desc                   string
		mintAllocationSupplier any
		expectedErr            error
	}{
		{
			desc:                   "invalid type",
			mintAllocationSupplier: "0",
			expectedErr:            tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:                   "invalid MintAllocationSupplier (<0)",
			mintAllocationSupplier: -0.1,
			expectedErr:            tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint allocation to supplier must be greater than or equal to 0: got %f", -0.1),
		},
		{
			desc:                   "valid MintAllocationSupplier",
			mintAllocationSupplier: tokenomicstypes.DefaultMintAllocationPercentages.Supplier,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintAllocationSupplier(test.mintAllocationSupplier)
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
		desc                      string
		mintAllocationSourceOwner any
		expectedErr               error
	}{
		{
			desc:                      "invalid type",
			mintAllocationSourceOwner: "0",
			expectedErr:               tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:                      "invalid MintAllocationSourceOwner (<0)",
			mintAllocationSourceOwner: -0.1,
			expectedErr:               tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint allocation to source owner must be greater than or equal to 0: got %f", -0.1),
		},
		{
			desc:                      "valid MintAllocationSourceOwner",
			mintAllocationSourceOwner: tokenomicstypes.DefaultMintAllocationPercentages.SourceOwner,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintAllocationSourceOwner(test.mintAllocationSourceOwner)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateMintApplication(t *testing.T) {
	tests := []struct {
		desc                      string
		mintAllocationApplication any
		expectedErr               error
	}{
		{
			desc:                      "invalid type",
			mintAllocationApplication: "0",
			expectedErr:               tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:                      "invalid MintAllocationApplication (<0)",
			mintAllocationApplication: -0.1,
			expectedErr:               tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint allocation to application must be greater than or equal to 0: got %f", -0.1),
		},
		{
			desc:                      "valid MintAllocationApplication",
			mintAllocationApplication: tokenomicstypes.DefaultMintAllocationPercentages.Application,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintApplication(test.mintAllocationApplication)
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

func TestParams_ValidateGlobalInflationPerClaim(t *testing.T) {
	tests := []struct {
		desc                    string
		globalInflationPerClaim any
		expectedErr             error
	}{
		{
			desc:                    "invalid type",
			globalInflationPerClaim: float32(0.111),
			expectedErr:             tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: float32"),
		},
		{
			desc:                    "less than zero",
			globalInflationPerClaim: float64(-0.1),
			expectedErr: tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf(
				"GlobalInflationPerClaim must be greater than or equal to 0: %f", float64(-0.1),
			),
		},
		{
			desc:                    "valid GlobalInflationPerClaim",
			globalInflationPerClaim: tokenomicstypes.DefaultGlobalInflationPerClaim,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateGlobalInflationPerClaim(test.globalInflationPerClaim)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestParams_ValidateMintRatio tests the validation of the MintRatio parameter (PIP-41).
func TestParams_ValidateMintRatio(t *testing.T) {
	tests := []struct {
		desc        string
		mintRatio   any
		expectedErr error
	}{
		{
			desc:        "invalid type - string",
			mintRatio:   "0.975",
			expectedErr: tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: string"),
		},
		{
			desc:        "invalid type - int",
			mintRatio:   1,
			expectedErr: tokenomicstypes.ErrTokenomicsParamInvalid.Wrap("invalid parameter type: int"),
		},
		{
			desc:        "negative value",
			mintRatio:   float64(-0.5),
			expectedErr: tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint_ratio must be in range (0, 1]: got %f", float64(-0.5)),
		},
		{
			desc:        "greater than 1",
			mintRatio:   float64(1.1),
			expectedErr: tokenomicstypes.ErrTokenomicsParamInvalid.Wrapf("mint_ratio must be in range (0, 1]: got %f", float64(1.1)),
		},
		{
			desc:      "zero (allowed for upgrade compatibility)",
			mintRatio: float64(0),
			// Zero is allowed during validation - ValidateBasic sets it to default
			expectedErr: nil,
		},
		{
			desc:        "valid 0.975 (PIP-41 target)",
			mintRatio:   float64(0.975),
			expectedErr: nil,
		},
		{
			desc:        "valid 1.0 (default - no deflation)",
			mintRatio:   tokenomicstypes.DefaultMintRatio,
			expectedErr: nil,
		},
		{
			desc:        "valid edge case - very small",
			mintRatio:   float64(0.001),
			expectedErr: nil,
		},
		{
			desc:        "valid edge case - exactly 1",
			mintRatio:   float64(1.0),
			expectedErr: nil,
		},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			err := tokenomicstypes.ValidateMintRatio(test.mintRatio)
			if test.expectedErr != nil {
				require.ErrorContains(t, err, test.expectedErr.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}
