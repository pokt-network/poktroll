package testdelegation

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtx "github.com/cosmos/cosmos-sdk/x/auth/tx"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/delegation"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// NewLocalnetClient creates and returns a new DelegationClient that's configured for
// use with the LocalNet validator.
func NewLocalnetClient(ctx context.Context, t *testing.T) client.DelegationClient {
	t.Helper()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	deps := depinject.Supply(queryClient)
	dClient, err := delegation.NewDelegationClient(ctx, deps)
	require.NoError(t, err)

	return dClient
}

// NewAnyTimesRedelegationsSequence creates a new mock DelegationClient.
// This mock DelegationClient will expect any number of calls to RedelegationsSequence,
// and when that call is made, it returns the given EventsObservable[Redelegation].
func NewAnyTimesRedelegationsSequence(
	ctx context.Context,
	t *testing.T,
	appAddress string,
	redelegationObs observable.Observable[*apptypes.EventRedelegation],
) *mockclient.MockDelegationClient {
	t.Helper()

	// Create a mock for the delegation client which expects the
	// LastNRedelegations method to be called any number of times.
	delegationClientMock := NewAnyTimeLastNRedelegationsClient(t, appAddress)

	// Set up the mock expectation for the RedelegationsSequence method. When
	// the method is called, it returns a new replay observable that publishes
	// redelegation events sent on the given redelegationObs.
	delegationClientMock.EXPECT().
		RedelegationsSequence(ctx).
		Return(redelegationObs).
		AnyTimes()

	return delegationClientMock
}

// NewOneTimeRedelegationsSequenceDelegationClient creates a new mock
// DelegationClient. This mock DelegationClient will expect a call to
// RedelegationsSequence, and when that call is made, it returns a new
// RedelegationReplayObservable that publishes Redelegation events sent on
// the given redelegationPublishCh.
// redelegationPublishCh is the channel the caller can use to publish
// Redelegation events to the observable.
func NewOneTimeRedelegationsSequenceDelegationClient(
	ctx context.Context,
	t *testing.T,
	redelegationPublishCh chan *apptypes.EventRedelegation,
) *mockclient.MockDelegationClient {
	t.Helper()

	// Create a mock for the delegation client which expects the
	// LastNRedelegations method to be called any number of times.
	delegationClientMock := NewAnyTimeLastNRedelegationsClient(t, "")

	// Set up the mock expectation for the RedelegationsSequence method. When
	// the method is called, it returns a new replay observable that publishes
	// delegation changes sent on the given redelegationPublishCh.
	delegationClientMock.EXPECT().RedelegationsSequence(ctx).
		DoAndReturn(func(ctx context.Context) client.RedelegationReplayObservable {
			// Create a new replay observable with a replay buffer size of 1.
			// Redelegation events are published to this observable via the
			// provided redelegationPublishCh.
			withPublisherOpt := channel.WithPublisher(redelegationPublishCh)
			obs, _ := channel.NewReplayObservable[*apptypes.EventRedelegation](
				ctx, 1, withPublisherOpt,
			)
			return obs
		}).Times(1)

	delegationClientMock.EXPECT().Close().AnyTimes()

	return delegationClientMock
}

// NewAnyTimeLastNRedelegationsClient creates a mock DelegationClient that
// expects calls to the LastNRedelegations method any number of times. When
// the LastNRedelegations method is called, it returns a mock Redelegation
// with the provided appAddress.
func NewAnyTimeLastNRedelegationsClient(
	t *testing.T,
	appAddress string,
) *mockclient.MockDelegationClient {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock redelegation that returns the provided appAddress
	redelegation := &apptypes.EventRedelegation{
		Application: &apptypes.Application{
			Address:                   appAddress,
			DelegateeGatewayAddresses: []string{},
		},
	}
	// Create a mock delegation client that expects calls to
	// LastNRedelegations method and returns the mock redelegation.
	delegationClientMock := mockclient.NewMockDelegationClient(ctrl)
	delegationClientMock.EXPECT().
		LastNRedelegations(gomock.Any(), gomock.Any()).
		Return([]*apptypes.EventRedelegation{redelegation}).AnyTimes()
	delegationClientMock.EXPECT().Close().AnyTimes()

	return delegationClientMock
}

