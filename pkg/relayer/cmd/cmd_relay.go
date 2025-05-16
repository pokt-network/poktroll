// Package cmd contains the relayminer CLI commands and utilities.
package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"

	"github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/pokt-network/shannon-sdk"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"github.com/spf13/cobra"

	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

// TODO_IMPROVE(@olshansk): Add more configurations & flags to make testing easier and more extensible:
// --dry-run avoid sending the relay
// --dont-validate to avoid requiring a valid signature
// --bypass-session to avoid requiring a valid session and going straight to the supplier

var (
	// Custom flags for 'pocketd relayminer relay' subcommand
	flagRelayApp                       string // Application address
	flagRelaySupplier                  string // Supplier address
	flagRelayPayload                   string // Relay payload
	flagSupplierPublicEndpointOverride string // Optional endpoint override

	// Cosmos flags for 'pocketd relayminer relay' subcommand
	flagNodeGRPCURLRelay      string
	flagNodeGRPCInsecureRelay bool

	// TODO_TECHDEBT(@olshansk): Reconsider the need for this flag.
	// This flag can theoretically be avoided because it is only used to get the height of the latest block for session generation.
	// Passing `0` as the block height defaults to the latest height.
	// We are keeping it to use this file as an example of an end-to-end system that leverages the shannon-sdk for example purposes.
	flagNodeRPCURLRelay string
)

// relayCmd defines the `relay` subcommand for sending a relay as an application.
//
// - Sends a test relay to a Supplier's RelayMiner from a staked Application
// - Useful for local testing, debugging, and verifying Supplier setup
// - See TODO_IMPROVE for planned enhancements
func relayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay --app <app> --supplier <supplier> --service-id <service-id> --payload <payload> [--supplier-public-endpoint-override <url>]",
		Short: "Send a relay as an application to a particular supplier",
		Long: `Send a test relay to a Supplier's RelayMiner from a staked Application.

RelayMiner relays simulate real-world requests and responses between a staked Application and a Supplier.

Key actions performed:
- Sends a JSON-RPC relay from a staked Application to a Supplier
- Signs the relay using the Application's private key
- Validates the Supplier's response and signature
- Prints the backend response and relay status

Callouts:
- Make sure both the Application and Supplier are staked before running relays.
- Use the '--supplier-public-endpoint-override' flag to test against a local endpoint.

For more info, run 'relay --help'.`,
		Example: `
  ./pocketd relayminer relay \
    --app=pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 \
    --supplier=pokt19a3t4yunp0dlpfjrp7qwnzwlrzd5fzs2gjaaaj \
    --service-id=anvil \
    --node=tcp://127.0.0.1:26657 \
    --grpc-addr=localhost:9090 \
    --grpc-insecure=true \
    --payload="{\"jsonrpc\": \"2.0\", \"id\": 1, \"method\": \"eth_blockNumber\", \"params\": []}" \
    --supplier-public-endpoint-override=http://localhost:8085`,
		RunE: runRelay,
	}

	// Cosmos flags
	cmd.Flags().StringVar(&flagNodeRPCURLRelay, cosmosflags.FlagNode, "tcp://127.0.0.1:26657", "Cosmos node RPC URL (defaults to LocalNet)")
	cmd.Flags().StringVar(&flagNodeGRPCURLRelay, cosmosflags.FlagGRPC, "localhost:9090", "Cosmos node GRPC URL (defaults to LocalNet)")
	cmd.Flags().BoolVar(&flagNodeGRPCInsecureRelay, cosmosflags.FlagGRPCInsecure, true, "Used to initialize the Cosmos query context with grpc security options (defaults to true for LocalNet)")

	// Custom Flags
	cmd.Flags().StringVar(&flagRelayApp, "app", "", "(Required) Staked application address")
	cmd.Flags().StringVar(&flagRelaySupplier, "supplier", "", "(Required) Staked Supplier address")
	cmd.Flags().StringVar(&flagRelayPayload, "payload", "", "(Required) JSON-RPC payload")
	cmd.Flags().StringVar(&flagSupplierPublicEndpointOverride, "supplier-public-endpoint-override", "http://localhost:8085", "(Optional) Override the publicly exposed endpoint of the Supplier (useful for LocalNet testing)")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("supplier")
	_ = cmd.MarkFlagRequired("payload")

	return cmd
}

