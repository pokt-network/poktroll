package delegation

import (
	"context"

	"cosmossdk.io/depinject"

	"github.com/pokt-network/poktroll/pkg/client"
	mappedclient "github.com/pokt-network/poktroll/pkg/client/mapped_client"
)

const delegationEventQuery = "tm.event='Tx' AND message.action='pocket.application.EventDelegateeChange'"

// NewDelegationClient creates a new delegation client from the given
// dependencies and cometWebsocketURL. It uses the defined delegationEventQuery
// to subscribe to new delegation events and maps them to DelegateeChange
// objects.
//
// Required dependencies:
//   - client.EventsQueryClient
func NewDelegationClient(
	ctx context.Context,
	deps depinject.Config,
	cometWebsocketURL string,
) (client.MappedClient[client.DelegateeChange, client.EventsObservable[client.DelegateeChange]], error) {
	client, err := mappedclient.NewMappedClient[client.DelegateeChange, client.EventsObservable[client.DelegateeChange]](
		ctx,
		deps,
		cometWebsocketURL,
		delegationEventQuery,
		newDelegateeChangeEvent,
	)
	if err != nil {
		return nil, err
	}
	return client, nil
}
