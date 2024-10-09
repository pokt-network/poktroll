package testdelegation

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/cometbft/cometbft/libs/json"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/delegation"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/testclient/testeventsquery"
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
	redelegationObs observable.Observable[client.Redelegation],
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
	redelegationPublishCh chan client.Redelegation,
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
			obs, _ := channel.NewReplayObservable[client.Redelegation](
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
	redelegation := NewAnyTimesRedelegation(t, appAddress, "")
	// Create a mock delegation client that expects calls to
	// LastNRedelegations method and returns the mock redelegation.
	delegationClientMock := mockclient.NewMockDelegationClient(ctrl)
	delegationClientMock.EXPECT().
		LastNRedelegations(gomock.Any(), gomock.Any()).
		Return([]client.Redelegation{redelegation}).AnyTimes()
	delegationClientMock.EXPECT().Close().AnyTimes()

	return delegationClientMock
}

// NewAnyTimesRedelegation creates a mock Redelegation that expects calls
// to the AppAddress method any number of times. When the method is called, it
// returns the provided app address.
func NewAnyTimesRedelegation(
	t *testing.T,
	appAddress string,
	gatewayAddress string,
) *mockclient.MockRedelegation {
	t.Helper()
	ctrl := gomock.NewController(t)

	// Create a mock redelegation that returns the provided address AnyTimes.
	redelegation := mockclient.NewMockRedelegation(ctrl)
	redelegation.EXPECT().GetAppAddress().Return(appAddress).AnyTimes()
	redelegation.EXPECT().GetGatewayAddress().Return(gatewayAddress).AnyTimes()

	return redelegation
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
	jsonTemplate := `{"tx":"SGVsbG8sIHdvcmxkIQ==","result":{"events":[{"type":"message","attributes":[{"key":"action","value":"/poktroll.application.MsgDelegateToGateway"},{"key":"sender","value":"pokt1exampleaddress"},{"key":"module","value":"application"}]},{"type":"poktroll.application.EventRedelegation","attributes":[{"key":"app_address","value":"\"%s\""},{"key":"gateway_address","value":"\"%s\""}]}]}}`

	txResultEvent := &tx.CometTxEvent{}

	err := json.Unmarshal(
		[]byte(fmt.Sprintf(jsonTemplate, appAddress, gatewayAddress)),
		&txResultEvent.Data.Value.TxResult,
	)
	require.NoError(t, err)

	txResultBz, err := json.Marshal(txResultEvent)
	require.NoError(t, err)

	rpcResult := &rpctypes.RPCResponse{Result: txResultBz}
	rpcResultBz, err := json.Marshal(rpcResult)
	require.NoError(t, err)

	return rpcResultBz
}
