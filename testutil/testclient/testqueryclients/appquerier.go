package testqueryclients

import (
	"context"
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/application/types"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// NewTestApplicationQueryClient creates a mock of the ApplicationQueryClient
// which allows the caller to call GetApplication any times and will return
// an application with the given address.
// The delegateeNumber parameter is used to determine how many delegated
// gateways any application returned from the GetApplication method will have.
func NewTestApplicationQueryClient(
	t *testing.T,
	ctx context.Context,
	delegateeNumber int,
) *mockclient.MockApplicationQueryClient {
	ctrl := gomock.NewController(t)

	applicationQuerier := mockclient.NewMockApplicationQueryClient(ctrl)
	applicationQuerier.EXPECT().GetApplication(gomock.Eq(ctx), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			appAddress string,
		) (application *types.Application, err error) {
			delegateeGatewayAddresses := make([]string, delegateeNumber)
			for i := range delegateeGatewayAddresses {
				delegateeGatewayAddresses[i] = sample.AccAddress()
			}
			return &apptypes.Application{
				Address: appAddress,
				Stake:   &sdk.Coin{Denom: "upokt", Amount: sdk.NewInt(100)},
				ServiceConfigs: []*sharedtypes.ApplicationServiceConfig{
					{
						Service: &sharedtypes.Service{
							Id:   "svc1",
							Name: "service one",
						},
					},
				},
				DelegateeGatewayAddresses: delegateeGatewayAddresses,
			}, nil
		}).
		AnyTimes()

	return applicationQuerier
}
