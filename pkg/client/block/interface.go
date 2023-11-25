//go:generate mockgen -destination=../../../testutil/mockclient/block/block_client_mock.go -package=mockblockclient . Client
package block

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
)

type (
	// Observable wraps the generic observable.ReplayObservable[Block] type
	Observable observable.ReplayObservable[client.Block]

	// Client is an interface that wraps the EventsReplayClient interface
	// specific for the EventsReplayClient[Block] implementation
	Client interface {
		// CommittedBlockSequence returns a BlockObservable that emits the
		// latest block that has been committed to the chain.
		CommittedBlockSequence(context.Context) Observable
		// LastNBlocks returns the latest N blocks that have been committed to
		// the chain.
		LastNBlocks(context.Context, int) []client.Block
		// Close unsubscribes all observers of the committed block sequence
		// observable and closes the events query client.
		Close()
	}
)
