package appgateserver

import (
	"log"
	"net/http"

	"github.com/pokt-network/poktroll/pkg/partials"
)

// replyWithError replies to the application with an error response.
// TODO_TECHDEBT: This method should be aware of the nature of the error to use the appropriate JSONRPC
// Code, Message and Data. Possibly by augmenting the passed in error with the adequate information.
// NOTE: This method is used to reply with an "internal" error that is related
// to the appgateserver itself and not to the relay request.
func (app *appGateServer) replyWithError(payloadBz []byte, writer http.ResponseWriter, err error) {
	responseBz, err := partials.GetErrorReply(payloadBz, err)
	if err != nil {
		log.Printf("ERROR: failed getting error reply: %s", err)
		return
	}

	if _, err = writer.Write(responseBz); err != nil {
		log.Printf("ERROR: failed writing relay response: %s", err)
		return
	}
}
