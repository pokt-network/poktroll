package e2e

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/cometbft/cometbft/abci/types"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	comettypes "github.com/cometbft/cometbft/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

// TODO_IN_THIS_COMMIT: godoc...
type CometBFTMethod string

// TODO_IN_THIS_COMMIT: godoc...
type ServiceMethodUri string

const (
	abciQueryMethod         = CometBFTMethod("abci_query")
	broadcastTxSyncMethod   = CometBFTMethod("broadcast_tx_sync")
	broadcastTxAsyncMethod  = CometBFTMethod("broadcast_tx_async")
	broadcastTxCommitMethod = CometBFTMethod("broadcast_tx_commit")

	authAccountQueryUri = ServiceMethodUri("/cosmos.auth.v1beta1.Query/Account")
)

// handleABCIQuery handles the actual ABCI query logic
func newPostHandler(client gogogrpc.ClientConn, app *E2EApp) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		ctx := context.Background()
		// DEV_NOTE: http.Error() automatically sets the Content-Type header to "text/plain".
		w.Header().Set("Content-Type", "application/json")

		// Read and log request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()

		// Parse JSON-RPC request
		var req rpctypes.RPCRequest
		if err = json.Unmarshal(body, &req); err != nil {
			writeErrorResponseFromErr(w, req, err)
			return
		}

		params := make(map[string]json.RawMessage)
		if err = json.Unmarshal(req.Params, &params); err != nil {
			writeErrorResponseFromErr(w, req, err)
			return
		}

		var response rpctypes.RPCResponse
		switch CometBFTMethod(req.Method) {
		// TODO_IN_THIS_COMMIT: extract...
		case abciQueryMethod:
			var (
				resData []byte
				height  int64
			)

			pathRaw, hasPath := params["path"]
			if !hasPath {
				writeErrorResponse(w, req, "missing path param", string(req.Params))
				return
			}

			var path string
			if err = json.Unmarshal(pathRaw, &path); err != nil {
				writeErrorResponseFromErr(w, req, err)
				return
			}

			switch ServiceMethodUri(path) {
			case authAccountQueryUri:
				dataRaw, hasData := params["data"]
				if !hasData {
					writeErrorResponse(w, req, "missing data param", string(req.Params))
					return
				}

				data, err := hex.DecodeString(string(bytes.Trim(dataRaw, `"`)))
				if err != nil {
					writeErrorResponseFromErr(w, req, err)
					return
				}

				queryReq := new(authtypes.QueryAccountRequest)
				if err = queryReq.Unmarshal(data); err != nil {
					writeErrorResponseFromErr(w, req, err)
					return
				}

				var height int64
				heightRaw, hasHeight := params["height"]
				if hasHeight {
					if err = json.Unmarshal(bytes.Trim(heightRaw, `"`), &height); err != nil {
						writeErrorResponseFromErr(w, req, err)
						return
					}
				}

				queryRes := new(authtypes.QueryAccountResponse)
				if err = client.Invoke(ctx, path, queryReq, queryRes); err != nil {
					writeErrorResponseFromErr(w, req, err)
					return
				}

				resData, err = queryRes.Marshal()
				if err != nil {
					writeErrorResponseFromErr(w, req, err)
					return
				}
			}

			abciQueryRes := coretypes.ResultABCIQuery{
				Response: types.ResponseQuery{
					//Code:      0,
					//Index:     0,
					//Key:       nil,
					Value:  resData,
					Height: height,
				},
			}

			response = rpctypes.NewRPCSuccessResponse(req.ID, abciQueryRes)
		case broadcastTxSyncMethod, broadcastTxAsyncMethod, broadcastTxCommitMethod:
			fmt.Println(">>>> BROADCAST_TX")

			var txBz []byte
			txRaw, hasTx := params["tx"]
			if !hasTx {
				writeErrorResponse(w, req, "missing tx param", string(req.Params))
				return
			}
			if err = json.Unmarshal(txRaw, &txBz); err != nil {
				writeErrorResponseFromErr(w, req, err)
				return
			}

			// TODO_CONSIDERATION: more correct implementation of the different
			// broadcast_tx methods (i.e. sync, async, commit) is a matter of
			// the sequencing of the following:
			// - calling the finalize block ABCI method
			// - returning the JSON-RPC response
			// - emitting websocket event

			_, finalizeBlockRes, err := app.RunTx(nil, txBz)
			if err != nil {
				writeErrorResponseFromErr(w, req, err)
				return
			}

			// TODO_IN_THIS_COMMIT: something better...
			go func() {
				// Simulate 1 second block production delay.
				time.Sleep(time.Second * 1)

				fmt.Println(">>> emitting ws events")
				//app.EmitWSEvents(app.GetSdkCtx().EventManager().Events())

				// TODO_IMPROVE: If we want/need to support multiple txs per
				// block in the future, this will have to be refactored.
				app.EmitWSEvents(finalizeBlockRes, txBz)
			}()

			// DEV_NOTE: There SHOULD ALWAYS be exactly one tx result so long as
			// we're finalizing one tx at a time (single tx blocks).
			txRes := finalizeBlockRes.GetTxResults()[0]

			bcastTxRes := coretypes.ResultBroadcastTx{
				Code:      txRes.GetCode(),
				Data:      txRes.GetData(),
				Log:       txRes.GetLog(),
				Codespace: txRes.GetCodespace(),
				Hash:      comettypes.Tx(txBz).Hash(),
			}

			response = rpctypes.NewRPCSuccessResponse(req.ID, bcastTxRes)
		default:
			response = rpctypes.NewRPCErrorResponse(req.ID, 500, "unsupported method", string(req.Params))
		}

		if err = json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// TODO_IN_THIS_COMMIT: godoc...
func writeErrorResponseFromErr(w http.ResponseWriter, req rpctypes.RPCRequest, err error) {
	var errMsg string
	if err != nil {
		errMsg = err.Error()
	}
	writeErrorResponse(w, req, errMsg, "")
}

// TODO_IN_THIS_COMMIT: godoc...
func writeErrorResponse(w http.ResponseWriter, req rpctypes.RPCRequest, msg, data string) {
	errRes := rpctypes.NewRPCErrorResponse(req.ID, 500, msg, data)
	if err := json.NewEncoder(w).Encode(errRes); err != nil {
		// TODO_IN_THIS_COMMIT: log error
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
