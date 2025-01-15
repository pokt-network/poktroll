package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	comettypes "github.com/cometbft/cometbft/types"
	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/require"
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
	logger := app.Logger().With("method", "handleWebSocketConnection")

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
		if err = json.Unmarshal(message, &req); err != nil {
			continue
		}

		// Handle subscription requests.
		if req.Method == "subscribe" {
			var params struct {
				Query string `json:"query"`
			}
			if err = json.Unmarshal(req.Params, &params); err != nil {
				continue
			}

			app.wsConnMutex.Lock()
			app.wsConnections[conn][params.Query] = struct{}{}
			app.wsConnMutex.Unlock()

			// Send initial subscription response
			resp := rpctypes.RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				// DEV_NOTE: Query subscription responses are initially empty; data is sent as subsequent events occur.
				Result: json.RawMessage("{}"),
			}
			if err = conn.WriteJSON(resp); err != nil {
				logger.Error(fmt.Sprintf("writing JSON-RPC response: %s", err))
			}
		}
	}
}

// handleResultEvents coordinates block finalization with WebSocket event broadcasting
func (app *E2EApp) handleResultEvents(t *testing.T) {
	t.Helper()

	for event := range app.resultEventChan {
		app.wsConnMutex.RLock()
		for conn, queries := range app.wsConnections {
			// Check if connection is subscribed to this event type
			for query := range queries {
				queryPartPairs := parseQuery(t, query)

				for queryKey, queryValue := range queryPartPairs {
					eventQueryValue, hasQueryKey := event.Events[queryKey]
					if !hasQueryKey {
						continue
					}

					// TODO_IN_THIS_COMMIT: comment explaining 0th index...
					if eventQueryValue[0] != strings.Trim(queryValue, "'") {
						continue
					}

					// DEV_NOTE: An empty request ID is consistent with the cometbft
					// implementation and is the reason that we MUST use a distinct
					// websocket connection per query; it's not possible to determine
					// to which query any given event corresponds.
					response := rpctypes.NewRPCSuccessResponse(nil, event)

					if err := conn.WriteJSON(response); err != nil {
						app.wsConnMutex.RUnlock()
						app.wsConnMutex.Lock()
						delete(app.wsConnections, conn)
						app.wsConnMutex.Unlock()
						app.wsConnMutex.RLock()
					}
				}
			}
		}
		app.wsConnMutex.RUnlock()
	}
}

// TODO_IN_THIS_COMMIT: godoc and move...
func parseQuery(t *testing.T, query string) map[string]string {
	t.Helper()

	queryParts := strings.Split(query, " AND ")
	queryPartPairs := make(map[string]string)
	for _, queryPart := range queryParts {
		queryPartPair := strings.Split(queryPart, "=")
		require.Equal(t, 2, len(queryPartPair))

		queryPartKey := strings.Trim(queryPartPair[0], `" `)
		queryPartValue := strings.Trim(queryPartPair[1], `" `)
		queryPartPairs[queryPartKey] = queryPartValue
	}

	return queryPartPairs
}

// TODO_IN_THIS_COMMIT: godoc...
func (app *E2EApp) EmitWSEvents(finalizeBlockRes *abci.ResponseFinalizeBlock, txBz []byte) {
	events := validateAndStringifyEvents(finalizeBlockRes.GetEvents())
	// DEV_NOTE: see https://github.com/cometbft/cometbft/blob/v0.38.10/types/event_bus.go#L138
	events[comettypes.EventTypeKey] = append(events[comettypes.EventTypeKey], comettypes.EventNewBlock)

	evtDataNewBlock := comettypes.EventDataNewBlock{
		Block: &comettypes.Block{
			Header: comettypes.Header{
				//Version:            version.Consensus{},
				ChainID:     "poktroll-test",
				Height:      app.GetSdkCtx().BlockHeight(),
				Time:        time.Now(),
				LastBlockID: app.GetCometBlockID(),
				//LastCommitHash:     nil,
				//DataHash:           nil,
				//ValidatorsHash:     nil,
				//NextValidatorsHash: nil,
				//ConsensusHash:      nil,
				//AppHash:            nil,
				//LastResultsHash:    nil,
				//EvidenceHash:       nil,
				//ProposerAddress:    nil,
			},
			//Data:       comettypes.Data{},
			//Evidence:   comettypes.EvidenceData{},
			//LastCommit: nil,
		},
		BlockID:             app.GetCometBlockID(),
		ResultFinalizeBlock: *finalizeBlockRes,
	}

	// TODO_IN_THIS_COMMIT: comment...
	resultEvent := &coretypes.ResultEvent{
		Query:  comettypes.EventQueryNewBlock.String(),
		Data:   evtDataNewBlock,
		Events: events,
	}

	app.resultEventChan <- resultEvent

	// TODO_IN_THIS_COMMIT: comment...
	for idx, txResult := range finalizeBlockRes.GetTxResults() {
		events = validateAndStringifyEvents(txResult.GetEvents())
		// DEV_NOTE: see https://github.com/cometbft/cometbft/blob/v0.38.10/types/event_bus.go#L180
		events[comettypes.EventTypeKey] = append(events[comettypes.EventTypeKey], comettypes.EventTx)
		events[comettypes.TxHashKey] = append(events[comettypes.TxHashKey], fmt.Sprintf("%X", comettypes.Tx(txBz).Hash()))
		events[comettypes.TxHeightKey] = append(events[comettypes.TxHeightKey], fmt.Sprintf("%d", app.GetSdkCtx().BlockHeight()))

		evtDataTx := comettypes.EventDataTx{
			TxResult: abci.TxResult{
				Height: app.GetSdkCtx().BlockHeight(),
				Index:  uint32(idx),
				Tx:     txBz,
				Result: *txResult,
			},
		}

		resultEvent = &coretypes.ResultEvent{
			Query:  comettypes.EventQueryTx.String(),
			Data:   evtDataTx,
			Events: events,
		}

		app.resultEventChan <- resultEvent
	}

	// TODO_IN_THIS_COMMIT: emit individual events...
}

// TODO_IN_THIS_COMMIT: godoc... see: https://github.com/cometbft/cometbft/blob/v0.38.10/types/event_bus.go#L112
func validateAndStringifyEvents(events []abci.Event) map[string][]string {
	result := make(map[string][]string)
	for _, event := range events {
		if len(event.Type) == 0 {
			continue
		}

		for _, attr := range event.Attributes {
			if len(attr.Key) == 0 {
				continue
			}

			compositeTag := fmt.Sprintf("%s.%s", event.Type, attr.Key)
			result[compositeTag] = append(result[compositeTag], attr.Value)
		}
	}

	return result
}
