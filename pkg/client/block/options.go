package block

import "github.com/pokt-network/poktroll/pkg/client"

// WithConnRetryLimit returns an option function which sets the number
// of times the underlying replay client should retry in the event that it encounters
// an error or its connection is interrupted.
// If connRetryLimit is < 0, it will retry indefinitely.
func WithConnRetryLimit(limit int) client.BlockClientOption {
	return func(client client.BlockClient) {
		client.(*blockReplayClient).connRetryLimit = limit
	}
}
