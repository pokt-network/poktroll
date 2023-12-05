package sdk

import (
	"context"
	"fmt"

	"cosmossdk.io/depinject"
	grpctypes "github.com/cosmos/gogoproto/grpc"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	block "github.com/pokt-network/poktroll/pkg/client/block"
	eventsquery "github.com/pokt-network/poktroll/pkg/client/events_query"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
)

// buildDeps builds the dependencies for the POKTRollSDK if they are not provided
// in the config. This is useful for the SDK consumers that do not want or
// cannot provide the dependencies through depinject.
func (sdk *poktrollSDK) buildDeps(
	ctx context.Context,
	config *POKTRollSDKConfig,
) (depinject.Config, error) {
	pocketNodeWebsocketURL := fmt.Sprintf("ws://%s/websocket", config.PocketNodeUrl.Host)

	// Have a new depinject config
	deps := depinject.Configs()

	// Create and supply the events query client
	eventsQueryClient := eventsquery.NewEventsQueryClient(pocketNodeWebsocketURL)
	depinject.Configs(deps, depinject.Supply(eventsQueryClient))

	// Create and supply the block client that depends on the events query client
	blockClient, err := block.NewBlockClient(ctx, deps, pocketNodeWebsocketURL)
	if err != nil {
		return nil, err
	}
	depinject.Configs(deps, depinject.Supply(blockClient))

	// Create and supply the grpc client used by the queriers
	// TODO_TECHDEBT: Configure the grpc client options from the config
	var grpcClient grpctypes.ClientConn
	grpcClient, err = grpc.Dial(
		config.PocketNodeUrl.Host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	depinject.Configs(deps, depinject.Supply(grpcClient))

	// Create and supply the account querier
	accountQuerier, err := query.NewAccountQuerier(deps)
	if err != nil {
		return nil, err
	}
	depinject.Configs(deps, depinject.Supply(accountQuerier))

	// Create and supply the application querier
	applicationQuerier, err := query.NewApplicationQuerier(deps)
	if err != nil {
		return nil, err
	}
	depinject.Configs(deps, depinject.Supply(applicationQuerier))

	// Create and supply the session querier
	sessionQuerier, err := query.NewSessionQuerier(deps)
	if err != nil {
		return nil, err
	}
	depinject.Configs(deps, depinject.Supply(sessionQuerier))

	// Create and supply the ring cache that depends on application and account queriers
	ringCache, err := rings.NewRingCache(deps)
	if err != nil {
		return nil, err
	}
	depinject.Configs(deps, depinject.Supply(ringCache))

	return deps, nil
}