// NewRedelegationEventBytes returns a byte slice containing a JSON string
// that mocks the event bytes returned from the events query client for a
// Redelegation event.
func NewRedelegationEventBytes(
	t *testing.T,
	appAddress string,
	gatewayAddress string,
) []byte {
	t.Helper()

	txResultEvent := newRedelegationTxResultEvent(t, appAddress, gatewayAddress)
	txResultBz, err := json.Marshal(txResultEvent)
	require.NoError(t, err)

	rpcResult := &rpctypes.RPCResponse{Result: txResultBz}
	rpcResultBz, err := json.Marshal(rpcResult)
	require.NoError(t, err)

	return rpcResultBz
}

// newRedelegationTxResultEvent creates a new CometTxEvent that contains a MsgDelegateToGateway
// and the corresponding events, populated with given application and gateway addresses, which
// are emitted when an application (re)delegation is committed:
// 1. A "message" type event, injected by cosmos-sdk for all messages in a tx.
// 2. A "poktroll.application.EventRedelegation" type event, emitted by the application module.
func newRedelegationTxResultEvent(
	t *testing.T,
	appAddress string,
	gatewayAddress string,
) *tx.CometTxEvent {
	t.Helper()

	txBz := newRedelegationTxBytes(t, appAddress, gatewayAddress)
	abciEvents := newRedelegationABCIEvents(t, appAddress, gatewayAddress)
	txResultEvent := &tx.CometTxEvent{}
	txResultEvent.Data.Value.TxResult = abci.TxResult{
		Height: 999,
		Tx:     txBz,
		Result: abci.ExecTxResult{
			Code:   0,
			Data:   nil,
			Events: abciEvents,
		},
	}
	return txResultEvent
}

// newRedelegationABCIEvents creates a slice of ABCI events that are emitted
// when an application (re)delegation is committed:
// 1. A "message" type event, injected by cosmos-sdk for all messages in a tx.
// 2. A "poktroll.application.EventRedelgation" type event, emitted by the application module.
func newRedelegationABCIEvents(
	t *testing.T,
	appAddress string,
	gatewayAddress string,
) []abci.Event {
	abciEvents := make([]abci.Event, 0)
	msgEvent := newRedelegationTxMessageEvent(appAddress)

	msgABCIEvent := abci.Event(msgEvent)
	abciEvents = append(abciEvents, msgABCIEvent)

	redelegationEvent, err := cosmostypes.TypedEventToEvent(&apptypes.EventRedelegation{
		Application: &apptypes.Application{
			Address:                   appAddress,
			DelegateeGatewayAddresses: []string{gatewayAddress},
		},
	})
	require.NoError(t, err)

	return append(abciEvents, abci.Event(redelegationEvent))
}

// newRedelegationTxBytes creates a byte slice containing a JSON string that mocks
// the transaction bytes sent to the events query client for a tx which contains a
// MsgDelegateToGateway message. This transaction is NOT signed nor valid for
// use on a real network.
func newRedelegationTxBytes(
	t *testing.T,
	appAddress string,
	gatewayAddress string,
) []byte {
	t.Helper()

	registry := codectypes.NewInterfaceRegistry()
	cdc := codec.NewProtoCodec(registry)
	txCfg := authtx.NewTxConfig(cdc, authtx.DefaultSignModes)
	txBuilder := txCfg.NewTxBuilder()
	err := txBuilder.SetMsgs(&apptypes.MsgDelegateToGateway{
		AppAddress:     appAddress,
		GatewayAddress: gatewayAddress,
	})
	require.NoError(t, err)
	txBz, err := txCfg.TxEncoder()(txBuilder.GetTx())
	require.NoError(t, err)

	return txBz
}

// newRedelegationTxMessageEvent creates a new "message" type ABCI event that
// is emitted when an application (re)delegation is committed. This event type
// is normally injected by cosmos-sdk for all messages in a tx.
func newRedelegationTxMessageEvent(appAddress string) cosmostypes.Event {
	return cosmostypes.NewEvent(
		"message",
		cosmostypes.NewAttribute("action", cosmostypes.MsgTypeURL(&apptypes.MsgDelegateToGateway{})),
		cosmostypes.NewAttribute("sender", appAddress),
		cosmostypes.NewAttribute("module", "application"),
	)
}
