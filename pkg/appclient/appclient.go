package appclient

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"strings"

	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	accounttypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	blocktypes "pocket/pkg/client"
	"pocket/x/service/types"
	sessiontypes "pocket/x/session/types"
)

type appClient struct {
	// keyName is the name of the key in the keyring that will be used to sign relay requests.
	keyName string
	keyring keyring.Keyring

	// clientCtx is the client context for the application.
	// It is used to query for the application's account to unmarshal the supplier's account
	// and get the public key to verify the relay response signature.
	clientCtx sdkclient.Context

	// appAddress is the address of the application that this app client is for.
	appAddress string

	// sessionQuerier is the querier for the session module.
	// It used to get the current session for the application given a requested service.
	sessionQuerier sessiontypes.QueryClient

	// accountQuerier is the querier for the account module.
	// It is used to get the the supplier's public key to verify the relay response signature.
	accountQuerier accounttypes.QueryClient

	// blockClient is the client for the block module.
	// It is used to get the current block height to query for the current session.
	blockClient blocktypes.BlockClient

	// server is the HTTP server that will be used capture application requests
	// so that they can be signed and relayed to the supplier.
	server *http.Server
}

func NewAppClient(
	clientCtx sdkclient.Context,
	keyName string,
	keyring keyring.Keyring,
	applicationEndpoint *url.URL,
	blockClient blocktypes.BlockClient,
) *appClient {
	sessionQuerier := sessiontypes.NewQueryClient(clientCtx)
	accountQuerier := accounttypes.NewQueryClient(clientCtx)

	return &appClient{
		clientCtx:      clientCtx,
		keyName:        keyName,
		keyring:        keyring,
		sessionQuerier: sessionQuerier,
		accountQuerier: accountQuerier,
		blockClient:    blockClient,
		server:         &http.Server{Addr: applicationEndpoint.Host},
	}
}

// Start starts the application server and blocks until the context is done
// or the server returns an error.
func (app *appClient) Start(ctx context.Context) error {
	// Get and populate the application address from the keyring.
	keyRecord, err := app.keyring.Key(app.keyName)
	if err != nil {
		return err
	}

	accAddress, err := keyRecord.GetAddress()
	if err != nil {
		return err
	}

	app.appAddress = accAddress.String()

	// Shutdown the HTTP server when the context is done.
	go func() {
		<-ctx.Done()
		app.server.Shutdown(ctx)
	}()

	// Start the HTTP server.
	return app.server.ListenAndServe()
}

// Stop stops the application server and returns any error that occurred.
func (app *appClient) Stop(ctx context.Context) error {
	return app.server.Shutdown(ctx)
}

// ServeHTTP is the HTTP handler for the application server.
// It captures the application request, signs it, and sends it to the supplier.
// After receiving the response from the supplier, it verifies the response signature
// before returning the response to the application.
// The serviceId is extracted from the request path.
func (app *appClient) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	ctx := request.Context()

	// Extract the serviceId from the request path.
	path := request.URL.Path
	serviceId := strings.Split(path, "/")[1]

	// Currently only JSON RPC requests are supported.
	relayReponse, err := app.handleJSONRPCRelays(ctx, serviceId, request)
	if err != nil {
		// Reply with an error response if there was an error handling the relay.
		app.replyWithError(writer, err)
		log.Printf("ERROR: failed handling relay: %s", err)
		return
	}

	// Reply with the RelayResponse payload.
	if _, err := writer.Write(relayReponse); err != nil {
		app.replyWithError(writer, err)
		return
	}
	log.Print("INFO: request serviced successfully")
}

// replyWithError replies to the application with an error response.
// TODO_TECHDEBT: This method should be aware of the nature of the error to use the appropriate JSONRPC
// Code, Message and Data. Possibly by augmenting the passed in error with the adequate information.
func (app *appClient) replyWithError(writer http.ResponseWriter, err error) {
	relayResponse := &types.RelayResponse{
		Payload: &types.RelayResponse_JsonRpcPayload{
			JsonRpcPayload: &types.JSONRPCResponsePayload{
				Id:      make([]byte, 0),
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
