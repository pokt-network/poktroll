package testrelayer

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/mockrelayer"
)

// TODO_IN_THIS_COMMIT: comment ...
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
