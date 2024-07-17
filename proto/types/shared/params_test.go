package shared

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
