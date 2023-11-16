package proxy

import (
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
	relayResponse := &types.RelayResponse{
		Payload: &types.RelayResponse_JsonRpcPayload{
			JsonRpcPayload: &types.JSONRPCResponsePayload{
				// TODO_BLOCKER(@red-0ne): This MUST match the Id provided by the request.
				// If JSON-RPC request is not unmarshaled yet (i.e. can't extract ID), it SHOULD be a random ID.
				Id:      0,
				Jsonrpc: "2.0",
				Error: &types.JSONRPCResponseError{
					// Using conventional error code indicating internal server error.
					Code:    -32000,
					Message: err.Error(),
					Data:    nil,
				},
			},
		},
	}

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
