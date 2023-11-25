//go:generate mockgen -destination=../../../testutil/mockclient/delegation/delegation_client_mock.go -package=mockdelegationclient . Client
package delegation

import (
	"context"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/observable"
)

type (
	// Observable wraps the generic observable.ReplayObservable[DelegateeChange] type
	Observable observable.ReplayObservable[client.DelegateeChange]

	// Client is an interface that wraps the EventsReplayClient interface
	// specific for the EventsReplayClient[DelegateeChange] implementation
	Client interface {
		// DelegateeChangesSequence returns a Observable of DelegateeChanges that
		// emits the latest delegatee change that has occured on chain.
		DelegateeChangesSequence(context.Context) Observable
		// LastNBlocks returns the latest N blocks that have been committed to
		// the chain.
		LastNDelegateeChanges(context.Context, int) []client.DelegateeChange
		// Close unsubscribes all observers of the committed block sequence
		// observable and closes the events query client.
		Close()
	}
)
