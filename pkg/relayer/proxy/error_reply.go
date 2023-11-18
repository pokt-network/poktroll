package proxy

import (
	"log"
	"net/http"

	"github.com/pokt-network/poktroll/pkg/partials"
	"github.com/pokt-network/poktroll/x/service/types"
)

// replyWithError builds a JSONRPCResponseError from the passed in error and writes it to the writer.
// TODO_TECHDEBT: This method should be aware of the request id and use it in the response by having
// the caller pass it along with the error if available.
// TODO_TECHDEBT: This method should be aware of the nature of the error to use the appropriate JSONRPC
// Code, Message and Data. Possibly by augmenting the passed in error with the adequate information.
// NOTE: This method is used to reply with an "internal" error that is related
// to the proxy itself and not to the relayed request.
func (jsrv *jsonRPCServer) replyWithError(payloadBz []byte, writer http.ResponseWriter, err error) {
	responseBz, err := partials.GetErrorReply(payloadBz, err)
	if err != nil {
		log.Printf("ERROR: failed getting error reply: %s", err)
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
