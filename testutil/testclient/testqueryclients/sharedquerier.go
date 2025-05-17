package testqueryclients

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var sharedParamsHistory = sharedtypes.InitialParamsHistory(sharedtypes.DefaultParams())

// NewTestSharedQueryClient creates a mock of the SharedQueryClient which uses the
// default shared module params for its implementation.
func NewTestSharedQueryClient(
	t *testing.T,
) *mockclient.MockSharedQueryClient {
	ctrl := gomock.NewController(t)

	sharedQuerier := mockclient.NewMockSharedQueryClient(ctrl)

	params := sharedParamsHistory.GetCurrentParams()
	sharedQuerier.EXPECT().
		GetParams(gomock.Any()).
		Return(&params, nil).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetClaimWindowOpenHeight(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, queryHeight int64) (int64, error) {
				return sharedParamsHistory.GetClaimWindowOpenHeight(queryHeight), nil
			},
		).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetProofWindowOpenHeight(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, queryHeight int64) (int64, error) {
				return sharedParamsHistory.GetProofWindowOpenHeight(queryHeight), nil
			},
		).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetSessionGracePeriodEndHeight(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, queryHeight int64) (int64, error) {
				return sharedParamsHistory.GetSessionGracePeriodEndHeight(queryHeight), nil
			},
		).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetEarliestSupplierClaimCommitHeight(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(
				ctx context.Context,
				sessionEndHeight int64,
				supplierOperatorAddr string,
			) (int64, error) {
				return sharedtypes.GetEarliestSupplierClaimCommitHeight(
					sharedParamsHistory,
					sessionEndHeight,
					[]byte{},
					supplierOperatorAddr,
				), nil
			},
		).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetEarliestSupplierProofCommitHeight(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(
				ctx context.Context,
				sessionEndHeight int64,
				supplierOperatorAddr string,
			) (int64, error) {
				return sharedtypes.GetEarliestSupplierClaimCommitHeight(
					sharedParamsHistory,
					sessionEndHeight,
					[]byte{},
					supplierOperatorAddr,
				), nil
			},
		).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetParamsAtHeight(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, height int64) (*sharedtypes.Params, error) {
			sharedParams := sharedParamsHistory.GetParamsAtHeight(height)
			return &sharedParams, nil
		}).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetParamsUpdates(gomock.Any()).
		Return(sharedParamsHistory, nil).
		AnyTimes()

	return sharedQuerier
}
