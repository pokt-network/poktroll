package types

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

var _ client.ApplicationQueryClient = (*AppKeeperQueryClient)(nil)

// AppKeeperQueryClient is a thin wrapper around the AccountKeeper.
// It does not rely on the QueryClient, and therefore does not make any
// network requests as in the off-chain implementation.
type AppKeeperQueryClient struct {
	keeper ApplicationKeeper
}

// NewAppKeeperQueryClient returns a new ApplicationQueryClient that is backed
// by an ApplicationKeeper instance.
// It is used by the RingClient to get the gateway address that an application
// has delegated its signing power to.
// It should be injected into the RingClient when initialized from within the a keeper.
func NewAppKeeperQueryClient(appKeeper ApplicationKeeper) client.ApplicationQueryClient {
	return &AppKeeperQueryClient{keeper: appKeeper}
}

// GetApplication returns the application corresponding to the given address.
func (appQueryClient *AppKeeperQueryClient) GetApplication(
	ctx context.Context,
	appAddr string,
) (apptypes.Application, error) {
	foundApp, appFound := appQueryClient.keeper.GetApplication(ctx, appAddr)
	if !appFound {
		return apptypes.Application{}, ErrProofApplicationNotFound
	}

	return foundApp, nil
}

// GetAllApplications returns all the applications in the application store.
func (appQueryClient *AppKeeperQueryClient) GetAllApplications(ctx context.Context) ([]apptypes.Application, error) {
	return appQueryClient.keeper.GetAllApplications(ctx), nil
}

// GetParams returns the application module parameters.
func (appQueryClient *AppKeeperQueryClient) GetParams(ctx context.Context) (*apptypes.Params, error) {
	return appQueryClient.GetParams(ctx)
}
