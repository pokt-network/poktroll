package block

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	mappedclient "github.com/pokt-network/poktroll/pkg/client/mapped_client"
)

const committedBlocksQuery = "tm.event='NewBlock'"

// NewBlockClient creates a new block client from the given dependencies and
// cometWebsocketURL. It uses the defined committedBlocksQuery to subscribe to
// newly committed block events and maps them to Block objects.
//
// Required dependencies:
//   - client.EventsQueryClient
func NewBlockClient(
	ctx context.Context,
	deps depinject.Config,
	cometWebsocketURL string,
) (client.MappedClient[client.Block], error) {
	client, err := mappedclient.NewMappedClient[client.Block](
		ctx,
		deps,
		cometWebsocketURL,
		committedBlocksQuery,
		newCometBlockEvent,
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}
