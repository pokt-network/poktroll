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

// addressApplicationMap is a map of application addresses that are deemed to
// exist on chain, if an address is not in this map an error will be returned
// from the mock ApplicationQueryClient's GetApplication method.
// The integer value is the number of delegated gateways the application is
// delegated to, these are randomly generated addresses.
var addressApplicationMap map[string]int

func init() {
	addressApplicationMap = make(map[string]int)
}

// NewTestApplicationQueryClient creates a mock of the ApplicationQueryClient
// which allows the caller to call GetApplication any times and will return
// an application with the given address.
// The delegateeNumber parameter is used to determine how many delegated
// gateways any application returned from the GetApplication method will have.
func NewTestApplicationQueryClient(
	t *testing.T,
) *mockclient.MockApplicationQueryClient {
	ctrl := gomock.NewController(t)

	applicationQuerier := mockclient.NewMockApplicationQueryClient(ctrl)
	applicationQuerier.EXPECT().GetApplication(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			appAddress string,
		) (application *types.Application, err error) {
			delegateeNumber, ok := addressApplicationMap[appAddress]
			if !ok {
				return nil, apptypes.ErrAppNotFound
			}
			delegateeGatewayAddresses := make([]string, 0)
			for i := 0; i < delegateeNumber; i++ {
				gatewayAddress := sample.AccAddress()
				delegateeGatewayAddresses = append(delegateeGatewayAddresses, gatewayAddress)
				addAddressToAccountMap(t, gatewayAddress)
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

// AddAddressToApplicationMap adds the given address to the addressApplicationMap
// with the given number of delegated gateways. It also adds it to the
// addressAccountMap so that the account will be deemed to exist on chain.
func AddAddressToApplicationMap(
	t *testing.T,
	address string,
	delegateeNumber int,
) {
	addressApplicationMap[address] = delegateeNumber
	addAddressToAccountMap(t, address)
	t.Cleanup(func() {
		delete(addressApplicationMap, address)
	})
}
