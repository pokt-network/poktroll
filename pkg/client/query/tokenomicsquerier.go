package query

import (
	"context"

	"cosmossdk.io/depinject"
	"github.com/cosmos/gogoproto/grpc"

	"github.com/pokt-network/poktroll/pkg/client"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// tokenomicsQuerier is a wrapper around the tokenomicstypes.QueryClient that enables the
// querying of on-chain tokenomics module data.
type tokenomicsQuerier struct {
	clientConn        grpc.ClientConn
	tokenomicsQuerier tokenomicstypes.QueryClient
}

// NewTokenomicsQuerier returns a new instance of a client.TokenomicsQueryClient by
// injecting the dependecies provided by the depinject.Config.
//
// Required dependencies:
// - grpc.ClientConn
func NewTokenomicsQuerier(deps depinject.Config) (client.TokenomicsQueryClient, error) {
	querier := &tokenomicsQuerier{}

	if err := depinject.Inject(
		deps,
		&querier.clientConn,
	); err != nil {
		return nil, err
	}

	querier.tokenomicsQuerier = tokenomicstypes.NewQueryClient(querier.clientConn)

	return querier, nil
}

// GetServiceRelayDifficultyTargetHash queries the onchain data for
// the relay mining difficulty associated with the given service.
func (tq *tokenomicsQuerier) GetServiceRelayDifficultyTargetHash(
	ctx context.Context,
	serviceId string,
) (client.TokenomicsRelayMiningDifficulty, error) {
	req := &tokenomicstypes.QueryGetRelayMiningDifficultyRequest{
		ServiceId: serviceId,
	}

	res, err := tq.tokenomicsQuerier.RelayMiningDifficulty(ctx, req)
	if err != nil {
		return nil, err
	}
	return &res.RelayMiningDifficulty, nil
}
