package query

import (
	"context"

	"cosmossdk.io/depinject"
	grpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/proto/types/application"
)

var _ client.ApplicationQueryClient = (*appQuerier)(nil)

// appQuerier is a wrapper around the apptypes.QueryClient that enables the
// querying of on-chain application information through a single exposed method
// which returns an apptypes.Application interface
type appQuerier struct {
	clientConn         grpc.ClientConn
	applicationQuerier application.QueryClient
}

// NewApplicationQuerier returns a new instance of a client.ApplicationQueryClient
// by injecting the dependecies provided by the depinject.Config
//
// Required dependencies:
// - clientCtx
func NewApplicationQuerier(deps depinject.Config) (client.ApplicationQueryClient, error) {
	aq := &appQuerier{}

	if err := depinject.Inject(
		deps,
		&aq.clientConn,
	); err != nil {
		return nil, err
	}

	aq.applicationQuerier = application.NewQueryClient(aq.clientConn)

	return aq, nil
}

// GetApplication returns an apptypes.Application interface for a given address
func (aq *appQuerier) GetApplication(
	ctx context.Context,
	appAddress string,
) (application.Application, error) {
	req := application.QueryGetApplicationRequest{Address: appAddress}
	res, err := aq.applicationQuerier.Application(ctx, &req)
	if err != nil {
		return application.Application{}, application.ErrAppNotFound.Wrapf("app address: %s [%v]", appAddress, err)
	}
	return res.Application, nil
}

// GetAllApplications returns all staked applications
func (aq *appQuerier) GetAllApplications(ctx context.Context) ([]application.Application, error) {
	req := application.QueryAllApplicationsRequest{}
	res, err := aq.applicationQuerier.AllApplications(ctx, &req)
	if err != nil {
		return []application.Application{}, err
	}
	return res.Applications, nil
}
