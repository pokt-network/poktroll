package e2e

import (
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/testutil/testclient"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
)

func TestNewE2EApp(t *testing.T) {
	app := NewE2EApp(t)

	blockQueryClient, err := sdkclient.NewClientFromNode("tcp://127.0.0.1:42070")
	require.NoError(t, err)

	deps := depinject.Supply(app.QueryHelper(), blockQueryClient)

	sharedQueryClient, err := query.NewSharedQuerier(deps)
	require.NoError(t, err)

	sharedParams, err := sharedQueryClient.GetParams(app.GetSdkCtx())
	require.NoError(t, err)

	t.Logf("shared params: %+v", sharedParams)

	eventsQueryClient := events.NewEventsQueryClient("ws://127.0.0.1:6969/websocket")
	deps = depinject.Configs(deps, depinject.Supply(eventsQueryClient))
	blockClient, err := block.NewBlockClient(app.GetSdkCtx(), deps)
	require.NoError(t, err)

	keyRing := keyring.NewInMemory(app.GetCodec())
	// TODO: add the gateway2 key...
	_, err = keyRing.NewAccount(
		"gateway2",
		"suffer wet jelly furnace cousin flip layer render finish frequent pledge feature economy wink like water disease final erase goat include apple state furnace",
		"",
		cosmostypes.FullFundraiserPath,
		hd.Secp256k1,
	)
	require.NoError(t, err)

	flagSet := testclient.NewLocalnetFlagSet(t)
	clientCtx := testclient.NewLocalnetClientCtx(t, flagSet).WithKeyring(keyRing)

	txFactory, err := cosmostx.NewFactoryCLI(clientCtx, flagSet)
	require.NoError(t, err)

	deps = depinject.Configs(deps, depinject.Supply(txtypes.Context(clientCtx), txFactory))
	txContext, err := tx.NewTxContext(deps)
	require.NoError(t, err)

	deps = depinject.Configs(deps, depinject.Supply(blockClient, txContext))
	txClient, err := tx.NewTxClient(app.GetSdkCtx(), deps, tx.WithSigningKeyName("gateway2"))
	require.NoError(t, err)

	eitherErr := txClient.SignAndBroadcast(
		app.GetSdkCtx(),
		gatewaytypes.NewMsgStakeGateway(
			"pokt15w3fhfyc0lttv7r585e2ncpf6t2kl9uh8rsnyz",
			cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(100000000)),
		),
	)
	err, errCh := eitherErr.SyncOrAsyncError()
	require.NoError(t, err)
	require.NoError(t, <-errCh)
}
