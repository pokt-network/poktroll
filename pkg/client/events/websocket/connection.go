package websocket

import (
	gorillaws "github.com/gorilla/websocket"

	"github.com/pokt-network/poktroll/pkg/client"
)

var _ client.Connection = (*websocketConn)(nil)

// websocketConn implements the Connection interface using the gorilla websocket
// transport implementation.
type websocketConn struct {
	conn *gorillaws.Conn
}

// Receive implements the respective interface method using the underlying websocket.
func (wsConn *websocketConn) Receive() ([]byte, error) {
	_, msg, err := wsConn.conn.ReadMessage()
	if err != nil {
		return nil, ErrEventsWebsocketReceive.Wrapf("%s", err)
	}
	return msg, nil
}

// Send implements the respective interface method using the underlying websocket.
func (wsConn *websocketConn) Send(msg []byte) error {
	// Using the TextMessage message to indicate that msg is UTF-8 encoded.
	return wsConn.conn.WriteMessage(gorillaws.TextMessage, msg)
}

// Close implements the respective interface method using the underlying websocket.
func (wsConn *websocketConn) Close() error {
	return wsConn.conn.Close()
}
