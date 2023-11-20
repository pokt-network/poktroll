package query

import (
	"context"

	"cosmossdk.io/depinject"
	cosmosclient "github.com/cosmos/cosmos-sdk/client"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/query/types"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

type appQuerier struct {
	clientCtx          types.Context
	applicationQuerier apptypes.QueryClient
}

func NewApplicationQuerier(
	deps depinject.Config,
) (client.ApplicationQueryClient, error) {
	aq := &appQuerier{}

	if err := depinject.Inject(
		deps,
		&aq.clientCtx,
	); err != nil {
		return nil, err
	}

	aq.applicationQuerier = apptypes.NewQueryClient(cosmosclient.Context(aq.clientCtx))

	return aq, nil
}

func (aq *appQuerier) GetApplication(
	ctx context.Context,
	appAddress string,
) (apptypes.Application, error) {
	req := apptypes.QueryGetApplicationRequest{Address: appAddress}
	res, err := aq.applicationQuerier.Application(ctx, &req)
	if err != nil {
		return apptypes.Application{}, ErrQueryAccountNotFound.Wrapf("app address: %s [%v]", appAddress, err)
	}
	return res.Application, nil
}
