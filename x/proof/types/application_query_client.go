package types

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

var (
	_ client.ApplicationQueryClient = (*AppKeeperQueryClient)(nil)
)

type AppKeeperQueryClient struct {
	keeper ApplicationKeeper
}

func NewAppKeeperQueryClient(appKeeper ApplicationKeeper) client.ApplicationQueryClient {
	return &AppKeeperQueryClient{keeper: appKeeper}
}

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
