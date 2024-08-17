package testrelayer

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/mockrelayer"
)

// NewMockOneTimeRelayerProxy creates a new mock RelayerProxy. This mock
// RelayerProxy will expect a call to ServedRelays with the given context, and
// when that call is made, returnedRelaysObs is returned. It also expects a call
// to Start and Stop with the given context.
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

// NewMockOneTimeRelayerProxyWithPing creates a new mock RelayerProxy. This mock
// RelayerProxy will expect a call to ServedRelays with the given context, and
// when that call is made, returnedRelaysObs is returned. It also expects a call
// to Start, Ping, and Stop with the given context.
func NewMockOneTimeRelayerProxyWithPing(
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
	relayerProxyMock.EXPECT().
		Ping(gomock.Eq(ctx)).
		Times(1)

	return relayerProxyMock
}
