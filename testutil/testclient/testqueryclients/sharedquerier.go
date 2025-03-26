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
				sharedParamsUpdates := []*sharedtypes.ParamsUpdate{
					{
						Params:               sharedtypes.DefaultParams(),
						EffectiveBlockHeight: 1,
					},
				}
				return sharedtypes.GetClaimWindowOpenHeight(sharedParamsUpdates, queryHeight), nil
			},
		).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetProofWindowOpenHeight(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, queryHeight int64) (int64, error) {
				sharedParamsUpdates := []*sharedtypes.ParamsUpdate{
					{
						Params:               sharedtypes.DefaultParams(),
						EffectiveBlockHeight: 1,
					},
				}
				return sharedtypes.GetProofWindowOpenHeight(sharedParamsUpdates, queryHeight), nil
			},
		).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetSessionGracePeriodEndHeight(gomock.Any(), gomock.Any()).
		DoAndReturn(
			func(ctx context.Context, queryHeight int64) (int64, error) {
				sharedParamsUpdates := []*sharedtypes.ParamsUpdate{
					{
						Params:               sharedtypes.DefaultParams(),
						EffectiveBlockHeight: 1,
					},
				}
				return sharedtypes.GetSessionGracePeriodEndHeight(sharedParamsUpdates, queryHeight), nil
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
				sharedParamsUpdates := []*sharedtypes.ParamsUpdate{
					{
						Params:               sharedtypes.DefaultParams(),
						EffectiveBlockHeight: 1,
					},
				}
				return sharedtypes.GetEarliestSupplierClaimCommitHeight(
					sharedParamsUpdates,
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
				sharedParamsUpdates := []*sharedtypes.ParamsUpdate{
					{
						Params:               sharedtypes.DefaultParams(),
						EffectiveBlockHeight: 1,
					},
				}
				return sharedtypes.GetEarliestSupplierProofCommitHeight(
					sharedParamsUpdates,
					sessionEndHeight,
					[]byte{},
					supplierOperatorAddr,
				), nil
			},
		).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetParamsAtHeight(gomock.Any(), gomock.Any()).
		Return(&params, nil).
		AnyTimes()

	sharedQuerier.EXPECT().
		GetParamsUpdates(gomock.Any()).
		DoAndReturn(func(ctx context.Context) ([]*sharedtypes.ParamsUpdate, error) {
			sharedParamsUpdates := []*sharedtypes.ParamsUpdate{
				{
					Params:               sharedtypes.DefaultParams(),
					EffectiveBlockHeight: 1,
				},
			}
			return sharedParamsUpdates, nil
		}).
		AnyTimes()

	return sharedQuerier
}
