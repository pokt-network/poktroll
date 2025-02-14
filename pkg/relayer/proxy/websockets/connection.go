package websockets

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/websocket"

	"github.com/pokt-network/poktroll/pkg/polylog"
)

const (
	// Time allowed (in seconds) to write a message to the peer.
	writeWaitSec = 10 * time.Second

	// Time allowed (in seconds) to read the next pong message from the peer.
	pongWaitSec = 30 * time.Second

	// Send pings to peer with this period.
	// Must be less than pongWaitSec.
	pingPeriodSec = (pongWaitSec * 9) / 10
)

type messageSource string

const (
	messageSourceServiceBackend messageSource = "service_backend"
	messageSourceGateway        messageSource = "gateway"
)

type message struct {
	// data is the message payload
	data []byte

	// source may be either `client` or `endpoint`
	source messageSource

	// messageType is an int returned by the gorilla/websocket package
	messageType int
}

type connection struct {
	*websocket.Conn

	logger polylog.Logger

	source   messageSource
	msgChan  chan message
	stopChan chan error
}

func connectServiceBackend(serviceBackendUrl *url.URL, header http.Header) (*websocket.Conn, error) {
	var dialer *websocket.Dialer
	switch serviceBackendUrl.Scheme {
	case "wss":
		dialer = &websocket.Dialer{TLSClientConfig: &tls.Config{}}
	default:
		dialer = websocket.DefaultDialer
	}

	conn, _, err := dialer.Dial(serviceBackendUrl.String(), header)
	if err != nil {
		return nil, err
	}

	return conn, nil
}

func newConnection(
	logger polylog.Logger,
	conn *websocket.Conn,
	source messageSource,
	msgChan chan message,
	stopChan chan error,
) *connection {
	c := &connection{
		Conn:     conn,
		logger:   logger,
		source:   source,
		msgChan:  msgChan,
		stopChan: stopChan,
	}

	go c.connLoop()
	go c.pingLoop()

	return c
}

func (c *connection) connLoop() {
	for {
		select {
		case err := <-c.stopChan:
			if err := c.cleanup(err); err != nil {
				c.logger.Error().Err(err).Msg("cleaning up connection")
			}
			return
		default:
			messageType, msg, err := c.ReadMessage()
			if err != nil {
				c.handleError(err, c.source)
				return
			}

			c.msgChan <- message{
				data:        msg,
				source:      c.source,
				messageType: messageType,
			}
		}
	}
}

func (c *connection) pingLoop() {
	ticker := time.NewTicker(pingPeriodSec)
	defer ticker.Stop()

	if err := c.SetReadDeadline(time.Now().Add(pongWaitSec)); err != nil {
		c.logger.Error().Err(err).Msg("setting read deadline")
	}

	c.SetPongHandler(func(string) error {
		if err := c.SetReadDeadline(time.Now().Add(pongWaitSec)); err != nil {
			c.logger.Error().Err(err).Msg("setting read deadline")
		}

		return nil
	})

	for {
		select {
		case <-c.stopChan:
			return

		case <-ticker.C:
			if err := c.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWaitSec)); err != nil {
				c.logger.Error().Err(err).Msg("sending ping")
				c.stopChan <- err
				return
			}
		}
	}
}

func (c *connection) handleError(err error, source messageSource) {
	if websocket.IsCloseError(err, websocket.CloseNoStatusReceived) {
		c.logger.Info().Err(err).Msgf("connection closed by peer", source)
	} else {
		c.logger.Error().Err(err).Msgf("%s connection error", source)
	}

	select {
	case <-c.stopChan:
	default:
		c.stopChan <- err
	}
}

func (c *connection) cleanup(err error) error {
	closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, err.Error())

	if err := c.WriteControl(websocket.CloseMessage, closeMsg, time.Now().Add(writeWaitSec)); err != nil {
		c.logger.Error().Err(err).Msg("sending close message")
	}
	if err := c.Close(); err != nil {
		c.logger.Error().Err(err).Msg("closing connection")
		return err
	}

	return nil
}
