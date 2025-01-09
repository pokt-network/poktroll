package e2e

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/gorilla/websocket"
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

// TODO_IN_THIS_COMMIT: move
var mockBlockResultJSON = `
{
	"query" : "tm.event='NewBlock'",
	"data" : {
	  "type" : "tendermint/event/NewBlock",
	  "value" : {
		"block" : {
		  "header" : {
			"version" : {
			  "block" : "11"
			},
			"chain_id" : "poktroll",
			"height" : "7554",
			"time" : "2025-01-03T14:55:39.944873259Z",
			"last_block_id" : {
			  "hash" : "BC2C6FDDAC8A8A7CF79D4682FC3E76AAFCACB84974FC7C37577322D00D7C7525",
			  "parts" : {
				"total" : 1,
				"hash" : "478F75135E06F132E99770C9F6B3D276157532A97252B0D7EDF153E0C8143E80"
			  }
			},
			"last_commit_hash" : "13D79604C6171FCCA01F5F9CA85F03973F4AF6D367E3FED27CAD4B8F87B5A01F",
			"data_hash" : "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
			"validators_hash" : "100E8F1C31B34BBF84E5832CDE10B33EB1D8CA28E5761278A4347EA149CD49ED",
			"next_validators_hash" : "100E8F1C31B34BBF84E5832CDE10B33EB1D8CA28E5761278A4347EA149CD49ED",
			"consensus_hash" : "048091BC7DDC283F77BFBF91D73C44DA58C3DF8A9CBC867405D8B7F3DAADA22F",
			"app_hash" : "BC5335B4EEF06E71C08FC051C89BB965F77B5364E55495E0487294C0AF92C251",
			"last_results_hash" : "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
			"evidence_hash" : "E3B0C44298FC1C149AFBF4C8996FB92427AE41E4649B934CA495991B7852B855",
			"proposer_address" : "294A784968AC4F506E9DF5E672C430B821F1362F"
		  },
		  "data" : {
			"txs" : [ ]
		  },
		  "evidence" : {
			"evidence" : [ ]
		  },
		  "last_commit" : {
			"height" : "7553",
			"round" : 0,
			"block_id" : {
			  "hash" : "BC2C6FDDAC8A8A7CF79D4682FC3E76AAFCACB84974FC7C37577322D00D7C7525",
			  "parts" : {
				"total" : 1,
				"hash" : "478F75135E06F132E99770C9F6B3D276157532A97252B0D7EDF153E0C8143E80"
			  }
			},
			"signatures" : [ {
			  "block_id_flag" : 2,
			  "validator_address" : "294A784968AC4F506E9DF5E672C430B821F1362F",
			  "timestamp" : "2025-01-03T14:55:39.944873259Z",
			  "signature" : "Fur3Gg1nLlzpRuN4aBY90Xb/BZVBGl57zjbMkPscycAi8/cGBYub/EHUTQbxUZjchQ0h1hcQlDQNc2Eu1oiiBQ=="
			} ]
		  }
		},
		"block_id" : {
		  "hash" : "5F1522B51BCE44338C1ED0D4BBAC048CA149EA14D2488C2AC730FFE4F206FBC8",
		  "parts" : {
			"total" : 1,
			"hash" : "1DF66D12EF879200392F078033C7CE314AA57B1BE38084C20C526AAD42DF90EE"
		  }
		},
		"result_finalize_block" : {
		  "events" : [ {
			"type" : "coin_spent",
			"attributes" : [ {
			  "key" : "spender",
			  "value" : "pokt1m3h30wlvsf8llruxtpukdvsy0km2kum84hcvmc",
			  "index" : true
			}, {
			  "key" : "amount",
			  "value" : "",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  }, {
			"type" : "coin_received",
			"attributes" : [ {
			  "key" : "receiver",
			  "value" : "pokt17xpfvakm2amg962yls6f84z3kell8c5ldlu5h9",
			  "index" : true
			}, {
			  "key" : "amount",
			  "value" : "",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  }, {
			"type" : "transfer",
			"attributes" : [ {
			  "key" : "recipient",
			  "value" : "pokt17xpfvakm2amg962yls6f84z3kell8c5ldlu5h9",
			  "index" : true
			}, {
			  "key" : "sender",
			  "value" : "pokt1m3h30wlvsf8llruxtpukdvsy0km2kum84hcvmc",
			  "index" : true
			}, {
			  "key" : "amount",
			  "value" : "",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  }, {
			"type" : "message",
			"attributes" : [ {
			  "key" : "sender",
			  "value" : "pokt1m3h30wlvsf8llruxtpukdvsy0km2kum84hcvmc",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  }, {
			"type" : "mint",
			"attributes" : [ {
			  "key" : "bonded_ratio",
			  "value" : "0.000000000000013043",
			  "index" : true
			}, {
			  "key" : "inflation",
			  "value" : "0.000000000000000000",
			  "index" : true
			}, {
			  "key" : "annual_provisions",
			  "value" : "0.000000000000000000",
			  "index" : true
			}, {
			  "key" : "amount",
			  "value" : "0",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  }, {
			"type" : "coin_spent",
			"attributes" : [ {
			  "key" : "spender",
			  "value" : "pokt17xpfvakm2amg962yls6f84z3kell8c5ldlu5h9",
			  "index" : true
			}, {
			  "key" : "amount",
			  "value" : "",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  }, {
			"type" : "coin_received",
			"attributes" : [ {
			  "key" : "receiver",
			  "value" : "pokt1jv65s3grqf6v6jl3dp4t6c9t9rk99cd86emg48",
			  "index" : true
			}, {
			  "key" : "amount",
			  "value" : "",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  }, {
			"type" : "transfer",
			"attributes" : [ {
			  "key" : "recipient",
			  "value" : "pokt1jv65s3grqf6v6jl3dp4t6c9t9rk99cd86emg48",
			  "index" : true
			}, {
			  "key" : "sender",
			  "value" : "pokt17xpfvakm2amg962yls6f84z3kell8c5ldlu5h9",
			  "index" : true
			}, {
			  "key" : "amount",
			  "value" : "",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  }, {
			"type" : "message",
			"attributes" : [ {
			  "key" : "sender",
			  "value" : "pokt17xpfvakm2amg962yls6f84z3kell8c5ldlu5h9",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  }, {
			"type" : "commission",
			"attributes" : [ {
			  "key" : "amount",
			  "value" : "",
			  "index" : true
			}, {
			  "key" : "validator",
			  "value" : "poktvaloper18kk3aqe2pjz7x7993qp2pjt95ghurra9c5ef0t",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  }, {
			"type" : "rewards",
			"attributes" : [ {
			  "key" : "amount",
			  "value" : "",
			  "index" : true
			}, {
			  "key" : "validator",
			  "value" : "poktvaloper18kk3aqe2pjz7x7993qp2pjt95ghurra9c5ef0t",
			  "index" : true
			}, {
			  "key" : "mode",
			  "value" : "BeginBlock",
			  "index" : true
			} ]
		  } ],
		  "validator_updates" : [ ],
		  "consensus_param_updates" : {
			"block" : {
			  "max_bytes" : "22020096",
			  "max_gas" : "-1"
			},
			"evidence" : {
			  "max_age_num_blocks" : "100000",
			  "max_age_duration" : "172800000000000",
			  "max_bytes" : "1048576"
			},
			"validator" : {
			  "pub_key_types" : [ "ed25519" ]
			},
			"version" : { },
			"abci" : { }
		  },
		  "app_hash" : "kPhqTfc+m5pORs+A7Y/eGTM8w0gdZXFMGW64KLnjGdU="
		}
	  }
	},
	"events" : {
	  "commission.amount" : [ "" ],
	  "commission.mode" : [ "BeginBlock" ],
	  "transfer.sender" : [ "pokt1m3h30wlvsf8llruxtpukdvsy0km2kum84hcvmc", "pokt17xpfvakm2amg962yls6f84z3kell8c5ldlu5h9" ],
	  "transfer.amount" : [ "", "" ],
	  "message.sender" : [ "pokt1m3h30wlvsf8llruxtpukdvsy0km2kum84hcvmc", "pokt17xpfvakm2amg962yls6f84z3kell8c5ldlu5h9" ],
	  "message.mode" : [ "BeginBlock", "BeginBlock" ],
	  "mint.bonded_ratio" : [ "0.000000000000013043" ],
	  "mint.mode" : [ "BeginBlock" ],
	  "rewards.validator" : [ "poktvaloper18kk3aqe2pjz7x7993qp2pjt95ghurra9c5ef0t" ],
	  "coin_spent.amount" : [ "", "" ],
	  "transfer.recipient" : [ "pokt17xpfvakm2amg962yls6f84z3kell8c5ldlu5h9", "pokt1jv65s3grqf6v6jl3dp4t6c9t9rk99cd86emg48" ],
	  "rewards.amount" : [ "" ],
	  "rewards.mode" : [ "BeginBlock" ],
	  "coin_spent.spender" : [ "pokt1m3h30wlvsf8llruxtpukdvsy0km2kum84hcvmc", "pokt17xpfvakm2amg962yls6f84z3kell8c5ldlu5h9" ],
	  "coin_received.receiver" : [ "pokt17xpfvakm2amg962yls6f84z3kell8c5ldlu5h9", "pokt1jv65s3grqf6v6jl3dp4t6c9t9rk99cd86emg48" ],
	  "coin_received.mode" : [ "BeginBlock", "BeginBlock" ],
	  "transfer.mode" : [ "BeginBlock", "BeginBlock" ],
	  "mint.inflation" : [ "0.000000000000000000" ],
	  "mint.amount" : [ "0" ],
	  "coin_spent.mode" : [ "BeginBlock", "BeginBlock" ],
	  "coin_received.amount" : [ "", "" ],
	  "mint.annual_provisions" : [ "0.000000000000000000" ],
	  "commission.validator" : [ "poktvaloper18kk3aqe2pjz7x7993qp2pjt95ghurra9c5ef0t" ],
	  "tm.event" : [ "NewBlock" ]
	}
}`

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

// handleBlockEvents coordinates block finalization with WebSocket event broadcasting
func (app *E2EApp) handleBlockEvents(t *testing.T) {
	for event := range app.blockEventChan {
		app.wsConnMutex.RLock()
		for conn, queries := range app.wsConnections {
			// Check if connection is subscribed to this event type
			for query := range queries {
				_ = query
				_ = event
				//if eventMatchesQuery(event, query) {
				//	// Marshal the event to JSON
				//	eventJSON, err := json.Marshal(event)
				//	if err != nil {
				//		t.Logf("failed to marshal event: %v", err)
				//		continue
				//	}

				response := rpctypes.RPCResponse{
					JSONRPC: "2.0",
					ID:      nil, // Events don't have an ID
					// TODO_IN_THIS_COMMIT: make this dynamic!
					Result: json.RawMessage(mockBlockResultJSON),
				}

				if err := conn.WriteJSON(response); err != nil {
					app.wsConnMutex.RUnlock()
					app.wsConnMutex.Lock()
					delete(app.wsConnections, conn)
					app.wsConnMutex.Unlock()
					app.wsConnMutex.RLock()
					continue
				}
				//}
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
