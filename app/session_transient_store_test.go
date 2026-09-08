package app_test

import (
	"testing"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client/flags"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

// TestSessionTransientStoreIsMounted guards the wiring of the session module's
// transient store, which backs the block-scoped hydrated-session memo.
// runtime registers "transient:<module>" only when the module declares a
// store.TransientStoreService depinject input (x/session/module/depinject.go),
// so a mounted key proves x/session received a non-nil service. If that input
// is dropped the memo silently disables itself.
func TestSessionTransientStoreIsMounted(t *testing.T) {
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()

	pocketApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)

	// depinject names transient keys "transient:<module>".
	storeKey := pocketApp.UnsafeFindStoreKey("transient:" + sessiontypes.ModuleName)
	require.NotNil(t, storeKey, "session transient store key is not registered on the app")

	commitStore := pocketApp.CommitMultiStore()
	require.Equal(t, storetypes.StoreTypeTransient, commitStore.GetCommitKVStore(storeKey).GetStoreType())
}
