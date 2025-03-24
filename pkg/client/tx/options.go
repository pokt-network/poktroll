package tx

import (
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/pocket/pkg/client"
)

// WithCommitTimeoutBlocks sets the timeout duration in terms of number of blocks
// for the client to wait for broadcast transactions to be committed before
// returning a timeout error.
func WithCommitTimeoutBlocks(timeout int64) client.TxClientOption {
	return func(client client.TxClient) {
		client.(*txClient).commitTimeoutHeightOffset = timeout
	}
}

// WithSigningKeyName sets the name of the key which should be retrieved from the
// keyring and used for signing transactions.
func WithSigningKeyName(keyName string) client.TxClientOption {
	return func(client client.TxClient) {
		client.(*txClient).signingKeyName = keyName
	}
}

// WithConnRetryLimit returns an option function which sets the number
// of times the underlying replay client should retry in the event that it encounters
// an error or its connection is interrupted.
// If connRetryLimit is < 0, it will retry indefinitely.
func WithConnRetryLimit(limit int) client.TxClientOption {
	return func(client client.TxClient) {
		client.(*txClient).connRetryLimit = limit
	}
}

// WithGasPrices sets the gas price to be used when constructing transactions.
func WithGasPrices(gasPrices cosmostypes.DecCoins) client.TxClientOption {
	return func(client client.TxClient) {
		client.(*txClient).gasPrices = gasPrices
	}
}
