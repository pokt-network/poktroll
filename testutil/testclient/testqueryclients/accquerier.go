package testqueryclients

import (
	"context"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
)

// addressAccountMap is a map of addresses that are deemed to exist on chain
// if an address is not in this map a public key will not be included in the
// response from the mock AccountQueryClient's GetAccount method.
var addressAccountMap map[string]cryptotypes.PubKey

func init() {
	addressAccountMap = make(map[string]cryptotypes.PubKey)
}

// NewTestAccountQueryClient creates a mock of the AccountQueryClient
// which allows the caller to call GetApplication any times and will return
// an application with the given address.
func NewTestAccountQueryClient(
	t *testing.T,
) *mockclient.MockAccountQueryClient {
	ctrl := gomock.NewController(t)

	accoutQuerier := mockclient.NewMockAccountQueryClient(ctrl)
	accoutQuerier.EXPECT().GetAccount(gomock.Any(), gomock.Any()).
		DoAndReturn(func(
			ctx context.Context,
			address string,
		) (account accounttypes.AccountI, err error) {
			anyPk := (*codectypes.Any)(nil)
			if pk, ok := addressAccountMap[address]; ok {
				anyPk, err = codectypes.NewAnyWithValue(pk)
				if err != nil {
					return nil, err
				}
			}
			return &accounttypes.BaseAccount{
				Address: address,
				PubKey:  anyPk,
			}, nil
		}).
		AnyTimes()

	return accoutQuerier
}

// addAddressToAccountMap adds the given address to the addressAccountMap
// to mock it "existing" on chain, it will also remove the address from the
// map when the test is cleaned up.
func addAddressToAccountMap(t *testing.T, address string, pubkey cryptotypes.PubKey) {
	t.Helper()
	addressAccountMap[address] = pubkey
	t.Cleanup(func() {
		delete(addressAccountMap, address)
	})
}
