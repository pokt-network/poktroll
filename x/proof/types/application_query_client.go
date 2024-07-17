package types

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/proto/types/proof"
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
) (application.Application, error) {
	foundApp, appFound := appQueryClient.keeper.GetApplication(ctx, appAddr)
	if !appFound {
		return application.Application{}, proof.ErrProofApplicationNotFound
	}

	return foundApp, nil
}

// GetAllApplications returns all the applications in the application store.
func (appQueryClient *AppKeeperQueryClient) GetAllApplications(ctx context.Context) ([]application.Application, error) {
	return appQueryClient.keeper.GetAllApplications(ctx), nil
}