// runRelay executes the relay command logic.
//
// Steps:
// - Initializes gRPC connection
// - Fetches node status and application details
// - Builds application ring for signing
// - Gets latest block height and current session
// - Selects the correct endpoint for the supplier
// - Optionally overrides endpoint URL for local testing
// - Sends a relay request and prints results
//
// Returns error if any step fails.
func runRelay(cmd *cobra.Command, args []string) error {
	ctx, cancelCtx := context.WithCancel(cmd.Context())
	defer cancelCtx() // Ensure context cancellation

	// Set up logger options
	// TODO_TECHDEBT: Populate logger from config (ideally, from viper).
	loggerOpts := []polylog.LoggerOption{
		polyzero.WithLevel(polyzero.ParseLevel(flagLogLevel)),
		polyzero.WithOutput(os.Stderr),
	}

	// Construct logger and associate with command context
	logger := polyzero.NewLogger(loggerOpts...)
	ctx = logger.WithContext(ctx)
	cmd.SetContext(ctx)

	logger.Info().Msgf("About to send a relay to %s for app %s", flagRelaySupplier, flagRelayApp)

	// Initialize gRPC connection
	grpcConn, err := connectGRPC(GRPCConfig{
		HostPort: flagNodeGRPCURLRelay,
		Insecure: flagNodeGRPCInsecureRelay,
	})
	if err != nil {
		return err
	}
	defer grpcConn.Close()
	logger.Info().Msg("✅ gRPC connection initialized")

	// Create a connection to the POKT full node
	nodeStatusFetcher, err := sdk.NewPoktNodeStatusFetcher(flagNodeRPCURLRelay)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error fetching block height")
		return err
	}
	logger.Info().Msg("✅ Node status fetcher initialized")

	// Create an account client for fetching public keys
	accountClient := sdk.AccountClient{
		PoktNodeAccountFetcher: sdk.NewPoktNodeAccountFetcher(grpcConn),
	}
	logger.Info().Msg("✅ Account client initialized")

	// Create an application client to get application details
	appClient := sdk.ApplicationClient{
		QueryClient: apptypes.NewQueryClient(grpcConn),
	}
	app, err := appClient.GetApplication(ctx, flagRelayApp)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error fetching application")
		return err
	}
	logger.Info().Msgf("✅ Application fetched: %v", app)

	// Applications must have exactly can only have one service config
	if len(app.ServiceConfigs) != 1 {
		logger.Error().Msgf("❌ Application %s must have exactly one service config", app.Address)
		return errors.New("application must have exactly one service config")
	}
	serviceId := app.ServiceConfigs[0].ServiceId
	logger.Info().Msgf("✅ Service ID retrieved: %s", serviceId)

	// Create an application ring for signing
	ring := sdk.ApplicationRing{
		Application:      app,
		PublicKeyFetcher: &accountClient,
	}
	logger.Info().Msg("✅ Application ring created")

	// Get the latest block height
	blockClient := sdk.BlockClient{
		PoktNodeStatusFetcher: nodeStatusFetcher,
	}
	blockHeight, err := blockClient.LatestBlockHeight(ctx)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error fetching block height")
		return err
	}
	logger.Info().Msgf("✅ Block height retrieved: %d", blockHeight)

	// Get the current session
	sessionClient := sdk.SessionClient{
		PoktNodeSessionFetcher: sdk.NewPoktNodeSessionFetcher(grpcConn),
	}
	session, err := sessionClient.GetSession(
		ctx,
		app.Address,
		serviceId,
		blockHeight,
	)
	if err != nil {
		logger.Error().Err(err).Msgf("❌ Error fetching session for app %s and service ID %s", app.Address, serviceId)
		return err
	}
	logger.Info().Msgf("✅ Session fetched: %v", session)

	// Select an endpoint from the session
	sessionFilter := sdk.SessionFilter{
		Session:         session,
		EndpointFilters: []sdk.EndpointFilter{},
	}
	endpoints, err := sessionFilter.FilteredEndpoints()
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error filtering endpoints")
		return err
	}
	if len(endpoints) == 0 {
		logger.Error().Msg("❌ No endpoints available")
		return err
	}
	logger.Info().Msgf("✅ Endpoints fetched: %v", endpoints)

	var endpoint sdk.Endpoint
	for _, e := range endpoints {
		if string(e.Supplier()) == flagRelaySupplier {
			endpoint = e
			break
		}
	}
	if endpoint == nil {
		logger.Error().Msgf("❌ No endpoint found for supplier %s in the current session", flagRelaySupplier)
		return err
	}
	logger.Info().Msgf("✅ Endpoint selected: %v", endpoint)

	// Get the endpoint URL
	endpointUrl := endpoint.Endpoint().Url
	// Override the endpoint URL if specified
	if flagSupplierPublicEndpointOverride != "" {
		endpointUrl = flagSupplierPublicEndpointOverride
		logger.Warn().Msgf("⚠️ Using override endpoint URL: %s", endpointUrl)
	}

	// Prepare the JSON-RPC request payload
	body := io.NopCloser(bytes.NewReader([]byte(flagRelayPayload)))
	jsonRpcServiceReq, err := http.NewRequest(http.MethodPost, endpointUrl, body)
	if err != nil {
		return fmt.Errorf("failed to create a new HTTP request for url %s: %w", endpointUrl, err)
	}
	jsonRpcServiceReq.Header.Set("Content-Type", "application/json")
	_, payloadBz, err := sdktypes.SerializeHTTPRequest(jsonRpcServiceReq)
	if err != nil {
		return fmt.Errorf("failed to Serialize HTTP Request for URL %s: %w", endpointUrl, err)
	}
	logger.Info().Msg("✅ JSON-RPC request payload serialized.")

	// Build a relay request
	relayReq, err := sdk.BuildRelayRequest(endpoint, payloadBz)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error building relay request")
		return err
	}
	logger.Info().Msg("✅ Relay request built.")

	// TODO_TECHDEBT(@olshansk): Retrieve the passphrase from the keyring.
	// The initial version of this assumes the keyring is unlocked.
	passphrase := ""

	// Sign the relay request
	clientCtx := client.GetClientContextFromCmd(cmd)
	appPrivateKeyHex, err := getPrivateKeyHexFromKeyring(clientCtx.Keyring, app.Address, passphrase)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error getting private key")
		return err
	}
	logger.Info().Msgf("✅ Retrieved private key for app %s", app.Address)
	appSigner := sdk.Signer{PrivateKeyHex: appPrivateKeyHex}
	signedRelayReq, err := appSigner.Sign(ctx, relayReq, ring)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error signing relay request")
		return err
	}
	logger.Info().Msg("✅ Relay request signed.")

	// Marshal the signed relay request
	relayReqBz, err := signedRelayReq.Marshal()
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error marshaling relay request")
		return err
	}
	logger.Info().Msg("✅ Relay request marshaled.")

	// Parse the endpoint URL
	reqUrl, err := url.Parse(endpointUrl)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error parsing endpoint URL")
		return err
	}
	logger.Info().Msgf("✅ Endpoint URL parsed: %v", reqUrl)

	// Create the HTTP request with the relay request body
	httpReq := &http.Request{
		Method: http.MethodPost,
		URL:    reqUrl,
		Body:   io.NopCloser(bytes.NewReader(relayReqBz)),
	}

	// Send the request HTTP request containing the signed relay request
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error sending relay request")
		return err
	}
	defer httpResp.Body.Close()

	// Read the response
	respBz, err := io.ReadAll(httpResp.Body)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error reading response")
		return err
	}
	logger.Info().Msgf("✅ Response read %d bytes", len(respBz))

	// Ensure the supplier operator signature is present
	supplierSignerAddress := signedRelayReq.Meta.SupplierOperatorAddress
	if supplierSignerAddress == "" {
		logger.Error().Msg("❌ Supplier operator signature is missing")
		return errors.New("Relay response missing supplier operator signature")
	}

	// Ensure the supplier operator address matches the expected address
	if supplierSignerAddress != flagRelaySupplier {
		logger.Error().Msgf("❌ Supplier operator address %s does not match the expected address %s", supplierSignerAddress, flagRelaySupplier)
		return errors.New("Relay response supplier operator signature does not match")
	}

	// Validate the relay response
	relayResp, err := sdk.ValidateRelayResponse(
		ctx,
		sdk.SupplierAddress(supplierSignerAddress),
		respBz,
		&accountClient,
	)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error validating response")
		return err
	}
	// Deserialize the relay response
	backendHttpResponse, err := sdktypes.DeserializeHTTPResponse(relayResp.Payload)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error deserializing response payload")
		return err
	}
	logger.Info().Msgf("✅ Backend response status code: %v", backendHttpResponse.StatusCode)

	var jsonMap map[string]interface{}
	// Unmarshal the HTTP response body into jsonMap
	if err := json.Unmarshal(backendHttpResponse.BodyBz, &jsonMap); err != nil {
		logger.Error().Err(err).Msg("❌ Error deserializing response payload")
		return err
	}
	logger.Info().Msgf("✅ Deserialized response body as JSON map: %+v", jsonMap)

	// If "jsonrpc" key exists, try to further deserialize "result"
	if _, ok := jsonMap["jsonrpc"]; ok {
		resultRaw, exists := jsonMap["result"]
		if exists {
			switch v := resultRaw.(type) {
			case map[string]interface{}:
				logger.Info().Msgf("✅ Further deserialized 'result' (object): %+v", v)
			case []interface{}:
				logger.Info().Msgf("✅ Further deserialized 'result' (array): %+v", v)
			case string:
				logger.Info().Msgf("✅ Further deserialized 'result' (string): %s", v)
			case float64, bool, nil:
				logger.Info().Msgf("✅ Further deserialized 'result' (primitive): %+v", v)
			default:
				logger.Warn().Msgf("⚠️ 'result' is of an unhandled type: %T, value: %+v", v, v)
			}
		}
	}

	return nil
}
