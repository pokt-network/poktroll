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

// TODO_IN_THIS_COMMIT: move
var mockBlockResultJSON = `{
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

const mockTxResultEventJSON = `{
"query" : "tm.event='Tx' AND message.sender='pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw'",
"data" : {
  "type" : "tendermint/event/Tx",
  "value" : {
	"TxResult" : {
	  "height" : "471",
	  "tx" : "CpYBCpABChwvY29zbW9zLmJhbmsudjFiZXRhMS5Nc2dTZW5kEnAKK3Bva3QxZWVla3NoMnR2a2g3d3ptZnJsam5odzR3cmhzNTVsY3V2bWVra3cSK3Bva3QxNXczZmhmeWMwbHR0djdyNTg1ZTJuY3BmNnQya2w5dWg4cnNueXoaFAoFdXBva3QSCzEwMDAwMDAwMDAwGIY7EloKUApGCh8vY29zbW9zLmNyeXB0by5zZWNwMjU2azEuUHViS2V5EiMKIQLcGKC0uUKBIbYq4/pz+5scHn0Xk/qiXX23JtYSW0rpTRIECgIIARgEEgYQqqGCyQIaQDXGt2kVA/IWj/7HMgfX3fK5tJrwJ7V0nhWAeDlnpcFcJM7k958ee9gJvk8HjRL2BOD97PEnXalu/zFu+YYChuw=",
	  "result" : {
		"data" : "EiYKJC9jb3Ntb3MuYmFuay52MWJldGExLk1zZ1NlbmRSZXNwb25zZQ==",
		"gas_wanted" : "690000042",
		"gas_used" : "48137",
		"events" : [ {
		  "type" : "tx",
		  "attributes" : [ {
			"key" : "fee",
			"value" : "",
			"index" : true
		  }, {
			"key" : "fee_payer",
			"value" : "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw",
			"index" : true
		  } ]
		}, {
		  "type" : "tx",
		  "attributes" : [ {
			"key" : "acc_seq",
			"value" : "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw/4",
			"index" : true
		  } ]
		}, {
		  "type" : "tx",
		  "attributes" : [ {
			"key" : "signature",
			"value" : "Nca3aRUD8haP/scyB9fd8rm0mvAntXSeFYB4OWelwVwkzuT3nx572Am+TweNEvYE4P3s8SddqW7/MW75hgKG7A==",
			"index" : true
		  } ]
		}, {
		  "type" : "message",
		  "attributes" : [ {
			"key" : "action",
			"value" : "/cosmos.bank.v1beta1.MsgSend",
			"index" : true
		  }, {
			"key" : "sender",
			"value" : "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw",
			"index" : true
		  }, {
			"key" : "module",
			"value" : "bank",
			"index" : true
		  }, {
			"key" : "msg_index",
			"value" : "0",
			"index" : true
		  } ]
		}, {
		  "type" : "coin_spent",
		  "attributes" : [ {
			"key" : "spender",
			"value" : "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw",
			"index" : true
		  }, {
			"key" : "amount",
			"value" : "10000000000upokt",
			"index" : true
		  }, {
			"key" : "msg_index",
			"value" : "0",
			"index" : true
		  } ]
		}, {
		  "type" : "coin_received",
		  "attributes" : [ {
			"key" : "receiver",
			"value" : "pokt15w3fhfyc0lttv7r585e2ncpf6t2kl9uh8rsnyz",
			"index" : true
		  }, {
			"key" : "amount",
			"value" : "10000000000upokt",
			"index" : true
		  }, {
			"key" : "msg_index",
			"value" : "0",
			"index" : true
		  } ]
		}, {
		  "type" : "transfer",
		  "attributes" : [ {
			"key" : "recipient",
			"value" : "pokt15w3fhfyc0lttv7r585e2ncpf6t2kl9uh8rsnyz",
			"index" : true
		  }, {
			"key" : "sender",
			"value" : "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw",
			"index" : true
		  }, {
			"key" : "amount",
			"value" : "10000000000upokt",
			"index" : true
		  }, {
			"key" : "msg_index",
			"value" : "0",
			"index" : true
		  } ]
		}, {
		  "type" : "message",
		  "attributes" : [ {
			"key" : "sender",
			"value" : "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw",
			"index" : true
		  }, {
			"key" : "msg_index",
			"value" : "0",
			"index" : true
		  } ]
		} ]
	  }
	}
  }
},
"events" : {
  "coin_received.receiver" : [ "pokt15w3fhfyc0lttv7r585e2ncpf6t2kl9uh8rsnyz" ],
  "tx.fee" : [ "" ],
  "message.sender" : [ "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw", "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw" ],
  "coin_received.amount" : [ "10000000000upokt" ],
  "coin_received.msg_index" : [ "0" ],
  "transfer.recipient" : [ "pokt15w3fhfyc0lttv7r585e2ncpf6t2kl9uh8rsnyz" ],
  "transfer.amount" : [ "10000000000upokt" ],
  "tm.event" : [ "Tx" ],
  "tx.acc_seq" : [ "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw/4" ],
  "coin_spent.amount" : [ "10000000000upokt" ],
  "coin_spent.spender" : [ "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw" ],
  "transfer.msg_index" : [ "0" ],
  "tx.hash" : [ "119E4CAC3E395B256C9F87E5B2295DAC687C6312CCA9C701176A25153EE03B1A" ],
  "tx.fee_payer" : [ "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw" ],
  "tx.signature" : [ "Nca3aRUD8haP/scyB9fd8rm0mvAntXSeFYB4OWelwVwkzuT3nx572Am+TweNEvYE4P3s8SddqW7/MW75hgKG7A==" ],
  "message.msg_index" : [ "0", "0" ],
  "coin_spent.msg_index" : [ "0" ],
  "transfer.sender" : [ "pokt1eeeksh2tvkh7wzmfrljnhw4wrhs55lcuvmekkw" ],
  "tx.height" : [ "471" ],
  "message.action" : [ "/cosmos.bank.v1beta1.MsgSend" ],
  "message.module" : [ "bank" ]
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
