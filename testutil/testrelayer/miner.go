package testrelayer

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/mockrelayer"
)

// NewMockOneTimeMiner creates a new mock Miner. This mock Miner will expect a
// call to MinedRelays with the given context and expectedRelayObs args. When
// that call is made, returnedMinedRelaysObs is returned.
func NewMockOneTimeMiner(
	ctx context.Context,
	t *testing.T,
	expectedRelaysObs relayer.RelaysObservable,
	returnedMinedRelaysObs relayer.MinedRelaysObservable,
) *mockrelayer.MockMiner {
	t.Helper()

	ctrl := gomock.NewController(t)
	minerMock := mockrelayer.NewMockMiner(ctrl)
	minerMock.EXPECT().
		MinedRelays(
			gomock.Eq(ctx),
			gomock.Eq(expectedRelaysObs),
		).
		Return(returnedMinedRelaysObs).
		Times(1)
	return minerMock
}
