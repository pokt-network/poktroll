// Package cmd contains the relayminer CLI commands and utilities.
package cmd

import (
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

	"github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	sdk "github.com/pokt-network/shannon-sdk"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"

	"github.com/pokt-network/poktroll/cmd/flags"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
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
	defer grpcConn.Close()
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

<<<<<<< Updated upstream
	// Initialize the signer properly so that the private key hex is decoded into bytes.
	// Previously this was constructed as a literal which left privateKeyBytes empty and
	// caused: "Sign: error decoding private key to scalar: invalid scalar length".
	appSigner, err := sdk.NewSignerFromHex(appPrivateKeyHex)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error initializing signer from private key hex")
		return err
	}
=======
	// TODO_INVESTIGATE: CGO-related signer initialization disabled - https://github.com/pokt-network/poktroll/discussions/1822
	// Initialize the signer properly so that the private key hex is decoded into bytes.
	// Previously this was constructed as a literal which left privateKeyBytes empty and
	// caused: "Sign: error decoding private key to scalar: invalid scalar length".
	// appSigner, err := sdk.NewSignerFromHex(appPrivateKeyHex)
	// if err != nil {
	// 	logger.Error().Err(err).Msg("❌ Error initializing signer from private key hex")
	// 	return err
	// }
	appSigner := sdk.Signer{PrivateKeyHex: appPrivateKeyHex}
>>>>>>> Stashed changes

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

		signedRelayReq, err := appSigner.Sign(ctx, relayReq, &ring)
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

		// Create the HTTP request with the relay request body
		httpReq := &http.Request{
			Method: http.MethodPost,
			URL:    reqUrl,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
			Body: io.NopCloser(bytes.NewReader(relayReqBz)),
		}

		// Send the HTTP request containing the signed relay request
		httpResp, err := http.DefaultClient.Do(httpReq)
		if err != nil {
			logger.Error().Err(err).Msgf("❌ Error sending relay request %d", i)
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
			logger.Error().Err(err).Msgf("❌ Error reading response %d", i)
			continue
		}

		// This is intentionally not a defer because the loop could introduce memory leaks,
		// performance issues and bad connection management for high flagRelayRequestCount values
		httpResp.Body.Close()

		// Ensure the supplier operator signature is present
		supplierSignerAddress := signedRelayReq.Meta.SupplierOperatorAddress
		if supplierSignerAddress == "" {
			logger.Error().Msg("❌ Supplier operator signature is missing")
			continue
		}

		// Ensure the supplier operator address matches the expected address
		if flagRelaySupplier == "" {
			if flagRelayRequestCount == 1 {
				logger.Warn().Msg("⚠️ Supplier operator address not specified, skipping signature check")
			}
		} else if supplierSignerAddress != flagRelaySupplier {
			logger.Error().Msgf("❌ Supplier operator address %s does not match the expected address %s", supplierSignerAddress, flagRelaySupplier)
			continue
		}

		responseReadDuration := time.Since(beforeResponseReadTime)
		logger.Info().Msgf("⏱️ Response building duration: %s", responseReadDuration)

		beforeResponseVerificationTime := time.Now()

		// Validate the relay response
		relayResp, err := sdk.ValidateRelayResponse(
			ctx,
			sdk.SupplierAddress(supplierSignerAddress),
			respBz,
			&accountClient,
		)
		if err != nil {
			logger.Error().Err(err).Msgf("❌ Error validating response %d", i)
			continue
		}

		responseVerificationDuration := time.Since(beforeResponseVerificationTime)
		logger.Info().Msgf("⏱️ Response verification duration: %s", responseVerificationDuration)

		beforeBackendResponseExtractionTime := time.Now()

		// Deserialize the relay response
		backendHttpResponse, err := sdktypes.DeserializeHTTPResponse(relayResp.Payload)
		if err != nil {
			logger.Error().Err(err).Msgf("❌ Error deserializing response payload %d", i)
			continue
		}

		backendResponseExtractionDuration := time.Since(beforeBackendResponseExtractionTime)
		logger.Info().Msgf("⏱️ Backend response extraction duration: %s", backendResponseExtractionDuration)

		totalRequestDuration := time.Since(beforeRequestPreparationTime)
		logger.Info().Msgf("⏱️ Total request duration: %s", totalRequestDuration)

		// Unmarshal the HTTP response body into jsonMap
		var jsonMap map[string]interface{}
		if err := json.Unmarshal(backendHttpResponse.BodyBz, &jsonMap); err != nil {
			logger.Error().Err(err).Msgf("❌ Error unmarshaling response into a JSON map %d", i)
			continue
		}

		// Log response details
		if flagRelayRequestCount > 1 {
			logger.Info().Msgf("✅ Request %d: Status code %d, Response size %d bytes", i, backendHttpResponse.StatusCode, len(respBz))
		} else {
			logger.Info().Msgf("✅ Backend response status code: %v", backendHttpResponse.StatusCode)
			logger.Info().Msgf("✅ Response read %d bytes", len(respBz))
			logger.Info().Msgf("✅ Deserialized response body as JSON map: %+v", jsonMap)

<<<<<<< Updated upstream
			// Print the JSON response to stdout for CLI users and testing
			jsonOutput, err := json.MarshalIndent(jsonMap, "", "  ")
			if err != nil {
				logger.Error().Err(err).Msg("❌ Error marshaling JSON for stdout")
			} else {
				fmt.Println(string(jsonOutput))
			}
=======
			// TODO_INVESTIGATE: JSON stdout output disabled (may be related to CGO changes) - https://github.com/pokt-network/poktroll/discussions/1822
			// Print the JSON response to stdout for CLI users and testing
			// jsonOutput, err := json.MarshalIndent(jsonMap, "", "  ")
			// if err != nil {
			// 	logger.Error().Err(err).Msg("❌ Error marshaling JSON for stdout")
			// } else {
			// 	fmt.Println(string(jsonOutput))
			// }
>>>>>>> Stashed changes
		}

		// If "jsonrpc" key exists, try to further deserialize "result".
		// Only do this once for the first request.
		if flagRelayRequestCount == 1 || i == 1 {
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
				if flagRelayRequestCount > 1 {
					logger.Debug().Msg("⚠️ Will be skipping JSON-RPC deserialization for subsequent requests")
				}
			}
		}
	}

	if flagRelayRequestCount > 1 {
		logger.Info().Msgf("✅ Successfully sent %d relay requests", flagRelayRequestCount)
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

func (e *supplierEndpointWithHeader) RPCType() sharedtypes.RPCType {
	return e.endpoint.RpcType
}
