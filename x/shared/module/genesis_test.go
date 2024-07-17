package shared_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/proto/types/shared"
	keepertest "github.com/pokt-network/poktroll/testutil/keeper"
	"github.com/pokt-network/poktroll/testutil/nullify"
	sharedmodule "github.com/pokt-network/poktroll/x/shared/module"
)

func TestGenesis(t *testing.T) {
	genesisState := shared.GenesisState{
		Params: shared.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.SharedKeeper(t)
	sharedmodule.InitGenesis(ctx, k, genesisState)
	got := sharedmodule.ExportGenesis(ctx, k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
