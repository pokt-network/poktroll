package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/gorilla/websocket"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
)

// newWebSocketServer creates and configures a new WebSocket server for the E2EApp
func newWebSocketServer(app *E2EApp) *http.Server {
	mux := http.NewServeMux()
	mux.HandleFunc("/websocket", app.handleWebSocket)
	return &http.Server{Handler: mux}
}

// handleWebSocket handles incoming WebSocket connections and subscriptions
func (app *E2EApp) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := app.wsUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	app.wsConnMutex.Lock()
	app.wsConnections[conn] = make(map[string]struct{})
	app.wsConnMutex.Unlock()

	go app.handleWebSocketConnection(conn)
}

// handleWebSocketConnection handles messages from a specific WebSocket connection
func (app *E2EApp) handleWebSocketConnection(conn *websocket.Conn) {
	defer func() {
		app.wsConnMutex.Lock()
		delete(app.wsConnections, conn)
		app.wsConnMutex.Unlock()
		conn.Close()
	}()

	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			return
		}

		var req rpctypes.RPCRequest
		if err := json.Unmarshal(message, &req); err != nil {
			continue
		}

		// Handle subscribe/unsubscribe requests
		if req.Method == "subscribe" {
			var params struct {
				Query string `json:"query"`
			}
			if err := json.Unmarshal(req.Params, &params); err != nil {
				continue
			}

			app.wsConnMutex.Lock()
			app.wsConnections[conn][params.Query] = struct{}{}
			app.wsConnMutex.Unlock()

			// Send subscription response
			resp := rpctypes.RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				// TODO_IN_THIS_COMMIT: generate a mock result...
				Result: json.RawMessage("{}"),
			}
			if err := conn.WriteJSON(resp); err != nil {
				return
			}
		}
	}
}

// handleBlockEvents coordinates block finalization with WebSocket event broadcasting
func (app *E2EApp) handleBlockEvents(t *testing.T) {
	for event := range app.blockEventChan {
		app.wsConnMutex.RLock()
		for conn, queries := range app.wsConnections {
			// Check if connection is subscribed to this event type
			for query := range queries {
				if eventMatchesQuery(event, query) {
					// Marshal the event to JSON
					eventJSON, err := json.Marshal(event)
					if err != nil {
						t.Logf("failed to marshal event: %v", err)
						continue
					}

					response := rpctypes.RPCResponse{
						JSONRPC: "2.0",
						ID:      nil, // Events don't have an ID
						Result:  json.RawMessage(eventJSON),
					}

					if err := conn.WriteJSON(response); err != nil {
						app.wsConnMutex.RUnlock()
						app.wsConnMutex.Lock()
						delete(app.wsConnections, conn)
						app.wsConnMutex.Unlock()
						app.wsConnMutex.RLock()
						continue
					}
				}
			}
		}
		app.wsConnMutex.RUnlock()
	}
}

// TODO_IN_THIS_COMMIT: also wrap RunMsgs...
// TODO_IN_THIS_COMMIT: godoc...
// Override RunMsg to also emit transaction events via WebSocket
func (app *E2EApp) RunMsg(t *testing.T, msg sdk.Msg) (tx.MsgResponse, error) {
	msgRes, err := app.App.RunMsg(t, msg)
	if err != nil {
		return nil, err
	}

	// Create and emit block event with transaction results
	blockEvent := createBlockEvent(app.GetSdkCtx(), msgRes)
	app.blockEventChan <- blockEvent

	return msgRes, nil
}

// createBlockEvent creates a CometBFT-compatible event from transaction results
func createBlockEvent(ctx *sdk.Context, msgRes tx.MsgResponse) *coretypes.ResultEvent {
	// Convert SDK events to map[string][]string format that CometBFT expects
	events := make(map[string][]string)
	for _, event := range ctx.EventManager().Events() {
		// Each event type becomes a key, and its attributes become the values
		for _, attr := range event.Attributes {
			if events[event.Type] == nil {
				events[event.Type] = make([]string, 0)
			}
			events[event.Type] = append(events[event.Type], string(attr.Value))
		}
	}

	return &coretypes.ResultEvent{
		Query: "tm.event='NewBlock'",
		Data: map[string]interface{}{
			"height": ctx.BlockHeight(),
			"hash":   ctx.BlockHeader().LastBlockId.Hash,
			"events": events,
			// Add other relevant block and transaction data here as needed
		},
		Events: events,
	}
}

//// createTxEvent creates a CometBFT-compatible event from transaction results
//func createTxEvent(tx *coretypes.ResultTx, index int) *coretypes.ResultEvent {
//	return &coretypes.ResultEvent{
//		Query: "tm.event='Tx'",
//		Data: map[string]interface{}{
//			"height": ctx.BlockHeight(),
//			"hash":   ctx.BlockHeader().LastBlockId.Hash,
//			"events": events,
//			// Add other relevant block and transaction data here as needed
//		},
//		Events: events,
//	}
//}

// eventMatchesQuery checks if an event matches a subscription query
func eventMatchesQuery(event *coretypes.ResultEvent, query string) bool {
	// Basic implementation - should be expanded to handle more complex queries
	return strings.Contains(query, event.Query)
}
