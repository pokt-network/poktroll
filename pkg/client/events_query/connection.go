package eventsquery

import (
	"github.com/gorilla/websocket"

	"pocket/pkg/client"
)

var _ client.Connection = &websocketConn{}

// websocketConn implements the Connection interface using the gorilla websocket
// transport implementation.
type websocketConn struct {
	conn *websocket.Conn
}

// Receive implements the respective interface method using the underlying websocket.
func (wsConn *websocketConn) Receive() ([]byte, error) {
	_, msg, err := wsConn.conn.ReadMessage()
	if err != nil {
		return nil, ErrReceive.Wrapf("%s", err)
	}
	return msg, nil
}

// Send implements the respective interface method using the underlying websocket.
func (wsConn *websocketConn) Send(msg []byte) error {
	return wsConn.conn.WriteMessage(websocket.TextMessage, msg)
}

// Close implements the respective interface method using the underlying websocket.
func (wsConn *websocketConn) Close() error {
	return wsConn.conn.Close()
}
