//go:build integration

package tx_test

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"pocket/pkg/client/tx"

	"pocket/internal/testclient/testblock"
	"pocket/internal/testclient/testeventsquery"
	"pocket/internal/testclient/testtx"
	"pocket/pkg/client"
	apptypes "pocket/x/application/types"
)

func TestTxClient_SignAndBroadcast_Integration(t *testing.T) {
	t.Skip("TODO_TECHDEBT: this test depends on some setup which is currently not implemented in this test: staked application and servicer with matching services")

	var ctx = context.Background()

	keyring, signingKey := newTestKeyringWithKey(t)

	eventsQueryClient := testeventsquery.NewLocalnetClient(t)

	_, txCtx := testtx.NewAnyTimesTxTxContext(t, keyring)

	// Construct a new mock block client because it is a required dependency. Since
	// we're not exercising transactions timeouts in this test, we don't need to set any
	// particular expectations on it, nor do we care about the value of blockHash
	// argument.
	blockClientMock := testblock.NewLocalnetClient(ctx, t)

	// Construct a new depinject config with the mocks we created above.
	txClientDeps := depinject.Supply(
		eventsQueryClient,
		txCtx,
		blockClientMock,
	)

	// Construct the transaction client.
	txClient, err := tx.NewTxClient(ctx, txClientDeps, tx.WithSigningKeyName(testSigningKeyName))
	require.NoError(t, err)

	signingKeyAddr, err := signingKey.GetAddress()
	require.NoError(t, err)

	// Construct a valid (arbitrary) message to sign, encode, and broadcast.
	appStake := types.NewCoin("upokt", types.NewInt(1000000))
	appStakeMsg := &apptypes.MsgStakeApplication{
		Address:  signingKeyAddr.String(),
		Stake:    &appStake,
		Services: client.NewTestApplicationServiceConfig(testServiceIdPrefix, 2),
	}

	// Sign and broadcast the message.
	eitherErr := txClient.SignAndBroadcast(ctx, appStakeMsg)
	err, _ = eitherErr.SyncOrAsyncError()
	require.NoError(t, err)
}
