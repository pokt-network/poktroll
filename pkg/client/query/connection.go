package query

import (
	"github.com/gorilla/websocket"

	"pocket/pkg/client"
)

var _ client.Connection = &websocketConn{}

type websocketConn struct {
	conn *websocket.Conn
}

func (wsConn *websocketConn) ReadMessage() ([]byte, error) {
	_, msg, err := wsConn.conn.ReadMessage()
	return msg, err
}

func (wsConn *websocketConn) WriteJSON(any interface{}) error {
	return wsConn.conn.WriteJSON(any)
}

func (wsConn *websocketConn) Close() error {
	return wsConn.conn.Close()
}
