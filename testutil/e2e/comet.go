package e2e

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/cometbft/cometbft/abci/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
)

//func handleABCIQuery(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
//	// Only accept POST method
//	if r.Method != http.MethodPost {
//		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
//		return
//	}
//
//	// Read the request body
//	body, err := io.ReadAll(r.Body)
//	if err != nil {
//		http.Error(w, "Error reading request body", http.StatusBadRequest)
//		return
//	}
//	defer r.Body.Close()
//
//	// Parse the JSON-RPC request
//	var req comettypes.RPCRequest
//	if err := json.Unmarshal(body, &req); err != nil {
//		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
//		return
//	}
//
//	// Verify method
//	if req.Method != "abci_query" {
//		fmt.Printf(">>>> WRONG METHOD")
//
//		// TODO_IN_THIS_COMMIT: consolidate with other error response logic...
//		res := comettypes.RPCInvalidRequestError(req.ID, fmt.Errorf("Method %s not supported", req.Method))
//		json.NewEncoder(w).Encode(res)
//
//		return
//	}
//
//	// Process the ABCI query
//	// This is where you'd implement the actual ABCI query logic
//	result := processABCIQuery(req.Params)
//
//	// Send response
//	response := comettypes.RPCResponse{
//		JSONRPC: "2.0",
//		ID:      req.ID,
//		Result:  result,
//	}
//
//	w.Header().Set("Content-Type", "application/json")
//	json.NewEncoder(w).Encode(response)
//}

const (
	authAccountQuery = "/cosmos.auth.v1beta1.Query/Account"
)

// handleABCIQuery handles the actual ABCI query logic
func newPostHandler(client gogogrpc.ClientConn) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		ctx := context.Background()

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
		switch req.Method {
		// TODO_IN_THIS_COMMIT: extract...
		case "abci_query":
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

			switch path {
			case authAccountQuery:
				//abciQueryReq := new(cmtservice.ABCIQueryRequest)

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

			w.Header().Set("Content-Type", "application/json")

			jsonRes, err := cmtjson.Marshal(abciQueryRes)
			if err != nil {
				writeErrorResponseFromErr(w, req, err)
				return
			}

			response = rpctypes.RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  jsonRes,
			}
			fmt.Println(">>> Response sent")
		case "broadcast_tx_sync":
			fmt.Println(">>>> BROADCAST_TX_SYNC")
			response = rpctypes.NewRPCErrorResponse(req.ID, 500, "unsupported method", string(req.Params))
		case "broadcast_tx_async":
			fmt.Println(">>>> BROADCAST_TX_ASYNC")
			response = rpctypes.NewRPCErrorResponse(req.ID, 500, "unsupported method", string(req.Params))
		default:
			response = rpctypes.NewRPCErrorResponse(req.ID, 500, "unsupported method", string(req.Params))
		}

		if err = json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

// processABCIQuery handles the actual ABCI query logic
func processABCIQuery(params json.RawMessage) json.RawMessage {
	// Implement your ABCI query processing logic here
	// This would typically involve:
	// 1. Decoding the hex data
	// 2. Parsing the height
	// 3. Making the actual ABCI query to your blockchain node
	// 4. Processing the response
	// 5. Returning the result in the expected format

	fmt.Println(">>>> ABCI_QUERY")

	// For now, returning a placeholder
	//return map[string]interface{}{
	//	"height": height,
	//	"result": map[string]interface{}{
	//		// Add your actual query result structure here
	//		"code":   0,
	//		"log":    "",
	//		"height": height,
	//		"value":  "", // Base64-encoded response value
	//	},
	//}
	return nil
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
