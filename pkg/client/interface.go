package client

import (
	"context"
)

// TODO_CONSIDERATION: the cosmos-sdk CLI code seems to use a cometbft RPC client
// which includes a `#Subscribe()` method for a similar purpose. Perhaps we could
// replace this custom websocket client with that.
// (see: https://github.com/cometbft/cometbft/blob/main/rpc/client/http/http.go#L110)
// (see: https://github.com/cosmos/cosmos-sdk/blob/main/client/rpc/tx.go#L114)

// QueryClient is used to subscribe to chain event messages matching the given query,
type QueryClient interface {
	SubscribeWithQuery(ctx context.Context, query string, msgHandler MessageHandler) chan error
}

// MessageHandler is a function that handles a websocket chain-event subscription message.
type MessageHandler func(ctx context.Context, msg []byte) error
