package types_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/x/shared/types"
)

func TestGenesisState_Validate(t *testing.T) {
	defaultParams := types.DefaultParams()
	tests := []struct {
		desc     string
		genState *types.GenesisState
		valid    bool
	}{
		{
			desc:     "default is valid",
			genState: types.DefaultGenesis(),
			valid:    true,
		},
		{
			desc: "valid genesis state",
			genState: &types.GenesisState{
				Params: types.Params{
					NumBlocksPerSession: defaultParams.NumBlocksPerSession,
					// DEV_NOTE: the claim/proof window offsets MUST be carried over too.
					// Leaving them at their zero value makes GetNumPendingSessions() return
					// 0, which settlement divides by — see
					// validateNumPendingSessionsIsPositive. ValidateBasic rejects that, so
					// omitting them here would make this "valid" case invalid.
					GracePeriodEndOffsetBlocks:         defaultParams.GracePeriodEndOffsetBlocks,
					ClaimWindowOpenOffsetBlocks:        defaultParams.ClaimWindowOpenOffsetBlocks,
					ClaimWindowCloseOffsetBlocks:       defaultParams.ClaimWindowCloseOffsetBlocks,
					ProofWindowOpenOffsetBlocks:        defaultParams.ProofWindowOpenOffsetBlocks,
					ProofWindowCloseOffsetBlocks:       defaultParams.ProofWindowCloseOffsetBlocks,
					SupplierUnbondingPeriodSessions:    defaultParams.SupplierUnbondingPeriodSessions,
					ApplicationUnbondingPeriodSessions: defaultParams.ApplicationUnbondingPeriodSessions,
					GatewayUnbondingPeriodSessions:     defaultParams.GatewayUnbondingPeriodSessions,
					ComputeUnitsToTokensMultiplier:     defaultParams.ComputeUnitsToTokensMultiplier,
					ComputeUnitCostGranularity:         defaultParams.ComputeUnitCostGranularity,
				},

				// this line is used by starport scaffolding # types/genesis/validField
			},
			valid: true,
		},
		{
			desc: "invalid genesis state - all claim/proof window offsets are zero",
			genState: &types.GenesisState{
				// Every offset is individually valid at zero, but their sum drives
				// GetNumPendingSessions(), which settlement uses as a divisor.
				Params: types.Params{
					NumBlocksPerSession:                defaultParams.NumBlocksPerSession,
					SupplierUnbondingPeriodSessions:    defaultParams.SupplierUnbondingPeriodSessions,
					ApplicationUnbondingPeriodSessions: defaultParams.ApplicationUnbondingPeriodSessions,
					GatewayUnbondingPeriodSessions:     defaultParams.GatewayUnbondingPeriodSessions,
					ComputeUnitsToTokensMultiplier:     defaultParams.ComputeUnitsToTokensMultiplier,
					ComputeUnitCostGranularity:         defaultParams.ComputeUnitCostGranularity,
				},
			},
			valid: false,
		},
		// this line is used by starport scaffolding # types/genesis/testcase
	}
	for _, tc := range tests {
		t.Run(tc.desc, func(t *testing.T) {
			err := tc.genState.Validate()
			if tc.valid {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
			}
		})
	}
}
