package query

import (
	"context"
	"github.com/gorilla/websocket"

	"pocket/pkg/client"
)

var _ client.Dialer = &websocketDialer{}

type websocketDialer struct{}

func NewWebsocketDialer() client.Dialer {
	return &websocketDialer{}
}

func (wsDialer *websocketDialer) DialContext(
	ctx context.Context,
	urlString string,
) (client.Connection, error) {
	// TODO_THIS_COMMIT: check http response status/err?
	conn, _, err := websocket.DefaultDialer.DialContext(ctx, urlString, nil)
	if err != nil {
		return nil, err
	}
	return &websocketConn{conn: conn}, nil
}
