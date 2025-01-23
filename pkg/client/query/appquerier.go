package query

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	grpc "github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

var _ client.ApplicationQueryClient = (*appQuerier)(nil)

// appQuerier is a wrapper around the apptypes.QueryClient that enables the
// querying of onchain application information through a single exposed method
// which returns an apptypes.Application interface
type appQuerier struct {
	clientConn         grpc.ClientConn
	applicationQuerier apptypes.QueryClient

	blockClient    client.BlockClient
	appCache       map[string]*apptypes.Application
	appParamsCache *apptypes.Params
	appCacheMu     sync.Mutex
}

// NewApplicationQuerier returns a new instance of a client.ApplicationQueryClient
// by injecting the dependecies provided by the depinject.Config
//
// Required dependencies:
// - clientCtx
func NewApplicationQuerier(ctx context.Context, deps depinject.Config) (client.ApplicationQueryClient, error) {
	aq := &appQuerier{}

	if err := depinject.Inject(
		deps,
		&aq.blockClient,
		&aq.clientConn,
	); err != nil {
		return nil, err
	}

	aq.appCache = make(map[string]*apptypes.Application)

	aq.applicationQuerier = apptypes.NewQueryClient(aq.clientConn)

	channel.ForEach(
		ctx,
		aq.blockClient.CommittedBlocksSequence(ctx),
		func(ctx context.Context, block client.Block) {
			aq.appCacheMu.Lock()
			defer aq.appCacheMu.Unlock()

			aq.appCache = make(map[string]*apptypes.Application)
			aq.appParamsCache = nil
		},
	)

	return aq, nil
}

// GetApplication returns an apptypes.Application interface for a given address
func (aq *appQuerier) GetApplication(
	ctx context.Context,
	appAddress string,
) (apptypes.Application, error) {
	aq.appCacheMu.Lock()
	defer aq.appCacheMu.Unlock()

	if foundApp, isAppFound := aq.appCache[appAddress]; isAppFound {
		return *foundApp, nil
	}

	req := apptypes.QueryGetApplicationRequest{Address: appAddress}
	res, err := aq.applicationQuerier.Application(ctx, &req)
	if err != nil {
		return apptypes.Application{}, apptypes.ErrAppNotFound.Wrapf("app address: %s [%v]", appAddress, err)
	}
	aq.appCache[appAddress] = &res.Application
	return res.Application, nil
}

// GetAllApplications returns all staked applications
func (aq *appQuerier) GetAllApplications(ctx context.Context) ([]apptypes.Application, error) {
	req := apptypes.QueryAllApplicationsRequest{}
	res, err := aq.applicationQuerier.AllApplications(ctx, &req)
	if err != nil {
		return []apptypes.Application{}, err
	}
	return res.Applications, nil
}

// GetParams returns the application module parameters
func (aq *appQuerier) GetParams(ctx context.Context) (*apptypes.Params, error) {
	aq.appCacheMu.Lock()
	defer aq.appCacheMu.Unlock()

	if aq.appParamsCache != nil {
		return aq.appParamsCache, nil
	}

	req := apptypes.QueryParamsRequest{}
	res, err := aq.applicationQuerier.Params(ctx, &req)
	if err != nil {
		return nil, err
	}
	aq.appParamsCache = &res.Params
	return &res.Params, nil
}
