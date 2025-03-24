package websocket

import (
	"context"
	"crypto/tls"
	"strings"

	"github.com/gorilla/websocket"

	"github.com/pokt-network/pocket/pkg/client"
)

const wssPrefix = "wss://"

var _ client.Dialer = (*websocketDialer)(nil)

// websocketDialer implements the Dialer interface using the gorilla websocket
// transport implementation.
type websocketDialer struct{}

// NewWebsocketDialer creates a new websocketDialer.
func NewWebsocketDialer() client.Dialer {
	return &websocketDialer{}
}

// DialContext implements the respective interface method using the default gorilla
// websocket dialer.
func (wsDialer *websocketDialer) DialContext(
	ctx context.Context,
	urlString string,
) (client.Connection, error) {
	dialer := websocket.DefaultDialer

	if strings.HasPrefix(urlString, wssPrefix) {
		dialer.TLSClientConfig = &tls.Config{}
	}

	// TODO_IMPROVE: check http response status and potential err
	// TODO_TECHDEBT: add test coverage and ensure support for a 3xx responses
	conn, _, err := dialer.DialContext(ctx, urlString, nil)
	if err != nil {
		return nil, err
	}
	return &websocketConn{conn: conn}, nil
}
