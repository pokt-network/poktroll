package query

import (
	"context"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.ServiceQueryClient = (*serviceQuerier)(nil)

// serviceQuerier is a wrapper around the servicetypes.QueryClient that enables the
// querying of onchain service information through a single exposed method
// which returns a sharedtypes.Service struct
type serviceQuerier struct {
	clientConn     grpc.ClientConn
	serviceQuerier servicetypes.QueryClient

	blockClient                client.BlockClient
	serviceCache               map[string]*sharedtypes.Service
	relayMiningDifficultyCache map[string]servicetypes.RelayMiningDifficulty
	serviceCacheMu             sync.Mutex
}

// NewServiceQuerier returns a new instance of a client.ServiceQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
func NewServiceQuerier(ctx context.Context, deps depinject.Config) (client.ServiceQueryClient, error) {
	servq := &serviceQuerier{}

	if err := depinject.Inject(
		deps,
		&servq.blockClient,
		&servq.clientConn,
	); err != nil {
		return nil, err
	}

	servq.serviceQuerier = servicetypes.NewQueryClient(servq.clientConn)

	channel.ForEach(
		ctx,
		servq.blockClient.CommittedBlocksSequence(ctx),
		func(ctx context.Context, block client.Block) {
			servq.serviceCacheMu.Lock()
			defer servq.serviceCacheMu.Unlock()

			servq.serviceCache = make(map[string]*sharedtypes.Service)
			servq.relayMiningDifficultyCache = make(map[string]servicetypes.RelayMiningDifficulty)
		},
	)

	return servq, nil
}

// GetService returns a sharedtypes.Service struct for a given serviceId.
// It implements the ServiceQueryClient#GetService function.
func (servq *serviceQuerier) GetService(
	ctx context.Context,
	serviceId string,
) (sharedtypes.Service, error) {
	servq.serviceCacheMu.Lock()
	defer servq.serviceCacheMu.Unlock()

	if foundService, isServiceFound := servq.serviceCache[serviceId]; isServiceFound {
		return *foundService, nil
	}

	req := &servicetypes.QueryGetServiceRequest{
		Id: serviceId,
	}

	res, err := servq.serviceQuerier.Service(ctx, req)
	if err != nil {
		return sharedtypes.Service{}, ErrQueryRetrieveService.Wrapf(
			"serviceId: %s; error: [%v]",
			serviceId, err,
		)
	}

	servq.serviceCache[serviceId] = &res.Service
	return res.Service, nil
}

// GetServiceRelayDifficulty queries the onchain data for
// the relay mining difficulty associated with the given service.
func (servq *serviceQuerier) GetServiceRelayDifficulty(
	ctx context.Context,
	serviceId string,
) (servicetypes.RelayMiningDifficulty, error) {
	servq.serviceCacheMu.Lock()
	defer servq.serviceCacheMu.Unlock()

	if foundRelayMiningDifficulty, isRelayMiningDifficultyFound := servq.relayMiningDifficultyCache[serviceId]; isRelayMiningDifficultyFound {
		return foundRelayMiningDifficulty, nil
	}
	req := &servicetypes.QueryGetRelayMiningDifficultyRequest{
		ServiceId: serviceId,
	}

	res, err := servq.serviceQuerier.RelayMiningDifficulty(ctx, req)
	if err != nil {
		return servicetypes.RelayMiningDifficulty{}, err
	}

	servq.relayMiningDifficultyCache[serviceId] = res.RelayMiningDifficulty
	return res.RelayMiningDifficulty, nil
}
