package e2e

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	rpctypes "github.com/cometbft/cometbft/rpc/jsonrpc/types"
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
func handleABCIQuery(w http.ResponseWriter, r *http.Request, pathParams map[string]string) {
	fmt.Println(">>> handleABCIQuery called")
	fmt.Printf("Method: %s, URL: %s\n", r.Method, r.URL.Path)

	// Read and log request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		fmt.Printf("Error reading body: %v\n", err)
		http.Error(w, "Error reading request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()
	fmt.Printf("Request body: %s\n", string(body))

	// Parse JSON-RPC request
	var req rpctypes.RPCRequest
	if err := json.Unmarshal(body, &req); err != nil {
		fmt.Printf("Error unmarshaling request: %v\n", err)
		http.Error(w, "Invalid JSON request", http.StatusBadRequest)
		return
	}

	fmt.Printf("RPC Method: %s\n", req.Method)
	fmt.Printf("RPC ID: %v\n", req.ID)
	fmt.Printf("RPC Params: %s\n", string(req.Params))

	// Send response with nil result
	response := rpctypes.RPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  nil,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
	fmt.Println(">>> Response sent")
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
