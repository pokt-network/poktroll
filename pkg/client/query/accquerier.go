package query

import (
	"context"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/pkg/client"
)

type accQuerier struct {
	clientCtx      client.QueryClientContext
	accountQuerier accounttypes.QueryClient
}

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
