package testrelayer

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/mockrelayer"
)

func NewMockOneTimeRelayerSessionsManager(
	ctx context.Context,
	t *testing.T,
	expectedMinedRelaysObs relayer.MinedRelaysObservable,
) *mockrelayer.MockRelayerSessionsManager {
	t.Helper()

	ctrl := gomock.NewController(t)
	relayerSessionsManagerMock := mockrelayer.NewMockRelayerSessionsManager(ctrl)
	relayerSessionsManagerMock.EXPECT().
		InsertRelays(gomock.Eq(expectedMinedRelaysObs)).
		Times(1)
	relayerSessionsManagerMock.EXPECT().
		Start(gomock.Eq(ctx)).
		Times(1)
	relayerSessionsManagerMock.EXPECT().
		Stop().
		Times(1)
	return relayerSessionsManagerMock
}
