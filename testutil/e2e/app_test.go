package e2e

import (
	"testing"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	comethttp "github.com/cometbft/cometbft/rpc/client/http"
	cosmostx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	txtypes "github.com/pokt-network/poktroll/pkg/client/tx/types"
	"github.com/pokt-network/poktroll/testutil/testclient"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestNewE2EApp(t *testing.T) {
	app := NewE2EApp(t)

	// Construct dependencies...
	keyRing := keyring.NewInMemory(app.GetCodec())
	rec, err := keyRing.NewAccount(
		"gateway2",
		"suffer wet jelly furnace cousin flip layer render finish frequent pledge feature economy wink like water disease final erase goat include apple state furnace",
		"",
		cosmostypes.FullFundraiserPath,
		hd.Secp256k1,
	)
	require.NoError(t, err)

	gateway2Addr, err := rec.GetAddress()
	require.NoError(t, err)

	blockQueryClient, err := comethttp.New("tcp://127.0.0.1:42070", "/websocket")
	require.NoError(t, err)

	creds := insecure.NewCredentials()
	grpcConn, err := grpc.NewClient("127.0.0.1:42069", grpc.WithTransportCredentials(creds))
	require.NoError(t, err)

	deps := depinject.Supply(grpcConn, blockQueryClient)

	sharedQueryClient, err := query.NewSharedQuerier(deps)
	require.NoError(t, err)

	sharedParams, err := sharedQueryClient.GetParams(app.GetSdkCtx())
	require.NoError(t, err)
	require.Equal(t, sharedtypes.DefaultParams(), *sharedParams)

	eventsQueryClient := events.NewEventsQueryClient("ws://127.0.0.1:6969/websocket")
	deps = depinject.Configs(deps, depinject.Supply(eventsQueryClient))
	blockClient, err := block.NewBlockClient(app.GetSdkCtx(), deps)
	require.NoError(t, err)

	flagSet := testclient.NewFlagSet(t, "tcp://127.0.0.1:42070")
	// DEV_NOTE: DO NOT use the clientCtx as a grpc.ClientConn as it bypasses E2EApp integrations.
	clientCtx := testclient.NewLocalnetClientCtx(t, flagSet).WithKeyring(keyRing)

	txFactory, err := cosmostx.NewFactoryCLI(clientCtx, flagSet)
	require.NoError(t, err)

	deps = depinject.Configs(deps, depinject.Supply(txtypes.Context(clientCtx), txFactory))

	txContext, err := tx.NewTxContext(deps)
	require.NoError(t, err)

	deps = depinject.Configs(deps, depinject.Supply(blockClient, txContext))
	txClient, err := tx.NewTxClient(app.GetSdkCtx(), deps, tx.WithSigningKeyName("gateway2"))
	require.NoError(t, err)

	// Assert that no gateways are staked.
	gatewayQueryClient := gatewaytypes.NewQueryClient(grpcConn)
	allGatewaysRes, err := gatewayQueryClient.AllGateways(app.GetSdkCtx(), &gatewaytypes.QueryAllGatewaysRequest{})
	require.Equal(t, 0, len(allGatewaysRes.Gateways))

	// Fund gateway2 account.
	_, err = app.RunMsg(t, &banktypes.MsgSend{
		FromAddress: app.GetFaucetBech32(),
		ToAddress:   gateway2Addr.String(),
		Amount:      cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 10000000000)),
	})
	require.NoError(t, err)

	// Stake gateway2.
	eitherErr := txClient.SignAndBroadcast(
		app.GetSdkCtx(),
		gatewaytypes.NewMsgStakeGateway(
			"pokt15w3fhfyc0lttv7r585e2ncpf6t2kl9uh8rsnyz",
			cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(100000001)),
		),
	)

	err, errCh := eitherErr.SyncOrAsyncError()
	require.NoError(t, err)
	require.NoError(t, <-errCh)

	// Assert that only gateway2 is staked.
	allGatewaysRes, err = gatewayQueryClient.AllGateways(app.GetSdkCtx(), &gatewaytypes.QueryAllGatewaysRequest{})
	require.Equal(t, 1, len(allGatewaysRes.Gateways))
	require.Equal(t, "pokt15w3fhfyc0lttv7r585e2ncpf6t2kl9uh8rsnyz", allGatewaysRes.Gateways[0].Address)
}
