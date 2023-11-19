package types

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/cosmos-sdk/client"

	"github.com/pokt-network/poktroll/pkg/relayer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

type ApplicationQuerier interface {
	GetApplication(ctx context.Context, appAddress string) (apptypes.Application, error)
}

type appQuerier struct {
	clientCtx          relayer.QueryClientContext
	applicationQuerier apptypes.QueryClient
}

func NewApplicationQuerier(
	deps depinject.Config,
) (ApplicationQuerier, error) {
	aq := &appQuerier{}

	if err := depinject.Inject(
		deps,
		&aq.clientCtx,
	); err != nil {
		return nil, err
	}

	aq.applicationQuerier = apptypes.NewQueryClient(client.Context(aq.clientCtx))

	return aq, nil
}

func (aq *appQuerier) GetApplication(
	ctx context.Context,
	appAddress string,
) (apptypes.Application, error) {
	req := apptypes.QueryGetApplicationRequest{Address: appAddress}
	res, err := aq.applicationQuerier.Application(ctx, &req)
	if err != nil {
		return apptypes.Application{}, ErrDepsAccountNotFound.Wrapf("app address: %s [%v]", appAddress, err)
	}
	return res.Application, nil
}
