package query

import (
	"context"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query/types"
)

// accQuerier is a wrapper around the accounttypes.QueryClient that enables the
// querying of on-chain account information through a single exposed method
// which returns an accounttypes.AccountI interface
type accQuerier struct {
	clientCtx      types.Context
	accountQuerier accounttypes.QueryClient
}

// NewAccountQuerier returns a new instance of a client.AccountQueryClient by
// injecting the dependecies provided by the depinject.Config
func NewAccountQuerier(
	deps depinject.Config,
) (client.AccountQueryClient, error) {
	aq := &accQuerier{}

	if err := depinject.Inject(
		deps,
		&aq.clientCtx,
	); err != nil {
		return nil, err
	}

	aq.accountQuerier = accounttypes.NewQueryClient(cosmosclient.Context(aq.clientCtx))

	return aq, nil
}

// GetAccount returns an accounttypes.AccountI interface for a given address
func (aq *accQuerier) GetAccount(
	ctx context.Context,
	address string,
) (accounttypes.AccountI, error) {
	req := &accounttypes.QueryAccountRequest{Address: address}
	res, err := aq.accountQuerier.Account(ctx, req)
	if err != nil {
		return nil, ErrQueryAccountNotFound.Wrapf("address: %s [%v]", address, err)
	}
	var acc accounttypes.AccountI
	if err = queryCodec.UnpackAny(res.Account, &acc); err != nil {
		return nil, ErrQueryUnableToDeserialiseAccount.Wrapf("address: %s [%v]", address, err)
	}
	return acc, nil
}
