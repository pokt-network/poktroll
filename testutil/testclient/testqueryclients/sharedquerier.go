package testqueryclients

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/x/shared"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// NewTestSharedQueryClient creates a mock of the SharedQueryClient which uses the
// default shared module params for its implementation.
func NewTestSharedQueryClient(
	t *testing.T,
) *mockclient.MockSharedQueryClient {
	ctrl := gomock.NewController(t)

	sharedQuerier := mockclient.NewMockSharedQueryClient(ctrl)
	params := sharedtypes.DefaultParams()

	sharedQuerier.EXPECT().
		GetParams(gomock.Any()).
		Return(&params, nil).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetClaimWindowOpenHeight(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, queryHeight int64) (int64, error) {
				sharedParams := sharedtypes.DefaultParams()
				return shared.GetClaimWindowOpenHeight(&sharedParams, queryHeight), nil
			},
		).
		AnyTimes()

	return sharedQuerier
}
