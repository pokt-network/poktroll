package e2e

import (
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/cometbft/cometbft/abci/types"
	cmtjson "github.com/cometbft/cometbft/libs/json"
	coretypes "github.com/cometbft/cometbft/rpc/core/types"
	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
	cosmoscodec "github.com/cosmos/cosmos-sdk/codec"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	gogogrpc "github.com/cosmos/gogoproto/grpc"
	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	"github.com/stretchr/testify/require"
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

// handleABCIQuery handles the actual ABCI query logic
func newHandleABCIQuery(t *testing.T, cdc cosmoscodec.Codec, client gogogrpc.ClientConn) runtime.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
		ctx := context.Background()

		fmt.Println(">>> handleABCIQuery called")
		//fmt.Printf("Method: %s, URL: %s\n", r.Method, r.URL.Path)

		// Read and log request body
		body, err := io.ReadAll(r.Body)
		if err != nil {
			//fmt.Printf("Error reading body: %v\n", err)
			http.Error(w, "Error reading request body", http.StatusBadRequest)
			return
		}
		defer r.Body.Close()
		//fmt.Printf("Request body: %s\n", string(body))

		// Parse JSON-RPC request
		var req rpctypes.RPCRequest
		if err := json.Unmarshal(body, &req); err != nil {
			fmt.Printf("Error unmarshaling request: %v\n", err)
			http.Error(w, "Invalid JSON request", http.StatusBadRequest)
			return
		}

		//fmt.Printf("RPC Method: %s\n", req.Method)
		//fmt.Printf("RPC ID: %v\n", req.ID)
		//fmt.Printf("RPC Params: %s\n", string(req.Params))

		params := make(map[string]json.RawMessage)
		err = json.Unmarshal(req.Params, &params)
		require.NoError(t, err)

		//t.Logf(">>> params: %+v", params)

		//abciRes, err := client.ABCIQueryWithOptions(ctx, path, data, opts)
		//require.NoError(t, err)

		//var args, reply any
		//var jsonRes json.RawMessage
		switch req.Method {
		// TODO_IN_THIS_COMMIT: extract...
		case "abci_query":
			// TODO_IN_THIS_COMMIT: add switch for different service/method handlers...

			pathRaw, hasPath := params["path"]
			require.True(t, hasPath)

			//abciQueryReq := new(cmtservice.ABCIQueryRequest)
			var path string
			err = json.Unmarshal(pathRaw, &path)
			require.NoError(t, err)

			dataRaw, hasData := params["data"]
			require.True(t, hasData)

			data, err := hex.DecodeString(string(bytes.Trim(dataRaw, `"`)))
			require.NoError(t, err)

			queryReq := new(authtypes.QueryAccountRequest)
			err = queryReq.Unmarshal(data)
			require.NoError(t, err)

			var height int64
			heightRaw, hasHeight := params["height"]
			if hasHeight {
				err = json.Unmarshal(bytes.Trim(heightRaw, `"`), &height)
				require.NoError(t, err)
			}

			//proveRaw, hasProve := params["prove"]
			//if hasProve {
			//	err = json.Unmarshal(proveRaw, &abciQueryReq.Prove)
			//	require.NoError(t, err)
			//}

			//abciQueryRes := new(cmtservice.ABCIQueryResponse)
			//err = client.Invoke(ctx, fmt.Sprintf("%s", abciQueryReq.Path), abciQueryReq, abciQueryRes)
			queryRes := new(authtypes.QueryAccountResponse)

			err = client.Invoke(ctx, path, queryReq, queryRes)
			require.NoError(t, err)

			//resData, err := cdc.MarshalJSON(queryRes)
			resData, err := queryRes.Marshal()
			require.NoError(t, err)

			abciQueryRes := coretypes.ResultABCIQuery{
				Response: types.ResponseQuery{
					//Code:      0,
					//Log:       "",
					//Info:      "",
					//Index:     0,
					//Key:       nil,
					Value: resData,
					//ProofOps:  nil,
					Height: height,
					//Codespace: "",
				},
			}
			//abciQueryRes := &cmtservice.ABCIQueryResponse{
			//	//Code:      0,
			//	//Log:       "",
			//	//Info:      "",
			//	//Index:     0,
			//	//Key:       nil,
			//	Value: resData,
			//	//ProofOps:  nil,
			//	Height: height,
			//	//Codespace: "",
			//}
			////jsonRes, err = json.Marshal(abciQueryRes)
			////jsonRes, err = cmtjson.Marshal(queryRes)
			//require.NoError(t, err)

			w.Header().Set("Content-Type", "application/json")
			//json.NewEncoder(w).Encode(abciQueryRes)
			//err = json.NewEncoder(w).Encode(queryRes)
			//jsonRes, err := cdc.MarshalJSON(queryRes)
			//jsonRes, err := cmtjson.Marshal(queryRes)

			jsonRes, err := cmtjson.Marshal(abciQueryRes)
			require.NoError(t, err)

			response := rpctypes.RPCResponse{
				JSONRPC: "2.0",
				ID:      req.ID,
				Result:  jsonRes,
			}

			//_, err = w.Write(response)
			err = json.NewEncoder(w).Encode(response)
			//err = json.NewEncoder(w).Encode(abciQueryRes)
			require.NoError(t, err)
			fmt.Println(">>> Response sent")
		case "broadcast_tx_sync":
			fmt.Println(">>>> BROADCAST_TX_SYNC")
			t.Fatalf("ERROR: unsupported method %s", req.Method)
		case "broadcast_tx_async":
			fmt.Println(">>>> BROADCAST_TX_ASYNC")
			t.Fatalf("ERROR: unsupported method %s", req.Method)
		default:
			t.Fatalf("ERROR: unsupported method %s", req.Method)
		}

		// Send response with nil result
		//response := rpctypes.NewRPCSuccessResponse(req.ID, nil)
		//response := rpctypes.RPCResponse{
		//	JSONRPC: "2.0",
		//	ID:      req.ID,
		//	Result:  jsonRes,
		//}

		//w.Header().Set("Content-Type", "application/json")
		//json.NewEncoder(w).Encode(response)
		//fmt.Println(">>> Response sent")
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
