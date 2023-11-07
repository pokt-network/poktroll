package testchannel

import (
	"time"

	"github.com/pokt-network/poktroll/pkg/observable"
)

// DrainChannel attempts to receive from the given channel, blocking, until it is
// empty. It returns an error if the channel is not closed by the time it's empty.
// TODO_CONSIDERATION: this function could easily take a timeout parameter and add
// a case which returns an error if the timeout is exceeded. This would prevent
// the case where the channel never stops receiving from looping indefinitely.
func DrainChannel[V any](ch <-chan V) error {
	for {
		select {
		case _, ok := <-ch:
			if ok {
				continue
			}
			return nil
		case <-time.After(time.Millisecond):
			return observable.ErrObserverClosed
		}
	}
}
