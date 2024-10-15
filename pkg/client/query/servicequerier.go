package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

var _ client.ServiceQueryClient = (*serviceQuerier)(nil)

// serviceQuerier is a wrapper around the servicetypes.QueryClient that enables the
// querying of on-chain service information through a single exposed method
// which returns a sharedtypes.Service struct
type serviceQuerier struct {
	clientConn     grpc.ClientConn
	serviceQuerier servicetypes.QueryClient
}

// NewServiceQuerier returns a new instance of a client.ServiceQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - clientCtx (grpc.ClientConn)
func NewServiceQuerier(deps depinject.Config) (client.ServiceQueryClient, error) {
	servq := &serviceQuerier{}

	if err := depinject.Inject(
		deps,
		&servq.clientConn,
	); err != nil {
		return nil, err
	}

	servq.serviceQuerier = servicetypes.NewQueryClient(servq.clientConn)

	return servq, nil
}

// GetService returns a sharedtypes.Service struct for a given serviceId.
// It implements the ServiceQueryClient#GetService function.
func (servq *serviceQuerier) GetService(
	ctx context.Context,
	serviceId string,
) (sharedtypes.Service, error) {
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
	return res.Service, nil
}

// GetServiceRelayDifficulty queries the onchain data for
// the relay mining difficulty associated with the given service.
func (servq *serviceQuerier) GetServiceRelayDifficulty(
	ctx context.Context,
	serviceId string,
) (servicetypes.RelayMiningDifficulty, error) {
	req := &servicetypes.QueryGetRelayMiningDifficultyRequest{
		ServiceId: serviceId,
	}

	res, err := servq.serviceQuerier.RelayMiningDifficulty(ctx, req)
	if status.Code(err) == codes.NotFound {
		newServiceDifficulty := servicetypes.RelayMiningDifficulty{
			ServiceId:    serviceId,
			BlockHeight:  0,
			NumRelaysEma: 0,
			TargetHash:   protocol.BaseRelayDifficultyHashBz,
		}

		return newServiceDifficulty, nil
	}
	if err != nil {
		return servicetypes.RelayMiningDifficulty{}, err
	}

	return res.RelayMiningDifficulty, nil
}
