package testqueryclients

import (
	"context"
	"testing"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/x/application/types"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// appToGatewayMap is a map of:
//
//	applicationAddresses -> list of the app's delegated gateway addresses.
//
// All addresses are assumed to exist on chain.
var appToGatewayMap map[string][]string

func init() {
	appToGatewayMap = make(map[string][]string)
}

// NewTestApplicationQueryClient creates a mock of the ApplicationQueryClient
// which allows the caller to call GetApplication any times and will return
// an application with the given address.
func NewTestApplicationQueryClient(
	t *testing.T,
) *mockclient.MockApplicationQueryClient {
	ctrl := gomock.NewController(t)

	applicationQuerier := mockclient.NewMockApplicationQueryClient(ctrl)
	applicationQuerier.EXPECT().GetApplication(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			_ context.Context,
			appAddress string,
		) (application types.Application, err error) {
			delegateeAddresses, ok := appToGatewayMap[appAddress]
			if !ok {
				return types.Application{}, apptypes.ErrAppNotFound
			}
			return apptypes.Application{
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
				DelegateeGatewayAddresses: delegateeAddresses,
			}, nil
		}).
		AnyTimes()

	return applicationQuerier
}

// AddAddressToApplicationMap adds the given address to the addressApplicationMap
// with the given delegated gateways addresses. It also adds it to the
// addressAccountMap so that the account will be deemed to exist on chain.
func AddAddressToApplicationMap(
	t *testing.T,
	address string, pubkey cryptotypes.PubKey,
	delegateeAccounts map[string]cryptotypes.PubKey,
) {
	t.Helper()
	addAddressToAccountMap(t, address, pubkey)
	delegateeAddresses := make([]string, 0)
	for delegateeAddress, delegateePubKey := range delegateeAccounts {
		delegateeAddresses = append(delegateeAddresses, delegateeAddress)
		addAddressToAccountMap(t, delegateeAddress, delegateePubKey)
	}
	appToGatewayMap[address] = delegateeAddresses
	t.Cleanup(func() {
		delete(appToGatewayMap, address)
	})
}
