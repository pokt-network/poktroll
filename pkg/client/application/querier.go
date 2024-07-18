package application

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	apptypes "github.com/pokt-network/poktroll/proto/types/application"
)

var _ client.ApplicationQueryClient = (*appQueryClient)(nil)

// appQueryClient is a wrapper around the apptypes.QueryClient that enables the
// querying of on-chain application information through a single exposed method
// which returns an apptypes.Application interface
type appQueryClient struct {
	clientConn         grpc.ClientConn
	applicationQuerier apptypes.QueryClient
}

// NewApplicationQueryClient returns a new instance of a client.ApplicationQueryClient
// by injecting the dependecies provided by the depinject.Config
//
// Required dependencies:
// - clientCtx
func NewApplicationQueryClient(deps depinject.Config) (client.ApplicationQueryClient, error) {
	aq := &appQueryClient{}

	if err := depinject.Inject(
		deps,
		&aq.clientConn,
	); err != nil {
		return nil, err
	}

	aq.applicationQuerier = apptypes.NewQueryClient(aq.clientConn)

	return aq, nil
}

// GetApplication returns an apptypes.Application interface for a given address
func (aq *appQueryClient) GetApplication(
	ctx context.Context,
	appAddress string,
) (apptypes.Application, error) {
	req := apptypes.QueryGetApplicationRequest{Address: appAddress}
	res, err := aq.applicationQuerier.Application(ctx, &req)
	if err != nil {
		return apptypes.Application{}, apptypes.ErrAppNotFound.Wrapf("app address: %s [%v]", appAddress, err)
	}
	return res.Application, nil
}

// GetAllApplications returns all staked applications
func (aq *appQueryClient) GetAllApplications(ctx context.Context) ([]apptypes.Application, error) {
	req := apptypes.QueryAllApplicationsRequest{}
	res, err := aq.applicationQuerier.AllApplications(ctx, &req)
	if err != nil {
		return []apptypes.Application{}, err
	}
	return res.Applications, nil
}
