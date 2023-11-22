package proxy

import (
	"log"
	"net/http"

	"github.com/pokt-network/poktroll/pkg/partials"
	"github.com/pokt-network/poktroll/x/service/types"
)

// replyWithError builds the appropirate error format according to the payload
// using the passed in error and writes it to the writer.
// NOTE: This method is used to reply with an "internal" error that is related
// to the proxy itself and not to the relayed request.
func (sync *synchronousRPCServer) replyWithError(payloadBz []byte, writer http.ResponseWriter, err error) {
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
