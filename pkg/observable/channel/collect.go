package channel

import (
	"context"
	"sync"

	"github.com/pokt-network/poktroll/pkg/observable"
)

// Collect collects all notifications received from the observable and returns
// them as a slice. ctx MUST be canceled, after some finite duration as it blocks
// until either srcObservable is closed OR ctx is canceled. Collect is a terminal
// observable operator.
func Collect[V any](
	ctx context.Context,
	srcObservable observable.Observable[V],
) (dstCollection []V) {
	var dstCollectionMu sync.Mutex
	// Defer unlocking as lock is acquired immediately before returning.
	defer dstCollectionMu.Unlock()

	ForEach(ctx, srcObservable, func(ctx context.Context, src V) {
		dstCollectionMu.Lock()
		dstCollection = append(dstCollection, src)
		dstCollectionMu.Unlock()
	})

	// Wait for context to be done before returning.
	<-ctx.Done()

	// Lock to read from dstCollection in return.
	dstCollectionMu.Lock()
	return dstCollection
}
