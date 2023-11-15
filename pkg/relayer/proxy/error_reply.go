package proxy

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/pokt-network/poktroll/x/service/types"
)

// replyWithError builds a JSONRPCResponseError from the passed in error and writes it to the writer.
// TODO_TECHDEBT: This method should be aware of the request id and use it in the response by having
// the caller pass it along with the error if available.
// TODO_TECHDEBT: This method should be aware of the nature of the error to use the appropriate JSONRPC
// Code, Message and Data. Possibly by augmenting the passed in error with the adequate information.
func (jsrv *jsonRPCServer) replyWithError(writer http.ResponseWriter, err error) {
	response := map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      0,
		"error": map[string]interface{}{
			"code":    -32000,
			"message": err.Error(),
			"data":    nil,
		},
	}
	responseBz, err := json.Marshal(response)
	if err != nil {
		log.Printf("ERROR: failed marshaling json error structure: %s", err)
		return
	}

	relayResponse := &types.RelayResponse{Payload: responseBz}

	relayResponseBz, err := relayResponse.Marshal()
	if err != nil {
		log.Printf("ERROR: failed marshaling relay response: %s", err)
		return
	}

	if _, err = writer.Write(relayResponseBz); err != nil {
		log.Printf("ERROR: failed writing relay response: %s", err)
		return
	}
}
