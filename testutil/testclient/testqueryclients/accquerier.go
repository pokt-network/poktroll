package testqueryclients

import (
	"context"
	"testing"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	"github.com/pokt-network/poktroll/testutil/sample"
)

// addressAccountMap is a map of addresses that are deemed to exist on chain
// if an address is not in this map a public key will not be included in the
// response from the mock AccountQueryClient's GetAccount method.
var addressAccountMap map[string]struct{}

func init() {
	addressAccountMap = make(map[string]struct{})
}

// NewTestAccountQueryClient creates a mock of the AccountQueryClient
// which allows the caller to call GetApplication any times and will return
// an application with the given address.
// The public key in the account it returns is a randomly generated secp256k1
// public key, not related to the address provided.
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
			// Generate a random public key if the account "exists"
			if _, ok := addressAccountMap[address]; ok {
				_, pk := sample.AccAddressAndPubKey()
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
func addAddressToAccountMap(t *testing.T, address string) {
	t.Helper()
	addressAccountMap[address] = struct{}{}
	t.Cleanup(func() {
		delete(addressAccountMap, address)
	})
}
