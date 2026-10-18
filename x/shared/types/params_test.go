package types

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParams_ValidateNumBlocksPerSession(t *testing.T) {
	tests := []struct {
		desc                string
		numBlocksPerSession any
		err                 error
	}{
		{
			desc:                "invalid type",
			numBlocksPerSession: "invalid",
			err:                 ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", "invalid"),
		},
		{
			desc:                "zero NumBlocksPerSession",
			numBlocksPerSession: uint64(0),
			err:                 ErrSharedParamInvalid.Wrapf("invalid NumBlocksPerSession: (%v)", uint64(0)),
		},
		{
			desc:                "valid NumBlocksPerSession",
			numBlocksPerSession: uint64(4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateNumBlocksPerSession(tt.numBlocksPerSession)
			if tt.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateClaimWindowOpenOffsetBlocks(t *testing.T) {
	tests := []struct {
		desc                        string
		claimWindowOpenOffsetBlocks any
		err                         error
	}{
		{
			desc:                        "invalid type",
			claimWindowOpenOffsetBlocks: "invalid",
			err:                         ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", "invalid"),
		},
		{
			desc:                        "valid ClaimWindowOpenOffsetBlocks",
			claimWindowOpenOffsetBlocks: uint64(4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateClaimWindowOpenOffsetBlocks(tt.claimWindowOpenOffsetBlocks)
			if tt.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateClaimWindowCloseOffsetBlocks(t *testing.T) {
	tests := []struct {
		desc                         string
		claimWindowCloseOffsetBlocks any
		err                          error
	}{
		{
			desc:                         "invalid type",
			claimWindowCloseOffsetBlocks: "invalid",
			err:                          ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", "invalid"),
		},
		{
			desc:                         "valid ClaimWindowCloseOffsetBlocks",
			claimWindowCloseOffsetBlocks: uint64(4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateClaimWindowCloseOffsetBlocks(tt.claimWindowCloseOffsetBlocks)
			if tt.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateProofWindowOpenOffsetBlocks(t *testing.T) {
	tests := []struct {
		desc                        string
		proofWindowOpenOffsetBlocks any
		err                         error
	}{
		{
			desc:                        "invalid type",
			proofWindowOpenOffsetBlocks: "invalid",
			err:                         ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", "invalid"),
		},
		{
			// The proof seed block would precede the last block accepting claims.
			desc:                        "zero ProofWindowOpenOffsetBlocks",
			proofWindowOpenOffsetBlocks: uint64(0),
			err:                         ErrSharedParamInvalid.Wrapf("proof_window_open_offset_blocks must be at least 1, got 0"),
		},
		{
			desc:                        "valid ProofWindowOpenOffsetBlocks",
			proofWindowOpenOffsetBlocks: uint64(4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateProofWindowOpenOffsetBlocks(tt.proofWindowOpenOffsetBlocks)
			if tt.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateProofWindowCloseOffsetBlocks(t *testing.T) {
	tests := []struct {
		desc                         string
		proofWindowCloseOffsetBlocks any
		err                          error
	}{
		{
			desc:                         "invalid type",
			proofWindowCloseOffsetBlocks: "invalid",
			err:                          ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", "invalid"),
		},
		{
			desc:                         "valid ProofWindowCloseOffsetBlocks",
			proofWindowCloseOffsetBlocks: uint64(4),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateProofWindowCloseOffsetBlocks(tt.proofWindowCloseOffsetBlocks)
			if tt.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateGracePeriodEndOffsetBlocks(t *testing.T) {
	tests := []struct {
		desc                       string
		gracePeriodEndOffsetBlocks any
		err                        error
	}{
		{
			desc:                       "invalid type",
			gracePeriodEndOffsetBlocks: "invalid",
			err:                        ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", "invalid"),
		},
		{
			desc:                       "valid GracePeriodEndOffsetBlocks",
			gracePeriodEndOffsetBlocks: uint64(2),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateGracePeriodEndOffsetBlocks(tt.gracePeriodEndOffsetBlocks)
			if tt.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateSupplierUnbondingPeriodSessions(t *testing.T) {
	tests := []struct {
		desc                            string
		supplierUnbondingPeriodSessions any
		err                             error
	}{
		{
			desc:                            "invalid type",
			supplierUnbondingPeriodSessions: "invalid",
			err:                             ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", "invalid"),
		},
		{
			desc:                            "valid SupplierUnbondingPeriodSessions",
			supplierUnbondingPeriodSessions: uint64(2),
		},
		{
			desc:                            "zero SupplierUnbondingPeriodSessions",
			supplierUnbondingPeriodSessions: uint64(0),
			err:                             ErrSharedParamInvalid.Wrapf("invalid SupplierUnbondingPeriodSessions: (%v)", uint64(0)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateSupplierUnbondingPeriodSessions(tt.supplierUnbondingPeriodSessions)
			if tt.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateApplicationUnbondingPeriodSessions(t *testing.T) {
	tests := []struct {
		desc                               string
		applicationUnbondingPeriodSessions any
		err                                error
	}{
		{
			desc:                               "invalid type",
			applicationUnbondingPeriodSessions: "invalid",
			err:                                ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", "invalid"),
		},
		{
			desc:                               "valid ApplicationUnbondingPeriodSessions",
			applicationUnbondingPeriodSessions: uint64(2),
		},
		{
			desc:                               "zero ApplicationUnbondingPeriodSessions",
			applicationUnbondingPeriodSessions: uint64(0),
			err:                                ErrSharedParamInvalid.Wrapf("invalid ApplicationUnbondingPeriodSessions: (%v)", uint64(0)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateApplicationUnbondingPeriodSessions(tt.applicationUnbondingPeriodSessions)
			if tt.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateGatewayUnbondingPeriodSessions(t *testing.T) {
	tests := []struct {
		desc                           string
		gatewayUnbondingPeriodSessions any
		err                            error
	}{
		{
			desc:                           "invalid type",
			gatewayUnbondingPeriodSessions: "invalid",
			err:                            ErrSharedParamInvalid.Wrapf("invalid parameter type: %T", "invalid"),
		},
		{
			desc:                           "valid GatewayUnbondingPeriodSessions",
			gatewayUnbondingPeriodSessions: uint64(2),
		},
		{
			desc:                           "zero GatewayUnbondingPeriodSessions",
			gatewayUnbondingPeriodSessions: uint64(0),
			err:                            ErrSharedParamInvalid.Wrapf("invalid GatewayUnbondingPeriodSessions: (%v)", uint64(0)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.desc, func(t *testing.T) {
			err := ValidateGatewayUnbondingPeriodSessions(tt.gatewayUnbondingPeriodSessions)
			if tt.err != nil {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.err.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestParams_ValidateNumPendingSessionsIsPositive(t *testing.T) {
	// zeroWindowParams starts from the defaults and zeroes every claim/proof window
	// offset. The proof window open offset must be at least 1, which rejects this
	// combination before the cumulative check does.
	zeroWindowParams := DefaultParams()
	zeroWindowParams.ClaimWindowOpenOffsetBlocks = 0
	zeroWindowParams.ClaimWindowCloseOffsetBlocks = 0
	zeroWindowParams.ProofWindowOpenOffsetBlocks = 0
	zeroWindowParams.ProofWindowCloseOffsetBlocks = 0
	zeroWindowParams.GracePeriodEndOffsetBlocks = 0

	// Sanity check: without the guard, settlement would divide by this.
	require.Zero(t, GetNumPendingSessions(&zeroWindowParams),
		"test fixture must actually produce zero pending sessions",
	)

	t.Run("all-zero window offsets are rejected", func(t *testing.T) {
		err := zeroWindowParams.ValidateBasic()
		require.Error(t, err)
		require.Contains(t, err.Error(), "proof_window_open_offset_blocks must be at least 1")
	})

	t.Run("a proof window open offset of 1 is enough", func(t *testing.T) {
		params := zeroWindowParams
		params.ProofWindowOpenOffsetBlocks = 1

		require.Positive(t, GetNumPendingSessions(&params))
		require.NoError(t, params.ValidateBasic())
	})

	t.Run("defaults are unaffected", func(t *testing.T) {
		params := DefaultParams()

		require.Positive(t, GetNumPendingSessions(&params))
		require.NoError(t, params.ValidateBasic())
	})
}

// Valid params keep a supplier or application that unstakes in a session staked
// until that session's proof window closes.
func TestParams_UnbondingEndsAtOrAfterProofWindowClose(t *testing.T) {
	newParams := func(supplierUnbondingPeriodSessions, applicationUnbondingPeriodSessions uint64) Params {
		params := DefaultParams()
		params.NumBlocksPerSession = 8
		params.GracePeriodEndOffsetBlocks = 1
		params.ClaimWindowOpenOffsetBlocks = 1
		params.ClaimWindowCloseOffsetBlocks = 4
		params.ProofWindowOpenOffsetBlocks = 1
		params.ProofWindowCloseOffsetBlocks = 2
		params.SupplierUnbondingPeriodSessions = supplierUnbondingPeriodSessions
		params.ApplicationUnbondingPeriodSessions = applicationUnbondingPeriodSessions
		params.GatewayUnbondingPeriodSessions = 1
		return params
	}

	validate := func(params Params) error { return params.ValidateBasic() }

	// One session (8 blocks) ends one block before the proof window close (9 blocks after the session end).
	require.ErrorContains(t, validate(newParams(1, 2)), "SupplierUnbondingPeriodSessions")
	require.ErrorContains(t, validate(newParams(2, 1)), "ApplicationUnbondingPeriodSessions")

	params := newParams(2, 2)
	require.NoError(t, params.ValidateBasic())
	sessionEndHeight := GetSessionEndHeight(&params, 10)
	unbondingEndHeight := sessionEndHeight + int64(params.ApplicationUnbondingPeriodSessions*params.NumBlocksPerSession)
	require.GreaterOrEqual(t, unbondingEndHeight, GetProofWindowCloseHeight(&params, sessionEndHeight))
}
