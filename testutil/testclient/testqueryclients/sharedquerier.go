package testqueryclients

import (
	"context"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
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
				return sharedtypes.GetClaimWindowOpenHeight(&sharedParams, queryHeight), nil
			},
		).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetProofWindowOpenHeight(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, queryHeight int64) (int64, error) {
				sharedParams := sharedtypes.DefaultParams()
				return sharedtypes.GetProofWindowOpenHeight(&sharedParams, queryHeight), nil
			},
		).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetSessionGracePeriodEndHeight(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, queryHeight int64) (int64, error) {
				sharedParams := sharedtypes.DefaultParams()
				return sharedtypes.GetSessionGracePeriodEndHeight(&sharedParams, queryHeight), nil
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
				sharedParams := sharedtypes.DefaultParams()
				return sharedtypes.GetEarliestSupplierClaimCommitHeight(
					&sharedParams,
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
				sharedParams := sharedtypes.DefaultParams()
				return sharedtypes.GetEarliestSupplierProofCommitHeight(
					&sharedParams,
					sessionEndHeight,
					[]byte{},
					supplierOperatorAddr,
				), nil
			},
		).
		AnyTimes()

	return sharedQuerier
}
