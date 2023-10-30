package tx

import (
	"pocket/pkg/client"
)

// WithCommitTimeoutBlocks sets the timeout duration in terms of number of blocks
// for the client to wait for broadcast transactions to be committed before
// returning a timeout error.
func WithCommitTimeoutBlocks(timeout int64) client.TxClientOption {
	return func(client client.TxClient) {
		client.(*txClient).commitTimeoutHeightOffset = timeout
	}
}

func WithSigningKeyName(keyName string) client.TxClientOption {
	return func(client client.TxClient) {
		client.(*txClient).signingKeyName = keyName
	}
}
