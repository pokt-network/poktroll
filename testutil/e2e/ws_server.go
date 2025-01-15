package e2e

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	abci "github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	comettypes "github.com/cometbft/cometbft/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
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

		// Handle subscribe/unsubscribe requests
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

			// Send subscription response
			resp := rpctypes.RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				// TODO_IN_THIS_COMMIT: generate a mock result...
				//Result: json.RawMessage(mockBlockResultJSON),
				// DEV_NOTE: Query subscription responses are initially empty; data is sent as subsequent events occur.
				Result: json.RawMessage("{}"),
			}

			//time.Sleep(time.Second * 4)

			if err = conn.WriteJSON(resp); err != nil {
				panic(err)
			}
			//if err = conn.WriteJSON(resp); err != nil {
			//	return
			//}
		}
	}
}

// handleResultEvents coordinates block finalization with WebSocket event broadcasting
func (app *E2EApp) handleResultEvents(t *testing.T) {
	t.Helper()

	for event := range app.resultEventChan {
		fmt.Printf(">>> WS event: %+v\n", event)
		fmt.Printf(">>> num WS conns: %d\n", len(app.wsConnections))

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

					fmt.Printf(">>> checking query: %s\n", query)

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
						continue
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

//// TODO_IN_THIS_COMMIT: also wrap RunMsgs...
//// TODO_IN_THIS_COMMIT: godoc...
//// Override RunMsg to also emit transaction events via WebSocket
//func (app *E2EApp) RunMsg(t *testing.T, msg cosmostypes.Msg) (tx.MsgResponse, error) {
//	msgRes, err := app.App.RunMsg(t, msg)
//	if err != nil {
//		return nil, err
//	}
//
//	// Create and emit block event with transaction results
//	blockEvent := createBlockEvent(app.GetSdkCtx())
//	app.resultEventChan <- blockEvent
//
//	return msgRes, nil
//}

// createBlockEvent creates a CometBFT-compatible event from transaction results
func createBlockEvent(ctx *cosmostypes.Context) *coretypes.ResultEvent {
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

// TODO_IN_THIS_COMMIT: godoc...
func (app *E2EApp) EmitWSEvents(finalizeBlockRes *abci.ResponseFinalizeBlock, txBz []byte) {
	//resultEvent := &coretypes.ResultEvent{
	//	Query: "tm.event='NewBlock'",
	//	Data:  map[string]interface{}{
	//		//"height": ctx.BlockHeight(),
	//		//"hash":   ctx.BlockHeader().LastBlockId.Hash,
	//		//"events": events,
	//		// Add other relevant block and transaction data here as needed
	//	},
	//	//Events: events,
	//}

	//emitEvent := func(event abci.Event, query string) error {
	//	eventAny, err := codectypes.NewAnyWithValue(&event)
	//	if err != nil {
	//		return err
	//	}
	//
	//	resultEvent := &coretypes.ResultEvent{
	//		Query:  query,
	//		Data:   eventAny,
	//		Events: nil,
	//	}
	//
	//	app.resultEventChan <- resultEvent
	//
	//	return nil
	//}
	//for _, event := range finalizeBlockRes.GetEvents() {
	//	// TODO_IN_THIS_COMMIT: reconsider how to populate the queries...
	//	if err := emitEvent(event, comettypes.EventQueryNewBlock.String()); err != nil {
	//		app.Logger().Error(err.Error())
	//	}
	//}
	//for _, txResult := range finalizeBlockRes.GetTxResults() {
	//	for _, event := range txResult.GetEvents() {
	//		// TODO_IN_THIS_COMMIT: reconsider how to populate the queries...
	//		if err := emitEvent(event, comettypes.EventQueryTx.String()); err != nil {
	//			app.Logger().Error(err.Error())
	//		}
	//	}
	//}

	// TODO_IN_THIS_COMMIT: necessary?
	//app.wsConnMutex.RLock()
	//defer app.wsConnMutex.RUnlock()

	events := validateAndStringifyEvents(finalizeBlockRes.GetEvents())
	// DEV_NOTE: see https://github.com/cometbft/cometbft/blob/v0.38.10/types/event_bus.go#L138
	events[comettypes.EventTypeKey] = append(events[comettypes.EventTypeKey], comettypes.EventNewBlock)

	evtDataNewBlock := comettypes.EventDataNewBlock{
		// TODO_IN_THIS_COMMIT: add block...
		Block:               nil,
		BlockID:             app.GetCometBlockID(),
		ResultFinalizeBlock: abci.ResponseFinalizeBlock{},
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
