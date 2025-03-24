package proof_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	keepertest "github.com/pokt-network/pocket/testutil/keeper"
	"github.com/pokt-network/pocket/testutil/nullify"
	"github.com/pokt-network/pocket/testutil/sample"
	proof "github.com/pokt-network/pocket/x/proof/module"
	"github.com/pokt-network/pocket/x/proof/types"
	sessiontypes "github.com/pokt-network/pocket/x/session/types"
)

func TestGenesis(t *testing.T) {
	mockSessionId := "mock_session_id"

	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		ClaimList: []types.Claim{
			{
				SupplierOperatorAddress: sample.AccAddress(),
				SessionHeader: &sessiontypes.SessionHeader{
					SessionId:          mockSessionId,
					ApplicationAddress: sample.AccAddress(),
				},
				RootHash: []byte{1, 2, 3},
			},
		},
		// TODO_TEST: finish genesis proof list validation.
		//ProofList: []types.Proof{
		//	{
		//		Index: "0",
		//	},
		//	{
		//		Index: "1",
		//	},
		//},
		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.ProofKeeper(t)
	proof.InitGenesis(ctx, k, genesisState)
	got := proof.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	require.ElementsMatch(t, genesisState.ClaimList, got.ClaimList)
	require.ElementsMatch(t, genesisState.ProofList, got.ProofList)
	// this line is used by starport scaffolding # genesis/test/assert
}
