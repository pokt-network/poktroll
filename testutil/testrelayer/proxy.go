package testrelayer

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/mockrelayer"
)

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
