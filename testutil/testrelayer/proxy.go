package testrelayer

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/mockrelayer"
)

// NewMockOneTimeRelayerProxy creates a new mock RelayerProxy that:
// - Expects a call to ServedRelays with the given context
// - Returns returnedRelaysObs when ServedRelays is called
// - Expects one call each to Start, Ping, and Stop with the given context
func NewMockOneTimeRelayerProxy(
	ctx context.Context,
	t *testing.T,
	returnedRelaysObs relayer.RelaysObservable,
) *mockrelayer.MockRelayerProxy {
	t.Helper()

	ctrl := gomock.NewController(t)
	relayerProxyMock := mockrelayer.NewMockRelayerProxy(ctrl)
	relayerProxyMock.EXPECT().
		Start(gomock.Eq(ctx)).
		Times(1)
	relayerProxyMock.EXPECT().
		Stop(gomock.Eq(ctx)).
		Times(1)
	relayerProxyMock.EXPECT().
		ServedRelays().
		Return(returnedRelaysObs).
		Times(1)

	return relayerProxyMock
}

// NewMockOneTimeRelayerProxyWithPing creates a new mock RelayerProxy that:
// - Expects exactly one call to PingAll
// - Expects all the other behaviour inherited from NewMockOneTimeRelayerProxy
func NewMockOneTimeRelayerProxyWithPing(
	ctx context.Context,
	t *testing.T,
	returnedRelaysObs relayer.RelaysObservable,
) *mockrelayer.MockRelayerProxy {
	relayerProxyMock := NewMockOneTimeRelayerProxy(ctx, t, returnedRelaysObs)
	relayerProxyMock.EXPECT().
		PingAll(gomock.Eq(ctx)).
		Times(1)

	return relayerProxyMock
}
