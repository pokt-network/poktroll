package types

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

var _ client.ApplicationQueryClient = (*AppKeeperQueryClient)(nil)

type AppKeeperQueryClient struct {
	keeper ApplicationKeeper
}

// NewAppKeeperQueryClient returns a new ApplicationQueryClient that is backed
// by an ApplicationKeeper instance.
// It is used by the RingClient to get the applications that are delegated to
// by a given application.
// This implementation is a thin wrapper around the ApplicationKeeper and does
// not rely on the QueryClient contrariwise to the off-chain implementation.
// It should be injected into the RingClient when initialized from within the a keeper.
func NewAppKeeperQueryClient(appKeeper ApplicationKeeper) client.ApplicationQueryClient {
	return &AppKeeperQueryClient{keeper: appKeeper}
}

// GetApplication returns the application corresponding to the given address.
func (appQueryClient *AppKeeperQueryClient) GetApplication(
	ctx context.Context,
	appAddr string,
) (apptypes.Application, error) {
	app, _ := appQueryClient.keeper.GetApplication(ctx, appAddr)
	return app, nil
}

func (appQueryClient *AppKeeperQueryClient) GetAllApplications(ctx context.Context) ([]apptypes.Application, error) {
	return appQueryClient.keeper.GetAllApplications(ctx), nil
}
