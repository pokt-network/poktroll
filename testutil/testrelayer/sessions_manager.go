package testrelayer

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/mockrelayer"
)

// NewMockOneTimeRelayerSessionsManager creates a new mock RelayerSessionsManager.
// This mock RelayerSessionsManager will expect a call to InsertRelays with the
// given context and expectedMinedRelaysObs args. When that call is made,
// returnedMinedRelaysObs is returned. It also expects a call to Start with the
// given context, and stop.
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
