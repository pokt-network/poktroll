// Package cmd contains the relayminer CLI commands and utilities.
package cmd

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"time"
	"strings"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/pokt-network/shannon-sdk"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer/proxy"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// TODO_IMPROVE(@olshansk): Add more configurations & flags to make testing easier and more extensible:
// --dry-run avoid sending the relay
// --dont-validate to avoid requiring a valid signature
// --bypass-session to avoid requiring a valid session and going straight to the supplier
//
// TODO_IMPROVE: Add support for REST and WebSocket relays in pocketd relayminer relay
var (
	// Custom flags for 'pocketd relayminer relay' subcommand
	flagRelayApp                       string // Application address
	flagRelaySupplier                  string // Supplier address
	flagRelayPayload                   string // Relay payload
	flagSupplierPublicEndpointOverride string // Optional endpoint override
	flagRelayRequestCount              int    // Number of requests to send
)

// relayCmd defines the `relay` subcommand for sending a relay as an application.
//
// - Sends a test relay to a Supplier's RelayMiner from a staked Application
// - Useful for local testing, debugging, and verifying Supplier setup
// - See TODO_IMPROVE for planned enhancements
func relayCmd() *cobra.Command {
	cmdRelay := &cobra.Command{
		Use:   "relay --app <app> --supplier <supplier> --payload <payload> [--supplier-public-endpoint-override <url>]",
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

  # LocalNet example with an endpoint override
  $ pocketd relayminer relay \
    --app=pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4 \
    --supplier=pokt19a3t4yunp0dlpfjrp7qwnzwlrzd5fzs2gjaaaj \
    --node=tcp://127.0.0.1:26657 \
    --grpc-addr=localhost:9090 \
    --grpc-insecure=true \
    --payload="{\"jsonrpc\": \"2.0\", \"id\": 1, \"method\": \"eth_blockNumber\", \"params\": []}" \
    --supplier-public-endpoint-override=http://localhost:8085

  # Beta example for a real service
  pocketd relayminer relay \
	--app=pokt12fj3xlqg6d20fl4ynuejfqd3fkqmq25rs3yf7g \
	--supplier=pokt1hwed7rlkh52v6u952lx2j6y8k9cn5ahravmzfa \
	--node=https://shannon-testnet-grove-rpc.beta.poktroll.com \
	--grpc-addr=shannon-testnet-grove-grpc.beta.poktroll.com:443 \
	--payload="{\"jsonrpc\": \"2.0\", \"id\": 1, \"method\": \"eth_blockNumber\", \"params\": []}"
`,
		RunE: runRelay,
	}

	// Custom Flags
	cmdRelay.Flags().StringVar(&flagRelayApp, FlagApp, DefaultFlagApp, FlagAppUsage)
	cmdRelay.Flags().StringVar(&flagRelayPayload, FlagPayload, DefaultFlagPayload, FlagPayloadUsage)
	cmdRelay.Flags().StringVar(&flagRelaySupplier, FlagSupplier, DefaultFlagSupplier, FlagSupplierUsage)
	cmdRelay.Flags().StringVar(
		&flagSupplierPublicEndpointOverride,
		FlagSupplierPublicEndpointOverride,
		DefaultFlagSupplierPublicEndpointOverride,
		FlagSupplierPublicEndpointOverrideUsage,
	)
	cmdRelay.Flags().IntVar(&flagRelayRequestCount, FlagCount, DefaultFlagCount, FlagCountUsage)

	// Required cosmos-sdk CLI query flags.
	cmdRelay.Flags().String(cosmosflags.FlagGRPC, flags.OmittedDefaultFlagValue, flags.FlagGRPCUsage)
	cmdRelay.Flags().Bool(cosmosflags.FlagGRPCInsecure, true, flags.FlagGRPCInsecureUsage)

	// This command depends on the conventional cosmos-sdk CLI tx flags.
	cosmosflags.AddTxFlagsToCmd(cmdRelay)

	// Required flags
	_ = cmdRelay.MarkFlagRequired(FlagApp)
	_ = cmdRelay.MarkFlagRequired(FlagPayload)

	return cmdRelay
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

	logLevel, err := flags.GetFlagValueString(cmd, cosmosflags.FlagLogLevel)
	if err != nil {
		return err
	}

	nodeRPCURL, err := flags.GetFlagValueString(cmd, cosmosflags.FlagNode)
	if err != nil {
		return err
	}

	nodeGRPCURL, err := flags.GetFlagValueString(cmd, cosmosflags.FlagGRPC)
	if err != nil {
		return err
	}

	nodeGRPCInsecure, err := flags.GetFlagBool(cmd, cosmosflags.FlagGRPCInsecure)
	if err != nil {
		return err
	}

	// Set up logger options
	// TODO_TECHDEBT: Populate logger from config (ideally, from viper).
	loggerOpts := []polylog.LoggerOption{
		polyzero.WithLevel(polyzero.ParseLevel(logLevel)),
		polyzero.WithOutput(os.Stderr),
		polyzero.WithTimestamp(),
	}

	// Construct logger and associate with command context
	logger := polyzero.NewLogger(loggerOpts...)
	ctx = logger.WithContext(ctx)
	cmd.SetContext(ctx)

	logger.Info().Msgf("About to send %d relay(s) to supplier '%s' for app '%s'", flagRelayRequestCount, flagRelaySupplier, flagRelayApp)

	// Initialize gRPC connection
	grpcConn, err := connectGRPC(GRPCConfig{
		HostPort: nodeGRPCURL,
		Insecure: nodeGRPCInsecure,
	})
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error connecting to gRPC")
		return err
	}
	defer func(grpcConn *grpc.ClientConn) {
		err = grpcConn.Close()
		if err != nil {
			logger.Error().Err(err).Msg("❌ Error closing gRPC connection")
		}
	}(grpcConn)
	logger.Info().Msgf("✅ gRPC connection initialized: %v", grpcConn)

	// Create a connection to the POKT full node
	nodeStatusFetcher, err := sdk.NewPoktNodeStatusFetcher(nodeRPCURL)
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
	logger.Info().Msg("✅ Application client initialized")

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
	logger.Info().Msgf("✅ Service identified: '%s'", serviceId)

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
	logger.Info().Msgf("✅ Session with id %s at height 	%d fetched for app %s and service ID %s with %d suppliers", session.SessionId, blockHeight, app.Address, serviceId, len(session.Suppliers))

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
	logger.Info().Msgf("✅ %d endpoints fetched", len(endpoints))

	var endpoint sdk.Endpoint
	if flagRelaySupplier != "" {
		logger.Info().Msgf("✅ Supplier specified: '%s'", flagRelaySupplier)
		for _, e := range endpoints {
			if string(e.Supplier()) == flagRelaySupplier {
				endpoint = e
				logger.Info().Msgf("✅ Endpoint for supplier '%s' selected: %v", flagRelaySupplier, endpoint)
				break
			}
		}
		if endpoint == nil {
			logger.Error().Msgf("❌ No endpoint found for supplier %s in the current session", flagRelaySupplier)
			return err
		}
		// TODO_UPNEXT(@olshansk): Add support for sending a relay to a supplier that is not in the session.
		// endpoint, err = querySupplier(logger, grpcConn, ctx, serviceId, flagRelaySupplier)
		// if err != nil {
		// 	logger.Error().Err(err).Msg("❌ No endpoint found and could not fetch supplier directly")
		// 	return err
		// }
		// logger.Info().Msgf("✅ Supplier %s fetched successfully and using endpoint %v", flagRelaySupplier, endpoint)
	} else {
		endpoint = endpoints[rand.Intn(len(endpoints))]
		logger.Info().Msgf("✅ No supplier specified, randomly selected endpoint: %v", endpoint)
	}

	// Get the endpoint URL
	endpointUrl := endpoint.Endpoint().Url
	// Override the endpoint URL if specified
	if flagSupplierPublicEndpointOverride != "" {
		endpointUrl = flagSupplierPublicEndpointOverride
		logger.Warn().Msgf("⚠️ Using override endpoint URL: %s", endpointUrl)
	}

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

	// Parse the endpoint URL
	reqUrl, err := url.Parse(endpointUrl)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error parsing endpoint URL")
		return err
	}
	logger.Info().Msgf("✅ Endpoint URL parsed: %v", reqUrl)

	// Send multiple requests sequentially as specified by the count flag
	for i := 1; i <= flagRelayRequestCount; i++ {
		if flagRelayRequestCount > 1 {
			logger.Info().Msgf("📤 Sending request %d of %d", i, flagRelayRequestCount)
		}

		beforeRequestPreparationTime := time.Now()

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

		requestBuildingDuration := time.Since(beforeRequestPreparationTime)
		logger.Info().Msgf("⏱️ Request building duration: %s", requestBuildingDuration)

		beforeRequestSigningTime := time.Now()

		signedRelayReq, err := appSigner.Sign(ctx, relayReq, ring)
		if err != nil {
			logger.Error().Err(err).Msg("❌ Error signing relay request")
			return err
		}
		logger.Info().Msg("✅ Relay request signed.")

		requestSigningDuration := time.Since(beforeRequestSigningTime)
		logger.Info().Msgf("⏱️ Request signing duration: %s", requestSigningDuration)

		beforeRequestMarshallingTime := time.Now()

		// Marshal the signed relay request
		relayReqBz, err := signedRelayReq.Marshal()
		if err != nil {
			logger.Error().Err(err).Msg("❌ Error marshaling relay request")
			return err
		}
		logger.Info().Msg("✅ Relay request marshaled.")

		requestMarshallingDuration := time.Since(beforeRequestMarshallingTime)
		logger.Info().Msgf("⏱️ Request marshalling duration: %s", requestMarshallingDuration)

		beforeRequestSendingTime := time.Now()

	// Parse the endpoint URL
	reqUrl, err := url.Parse(endpointUrl)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error parsing endpoint URL")
		return err
	}
	logger.Info().Msgf("✅ Endpoint URL parsed: %v", reqUrl)

	// Create http client
	backendClient := &http.Client{
		Timeout: 600 * time.Second,
	}

	ctxWithTimeout, cancelFn := context.WithTimeout(context.Background(), 600*time.Second)
	defer cancelFn()

	// Send multiple requests sequentially as specified by the count flag
	for i := 1; i <= flagRelayRequestCount; i++ {

		// Create the HTTP request with the relay request body
		httpReq, err := http.NewRequestWithContext(
			ctxWithTimeout,
			http.MethodPost, // This is the method to the Relay Miner node
			reqUrl.String(),
			io.NopCloser(bytes.NewReader(relayReqBz)),
		)
		if err != nil {
			logger.Error().Err(err).Msg("❌ Error creating relay request")
			continue
		}

		if httpResp.StatusCode != http.StatusOK {
			logger.Error().Err(err).Msgf("❌ Error sending relay request %d due to response status code %d", i, httpResp.StatusCode)
			continue
		}

		requestSendingDuration := time.Since(beforeRequestSendingTime)
		logger.Info().Msgf("⏱️ Request sending duration: %s", requestSendingDuration)

		beforeResponseReadTime := time.Now()

		// Read the response
		respBz, err := io.ReadAll(httpResp.Body)
		if err != nil {
			logger.Error().Err(err).Msg("❌ Error sending relay request")
			continue
		}

		// Ensure the supplier operator signature is present
		supplierSignerAddress := signedRelayReq.Meta.SupplierOperatorAddress
		if supplierSignerAddress == "" {
			logger.Error().Msg("❌ Supplier operator signature is missing")
			proxy.CloseBody(logger, httpResp.Body)
			continue
		}
		// Ensure the supplier operator address matches the expected address
		if flagRelaySupplier == "" {
			logger.Warn().Msg("⚠️ Supplier operator address not specified, skipping signature check")
		} else if supplierSignerAddress != flagRelaySupplier {
			logger.Error().Msgf("❌ Supplier operator address %s does not match the expected address %s", supplierSignerAddress, flagRelaySupplier)
			proxy.CloseBody(logger, httpResp.Body)
			continue
		}

		logger.Info().Msgf("🔍 Backend response header, Content-Type: %s", httpResp.Header.Get("Content-Type"))

		// Handle response according to type
		if proxy.IsStreamingResponse(httpResp) {
			streamErr := processStreamRequest(ctx, httpResp, supplierSignerAddress, accountClient, logger)
			proxy.CloseBody(logger, httpResp.Body)
			if streamErr != nil {
				logger.Error().Err(streamErr).Msg("❌ Stream errored")
			}
		} else {
			// Normal, non-streaming request
			reqErr := processNormalRequest(ctx, httpResp, supplierSignerAddress, accountClient, logger)
			proxy.CloseBody(logger, httpResp.Body)
			if reqErr != nil {
				logger.Error().Err(reqErr).Msg("❌ Request errored")
			}
		}

		// This is intentionally not a defer because the loop could introduce memory leaks,
		// performance issues and bad connection management for high flagRelayRequestCount values
		proxy.CloseBody(logger, httpResp.Body)
	}

	return nil
}

// Handles the Pocket Network stream response from a Relay Miner.
//
// This functions uses an scanner that chunks the incomming response using the
// defined split function.
// Then it checks if the chunk is correctly signed, and tries to unmarshal it
// if the stream is of type SSE.
func processStreamRequest(ctx context.Context,
	httpResp *http.Response,
	supplierSignerAddress string,
	accountClient sdk.AccountClient,
	logger polylog.Logger) error {
	logger.Info().Msgf("🌊 Handling streaming response with status:")

	// Check if this is SSE (used below, if this is SSE we will unmarshal)
	isSSE := strings.Contains(strings.ToLower(httpResp.Header.Get("Content-Type")), "text/event-stream")
	if isSSE {
		logger.Info().Msgf("🔍 Detected SSE stream, we will try to unmarshal.")
	}

	// Start handling the body chunks
	scanner := bufio.NewScanner(httpResp.Body)
	// Assign the custom stream splitter
	scanner.Split(proxy.ScanEvents)
	// Scan
	for scanner.Scan() {
		// Get chunck
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		logger.Info().Msgf("📦 Read chunk of length %d", len(line))
		// Check and retrieve backend chunk
		backendHttpResponse, err := checkAndGetBackendResponse(ctx, supplierSignerAddress, line, accountClient, logger)
		if err != nil {
			return err
		}

		// get string body
		stringBody := string(backendHttpResponse.BodyBz)

		if !isSSE {
			// Just print content and continue
			logger.Info().Msgf("Chunk String Content: %s", stringBody)
			continue
		}

		responseReadDuration := time.Since(beforeResponseReadTime)
		logger.Info().Msgf("⏱️ Response building duration: %s", responseReadDuration)

		beforeResponseVerificationTime := time.Now()

		// This is SSE, unmarshal
		trimmedPrefix := strings.TrimPrefix(stringBody, "data: ")
		stringJson := strings.TrimSuffix(trimmedPrefix, "\n")
		if len(stringJson) == 0 {
			// this was probably a delimiter
			continue
		} else if stringJson == "[DONE]" {
			// SSE end
			logger.Info().Msgf("✅ SSE Done")
		} else {
			// Umarshal
			err = unmarshalAndPrintResponse([]byte(stringJson), logger)
			if err != nil {
				logger.Info().Msgf("Received: %s | Stripped: %s", stringBody, stringJson)
				return err
			}
		}
	}
	return nil
}

func processNormalRequest(ctx context.Context,
	httpResp *http.Response,
	supplierSignerAddress string,
	accountClient sdk.AccountClient,
	logger polylog.Logger) error {
	// Read the response
	respBz, err := io.ReadAll(httpResp.Body)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error reading response")
		return err
	}
	logger.Info().Msgf("✅ Response read %d bytes", len(respBz))

	// Check signature and get backend response
	backendHttpResponse, err := checkAndGetBackendResponse(ctx, supplierSignerAddress, respBz, accountClient, logger)
	if err != nil {
		return err
	}
	logger.Info().Msgf("✅ Backend response status code: %v", backendHttpResponse.StatusCode)

	// Log response details
	if flagRelayRequestCount > 1 {
		logger.Info().Msgf("✅ Status code %d, Response size %d bytes", backendHttpResponse.StatusCode, len(respBz))
	} else {
		logger.Info().Msgf("✅ Backend response status code: %v", backendHttpResponse.StatusCode)
		logger.Info().Msgf("✅ Response read %d bytes", len(respBz))
	}

	err = unmarshalAndPrintResponse(backendHttpResponse.BodyBz, logger)
	if err != nil {
		return err
	}

	return nil
}

func checkAndGetBackendResponse(ctx context.Context,
	supplierSignerAddress string,
	respBz []byte,
	accountClient sdk.AccountClient,
	logger polylog.Logger) (backendHttpResponse *sdktypes.POKTHTTPResponse, err error) {
	// Validate the relay response
	relayResp, err := sdk.ValidateRelayResponse(
		ctx,
		sdk.SupplierAddress(supplierSignerAddress),
		respBz,
		&accountClient,
	)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error validating response")
		return
	}
	// Deserialize the relay response
	backendHttpResponse, err = sdktypes.DeserializeHTTPResponse(relayResp.Payload)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error deserializing response payload")
		return
	}

	return
}

func unmarshalAndPrintResponse(BodyBz []byte, logger polylog.Logger) error {
	var jsonMap map[string]interface{}
	// Unmarshal the HTTP response body into jsonMap
	if err := json.Unmarshal(BodyBz, &jsonMap); err != nil {
		logger.Error().Err(err).Msg("❌ Error deserializing backend response payload")
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

// If a supplier is specified but not in the session, try to fetch it directly.
// TODO_UPNEXT(@olshansk): Add support for sending a relay to a supplier that is not in the session.
// This will require starting a relayminer in debug mode to avoid validating the session header.
// NOTE: This function is currently unused. Linters such as staticcheck will flag it as U1000 (unused code).
//
//nolint:unused // TODO_WIP(@olshansk): keeping it here for an upcoming iteration to streamline debugging.
func querySupplier(
	logger polylog.Logger,
	grpcConn *grpc.ClientConn,
	ctx context.Context,
	serviceId string,
	supplierAddr string,
) (sdk.Endpoint, error) {
	logger.Warn().Msgf("⚠️ Supplier %s specified but not in session. Going to try to fetch it directly...", flagRelaySupplier)
	supplierClient := sdk.SupplierClient{
		QueryClient: suppliertypes.NewQueryClient(grpcConn),
	}
	supplier, err := supplierClient.GetSupplier(ctx, supplierAddr)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error fetching supplier")
		return nil, err
	}
	logger.Info().Msgf("✅ Supplier fetched successfully: %v", supplier)
	logger.Warn().Msgf("⚠️ Since the supplier %s was not in the session, there's no guarantee it will service the request.", flagRelaySupplier)

	for _, serviceConfig := range supplier.Services {
		if serviceConfig.ServiceId == serviceId {
			supplierEndpoint := serviceConfig.Endpoints[0]

			// Compose struct with Header, Supplier, Endpoint to comply with interface
			endpoint := &supplierEndpointWithHeader{
				// Supplier is not in the session, so we can't populate the header
				header: sessiontypes.SessionHeader{
					ApplicationAddress:      flagRelayApp,
					ServiceId:               serviceId,
					SessionId:               "",
					SessionStartBlockHeight: 0,
					SessionEndBlockHeight:   0,
				},
				supplier: sdk.SupplierAddress(flagRelaySupplier),
				endpoint: *supplierEndpoint,
			}

			logger.Info().Msgf("✅ Endpoint for service ID '%s' selected: %v", serviceId, endpoint)
			return sdk.Endpoint(endpoint), nil
		}
	}
	return nil, errors.New("no endpoint found")
}

// Struct to comply with interface requiring Header, Supplier, and Endpoint fields
// Used for relay endpoint assignment when supplier is fetched directly
// Header type is assumed to be interface{}; adjust as needed for actual type
// Supplier and Endpoint types are inferred from sdk and sharedtypes
// TODO_TECHDEBT(@olshansk): Remove this once the shannon-sdk is updated to have
// a struct that implements the Endpoint interface.
var _ sdk.Endpoint = (*supplierEndpointWithHeader)(nil)

type supplierEndpointWithHeader struct {
	header   sessiontypes.SessionHeader
	supplier sdk.SupplierAddress
	endpoint sharedtypes.SupplierEndpoint
}

func (e *supplierEndpointWithHeader) Header() sessiontypes.SessionHeader {
	return e.header
}

func (e *supplierEndpointWithHeader) Supplier() sdk.SupplierAddress {
	return e.supplier
}

func (e *supplierEndpointWithHeader) Endpoint() sharedtypes.SupplierEndpoint {
	return e.endpoint
}
