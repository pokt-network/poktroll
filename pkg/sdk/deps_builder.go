package sdk

import (
	"context"

	"cosmossdk.io/depinject"
	grpctypes "github.com/cosmos/gogoproto/grpc"
	grpc "google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	block "github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/delegation"
	eventsquery "github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/query"
	pubkeyclient "github.com/pokt-network/poktroll/pkg/crypto/pubkey_client"
	"github.com/pokt-network/poktroll/pkg/crypto/rings"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// buildDeps builds the dependencies for the POKTRollSDK if they are not provided
// in the config. This is useful for the SDK consumers that do not want or
// cannot provide the dependencies through depinject.
func (sdk *poktrollSDK) buildDeps(
	ctx context.Context,
	config *POKTRollSDKConfig,
) (depinject.Config, error) {
	pocketNodeWebsocketURL := HostToWebsocketURL(config.QueryNodeUrl.Host)

	// Have a new depinject config
	deps := depinject.Configs()

	// Supply the logger
	deps = depinject.Configs(deps, depinject.Supply(polylog.Ctx(ctx)))

	// Create and supply the events query client
	eventsQueryClient := eventsquery.NewEventsQueryClient(pocketNodeWebsocketURL)
	deps = depinject.Configs(deps, depinject.Supply(eventsQueryClient))

	// Create and supply the block client that depends on the events query client
	blockClient, err := block.NewBlockClient(ctx, deps)
	if err != nil {
		return nil, err
	}
	deps = depinject.Configs(deps, depinject.Supply(blockClient))

	// Create and supply the grpc client used by the queriers
	// TODO_TECHDEBT: Configure the grpc client options from the config.
	var grpcClient grpctypes.ClientConn
	grpcClient, err = grpc.Dial(
		config.QueryNodeGRPCUrl.Host,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
	)
	if err != nil {
		return nil, err
	}
	deps = depinject.Configs(deps, depinject.Supply(grpcClient))

	// Create the account querier and add it to the pubKey client required dependencies.
	accountQuerier, err := query.NewAccountQuerier(deps)
	if err != nil {
		return nil, err
	}

	// Create the pubKey client and add it to the required dependencies
	pubKeyClientDeps := depinject.Supply(accountQuerier)
	pubKeyClient, err := pubkeyclient.NewPubKeyClient(pubKeyClientDeps)
	if err != nil {
		return nil, err
	}
	deps = depinject.Configs(deps, depinject.Supply(pubKeyClient))

	// Create and supply the application querier
	applicationQuerier, err := query.NewApplicationQuerier(deps)
	if err != nil {
		return nil, err
	}
	deps = depinject.Configs(deps, depinject.Supply(applicationQuerier))

	// Create and supply the session querier
	sessionQuerier, err := query.NewSessionQuerier(deps)
	if err != nil {
		return nil, err
	}
	deps = depinject.Configs(deps, depinject.Supply(sessionQuerier))

	// Create and supply the delegation client
	delegationClient, err := delegation.NewDelegationClient(ctx, deps)
	if err != nil {
		return nil, err
	}
	deps = depinject.Configs(deps, depinject.Supply(delegationClient))

	// Create and supply the ring cache that depends on:
	// the logger, application and account querier and the delegation client
	ringCache, err := rings.NewRingCache(deps)
	if err != nil {
		return nil, err
	}
	deps = depinject.Configs(deps, depinject.Supply(ringCache))

	return deps, nil
}
