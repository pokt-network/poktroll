package keeper

import (
	"testing"

	"cosmossdk.io/log"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/application/mocks"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// Recording an unchanged per-session spend limit does not read the shared params.
func TestRecordPerSessionSpendLimitChange_UnchangedLimitSkipsParamsRead(t *testing.T) {
	limit := sdk.NewInt64Coin("upokt", 100)
	sameLimit := sdk.NewInt64Coin("upokt", 100)
	newLimit := sdk.NewInt64Coin("upokt", 200)

	tests := []struct {
		desc           string
		prevLimit      *sdk.Coin
		limit          *sdk.Coin
		expectedReads  int
		expectedLength int
	}{
		{desc: "both nil", prevLimit: nil, limit: nil, expectedReads: 0, expectedLength: 0},
		{desc: "same pointer", prevLimit: &limit, limit: &limit, expectedReads: 0, expectedLength: 0},
		{desc: "equal value", prevLimit: &limit, limit: &sameLimit, expectedReads: 0, expectedLength: 0},
		{desc: "changed", prevLimit: &limit, limit: &newLimit, expectedReads: 1, expectedLength: 2},
	}

	for _, test := range tests {
		t.Run(test.desc, func(t *testing.T) {
			ctrl := gomock.NewController(t)
			sharedKeeper := mocks.NewMockSharedKeeper(ctrl)
			sharedKeeper.EXPECT().GetParams(gomock.Any()).
				Return(sharedtypes.DefaultParams()).
				Times(test.expectedReads)

			k := Keeper{sharedKeeper: sharedKeeper}
			ctx := sdk.NewContext(nil, cmtproto.Header{Height: 1}, false, log.NewNopLogger())
			app := &apptypes.Application{PerSessionSpendLimit: test.limit}

			k.recordPerSessionSpendLimitChange(ctx, app, test.prevLimit)
			require.Len(t, app.PerSessionSpendLimitHistory, test.expectedLength)
		})
	}
}
