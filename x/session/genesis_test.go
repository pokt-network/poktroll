package session_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	keepertest "pocket/testutil/keeper"
	"pocket/testutil/nullify"
	"pocket/x/session"
	"pocket/x/session/types"
)

func TestGenesis(t *testing.T) {
	genesisState := types.GenesisState{
		Params: types.DefaultParams(),

		// this line is used by starport scaffolding # genesis/test/state
	}

	k, ctx := keepertest.SessionKeeper(t)
	session.InitGenesis(ctx, *k, genesisState)
	got := session.ExportGenesis(ctx, *k)
	require.NotNil(t, got)

	nullify.Fill(&genesisState)
	nullify.Fill(got)

	// this line is used by starport scaffolding # genesis/test/assert
}
