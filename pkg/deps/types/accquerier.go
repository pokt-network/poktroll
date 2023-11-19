package types

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/client"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/pkg/relayer"
)

type AccountQuerier interface {
	GetAccount(ctx context.Context, address string) (accounttypes.AccountI, error)
}

type accQuerier struct {
	clientCtx      relayer.QueryClientContext
	accountQuerier accounttypes.QueryClient
}

func NewAccountQuerier(
	deps depinject.Config,
) (AccountQuerier, error) {
	aq := &accQuerier{}

	if err := depinject.Inject(
		deps,
		&aq.clientCtx,
	); err != nil {
		return nil, err
	}

	aq.accountQuerier = accounttypes.NewQueryClient(client.Context(aq.clientCtx))

	return aq, nil
}

func (aq *accQuerier) GetAccount(
	ctx context.Context,
	address string,
) (accounttypes.AccountI, error) {
	req := &accounttypes.QueryAccountRequest{Address: address}
	res, err := aq.accountQuerier.Account(ctx, req)
	if err != nil {
		return nil, ErrDepsAccountNotFound.Wrapf("address: %s [%v]", address, err)
	}
	var acc accounttypes.AccountI
	if err = depCodec.UnpackAny(res.Account, &acc); err != nil {
		return nil, ErrDepsUnableToDeserialiseAccount.Wrapf("address: %s [%v]", address, err)
	}
	return acc, nil
}
