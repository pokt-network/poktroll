package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	cosmosflags "github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/crypto"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/pokt-network/shannon-sdk"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

var (
	flagRelayApp                       string
	flagRelaySupplier                  string
	flagRelayPayload                   string
	flagServiceID                      string
	flagSupplierPublicEndpointOverride string

	flagNodeRPCURLRelay       string
	flagNodeGRPCURLRelay      string
	flagNodeGRPCInsecureRelay bool
)

// TODO_IN_THIS_PR: Revisit these ideas
// --dry-run
// --specific-endpoint
// -- don't validate
// -- what if I'm not staked?
// -- what if supplier is not staked?
// -- Both unstaked, one of unstaked
// -- One validates, no one valides

// relayCmd defines the `relay` subcommand for sending a relay as an application.
func relayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay",
		Short: "Send a relay as an application to a particular supplier",
		// TODO_IN_THIS_PR: Add a long description and add examples
		Long: "Send a test relay to an actively staked Application and its RelayMiner from a staked application.",
		RunE: runRelay,
	}

	// Cosmos flags
	// cmd.Flags().StringVar(&flagNodeRPCURLRelay, cosmosflags.FlagNode, "https://shannon-testnet-grove-rpc.beta.poktroll.com", "Cosmos node RPC URL (required)")
	// cmd.Flags().StringVar(&flagNodeGRPCURLRelay, cosmosflags.FlagGRPC, "shannon-testnet-grove-grpc.beta.poktroll.com:443", "Cosmos node GRPC URL (required)")
	cmd.Flags().StringVar(&flagNodeRPCURLRelay, cosmosflags.FlagNode, "tcp://127.0.0.1:26657", "Cosmos node RPC URL (required)")
	cmd.Flags().StringVar(&flagNodeGRPCURLRelay, cosmosflags.FlagGRPC, "localhost:9090", "Cosmos node GRPC URL (required)")
	cmd.Flags().BoolVar(&flagNodeGRPCInsecureRelay, cosmosflags.FlagGRPCInsecure, true, "Used to initialize the Cosmos query context with grpc security options.")

	// Custom Flags
	// app1 - pokt1mrqt5f7qh8uxs27cjm9t7v9e74a9vvdnq5jva4
	// supplier1 - pokt19a3t4yunp0dlpfjrp7qwnzwlrzd5fzs2gjaaaj
	cmd.Flags().StringVar(&flagRelayApp, "app", "application", "Name of the staked application key (required)")
	cmd.Flags().StringVar(&flagRelaySupplier, "supplier", "supplier", "Supplier endpoint URL (e.g. http://localhost:8081/relay)")
	cmd.Flags().StringVar(&flagServiceID, "service-id", "anvil", "Service ID (required)")
	cmd.Flags().StringVar(&flagRelayPayload, "payload", "{\"jsonrpc\": \"2.0\", \"id\": 1, \"method\": \"eth_blockNumber\", \"params\": []}", "Relay payload")
	cmd.Flags().StringVar(&flagSupplierPublicEndpointOverride, "supplier-public-endpoint-override", "http://localhost:8085", "Override the supplier public endpoint. Useful for local testing.")

	_ = cmd.MarkFlagRequired("app")
	_ = cmd.MarkFlagRequired("supplier")
	// TODO_IN_THIS_PR: Uncomment this.
	// _ = cmd.MarkFlagRequired("payload")
	// _ = cmd.MarkFlagRequired("service-id")

	return cmd
}

func runRelay(cmd *cobra.Command, args []string) error {
	fmt.Printf("About to send a relay to %s for app %s and service ID %s\n", flagRelaySupplier, flagRelayApp, flagServiceID)

	// Initialize gRPC connection
	grpcConn, err := connectGRPC(GRPCConfig{
		HostPort: flagNodeGRPCURLRelay,
		Insecure: flagNodeGRPCInsecureRelay,
	})
	if err != nil {
		return err
	}
	defer grpcConn.Close()
	fmt.Printf("✅ gRPC connection initialized\n")

	// Create a connection to the POKT full node
	nodeStatusFetcher, err := sdk.NewPoktNodeStatusFetcher(flagNodeRPCURLRelay)
	if err != nil {
		fmt.Printf("❌ Error fetching block height: %v\n", err)
		return err
	}
	fmt.Printf("✅ Node status fetcher initialized\n")

	// Create an account client for fetching public keys
	accountClient := sdk.AccountClient{
		PoktNodeAccountFetcher: sdk.NewPoktNodeAccountFetcher(grpcConn),
	}
	fmt.Printf("✅ Account client initialized\n")

	// Create an application client to get application details
	appClient := sdk.ApplicationClient{
		QueryClient: apptypes.NewQueryClient(grpcConn),
	}
	app, err := appClient.GetApplication(context.Background(), flagRelayApp)
	if err != nil {
		fmt.Printf("❌ Error fetching application: %v\n", err)
		return err
	}
	fmt.Printf("✅ Application fetched: %v\n", app)

	// Create an application ring for signing
	ring := sdk.ApplicationRing{
		Application:      app,
		PublicKeyFetcher: &accountClient,
	}
	fmt.Printf("✅ Application ring created\n")

	// Get the latest block height
	blockClient := sdk.BlockClient{
		PoktNodeStatusFetcher: nodeStatusFetcher,
	}
	blockHeight, err := blockClient.LatestBlockHeight(context.Background())
	if err != nil {
		fmt.Printf("❌ Error fetching block height: %v\n", err)
		return err
	}
	fmt.Printf("✅ Block height retrieved: %d\n", blockHeight)

	// Get the current session
	sessionClient := sdk.SessionClient{
		PoktNodeSessionFetcher: sdk.NewPoktNodeSessionFetcher(grpcConn),
	}
	session, err := sessionClient.GetSession(
		context.Background(),
		app.Address,
		flagServiceID,
		blockHeight,
	)
	if err != nil {
		fmt.Printf("❌ Error fetching session for app %s and service ID %s: %v\n", app.Address, flagServiceID, err)
		return err
	}
	fmt.Printf("✅ Session fetched: %v\n", session)

	// Select an endpoint from the session
	sessionFilter := sdk.SessionFilter{
		Session:         session,
		EndpointFilters: []sdk.EndpointFilter{},
	}
	endpoints, err := sessionFilter.FilteredEndpoints()
	if err != nil {
		fmt.Printf("❌ Error filtering endpoints: %v\n", err)
		return err
	}
	if len(endpoints) == 0 {
		fmt.Println("❌ No endpoints available")
		return err
	}
	fmt.Printf("✅ Endpoints fetched: %v\n", endpoints)

	var endpoint sdk.Endpoint
	for _, e := range endpoints {
		if string(e.Supplier()) == flagRelaySupplier {
			endpoint = e
			break
		}
	}
	if endpoint == nil {
		fmt.Printf("❌ No endpoint found for supplier %s in the current session\n", flagRelaySupplier)
		return err
	}
	fmt.Printf("✅ Endpoint selected: %v\n", endpoint)

	// Get the endpoint URL
	endpointUrl := endpoint.Endpoint().Url
	// Override the endpoint URL if specified
	if flagSupplierPublicEndpointOverride != "" {
		endpointUrl = flagSupplierPublicEndpointOverride
		fmt.Printf("⚠️ Using override endpoint URL: %s\n", endpointUrl)
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
	fmt.Printf("✅ JSON-RPC request payload serialized.\n")

	// Build a relay request
	relayReq, err := sdk.BuildRelayRequest(endpoint, payloadBz)
	if err != nil {
		fmt.Printf("❌ Error building relay request: %v\n", err)
		return err
	}
	fmt.Printf("✅ Relay request built.\n")

	// Sign the relay request
	clientCtx := client.GetClientContextFromCmd(cmd)
	appPrivateKeyHex, err := getPrivateKeyHexFromKeyring(clientCtx.Keyring, app.Address)
	if err != nil {
		fmt.Printf("❌ Error getting private key: %v\n", err)
		return err
	}
	fmt.Printf("✅ Retrieved private key for app %s\n", app.Address)
	appSigner := sdk.Signer{PrivateKeyHex: appPrivateKeyHex}
	signedRelayReq, err := appSigner.Sign(context.Background(), relayReq, ring)
	if err != nil {
		fmt.Printf("❌ Error signing relay request: %v\n", err)
		return err
	}
	fmt.Printf("✅ Relay request signed.\n")

	// Marshal the signed relay request
	relayReqBz, err := signedRelayReq.Marshal()
	if err != nil {
		fmt.Printf("❌ Error marshaling relay request: %v\n", err)
		return err
	}
	fmt.Printf("✅ Relay request marshaled.\n")

	// Parse the endpoint URL
	reqUrl, err := url.Parse(endpointUrl)
	if err != nil {
		fmt.Printf("❌ Error parsing endpoint URL: %v\n", err)
		return err
	}
	fmt.Printf("✅ Endpoint URL parsed: %v\n", reqUrl)

	// Create the HTTP request with the relay request body
	httpReq := &http.Request{
		Method: http.MethodPost,
		URL:    reqUrl,
		Body:   io.NopCloser(bytes.NewReader(relayReqBz)),
	}

	// Send the request HTTP request containing the signed relay request
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		fmt.Printf("❌ Error sending relay request: %v\n", err)
		return err
	}
	defer httpResp.Body.Close()

	// Read the response
	respBz, err := io.ReadAll(httpResp.Body)
	if err != nil {
		fmt.Printf("❌ Error reading response: %v\n", err)
		return err
	}
	fmt.Printf("✅ Response read %d bytes\n", len(respBz))

	// Ensure the supplier operator signature is present
	supplierSignerAddress := signedRelayReq.Meta.SupplierOperatorAddress
	if supplierSignerAddress == "" {
		fmt.Printf("❌ Supplier operator signature is missing\n")
		return errors.New("Relay response missing supplier operator signature")
	}

	// Ensure the supplier operator address matches the expected address
	if supplierSignerAddress != flagRelaySupplier {
		fmt.Printf("❌ Supplier operator address %s does not match the expected address %s\n", supplierSignerAddress, flagRelaySupplier)
		return errors.New("Relay response supplier operator signature does not match")
	}

	// Validate the relay response
	relayResp, err := sdk.ValidateRelayResponse(
		context.Background(),
		sdk.SupplierAddress(supplierSignerAddress),
		respBz,
		&accountClient,
	)
	if err != nil {
		fmt.Printf("❌ Error validating response: %v\n", err)
		return err
	}
	// Deserialize the relay response
	backendHttpResponse, err := sdktypes.DeserializeHTTPResponse(relayResp.Payload)
	if err != nil {
		fmt.Printf("❌ Error deserializing response payload: %v\n", err)
		return err
	}
	fmt.Printf("✅ Backend response status code: %v\n", backendHttpResponse.StatusCode)

	var jsonMap map[string]interface{}
	// Unmarshal the HTTP response body into jsonMap
	if err := json.Unmarshal(backendHttpResponse.BodyBz, &jsonMap); err != nil {
		fmt.Printf("❌ Error deserializing response payload: %v\n", err)
		return err
	}
	fmt.Printf("✅ Deserialized JSON map: %+v\n", jsonMap)

	// If "jsonrpc" key exists, try to further deserialize "result"
	if _, ok := jsonMap["jsonrpc"]; ok {
		resultRaw, exists := jsonMap["result"]
		if exists {
			switch v := resultRaw.(type) {
			case map[string]interface{}:
				fmt.Printf("✅ Further deserialized 'result' (object): %+v\n", v)
			case []interface{}:
				fmt.Printf("✅ Further deserialized 'result' (array): %+v\n", v)
			case string:
				fmt.Printf("✅ Further deserialized 'result' (string): %s\n", v)
			case float64, bool, nil:
				fmt.Printf("✅ Further deserialized 'result' (primitive): %+v\n", v)
			default:
				fmt.Printf("⚠️ 'result' is of an unhandled type: %T, value: %+v\n", v, v)
			}
		}
	}

	return nil
}

type GRPCConfig struct {
	HostPort          string        `yaml:"host_port"`
	Insecure          bool          `yaml:"insecure"`
	BackoffBaseDelay  time.Duration `yaml:"backoff_base_delay"`
	BackoffMaxDelay   time.Duration `yaml:"backoff_max_delay"`
	MinConnectTimeout time.Duration `yaml:"min_connect_timeout"`
	KeepAliveTime     time.Duration `yaml:"keep_alive_time"`
	KeepAliveTimeout  time.Duration `yaml:"keep_alive_timeout"`
}

func connectGRPC(config GRPCConfig) (*grpc.ClientConn, error) {
	if config.Insecure {
		transport := grpc.WithTransportCredentials(insecure.NewCredentials())
		dialOptions := []grpc.DialOption{transport}
		return grpc.NewClient(
			config.HostPort,
			dialOptions...,
		)
	}

	return grpc.NewClient(
		config.HostPort,
		grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})),
	)
}

// getPrivateKeyHexFromKeyring takes a key name and returns the private key in hex format
// func getPrivateKeyHexFromKeyring(kr keyring.Keyring, keyName string) (string, error) {
func getPrivateKeyHexFromKeyring(kr keyring.Keyring, address string) (string, error) {
	// Export the private key in armored format
	// armoredPrivKey, err := kr.ExportPrivKeyArmor(keyName, "") // Empty passphrase
	cosmosAddr := cosmostypes.MustAccAddressFromBech32(address)
	armoredPrivKey, err := kr.ExportPrivKeyArmorByAddress(cosmosAddr, "") // Empty passphrase
	if err != nil {
		return "", fmt.Errorf("failed to export armored private key: %w", err)
	}

	// Unarmor the private key
	privKey, _, err := crypto.UnarmorDecryptPrivKey(armoredPrivKey, "") // Empty passphrase
	if err != nil {
		return "", fmt.Errorf("failed to unarmor private key: %w", err)
	}

	// Convert to secp256k1 private key
	secpPrivKey, ok := privKey.(*secp256k1.PrivKey)
	if !ok {
		return "", fmt.Errorf("key %s is not a secp256k1 key", address)
	}

	// Convert to hex
	hexKey := hex.EncodeToString(secpPrivKey.Key)
	return hexKey, nil
}

// TODO_IN_THIS_PR: Cleanup the code blow
// The code below was copied from the PATH SDK.
// sdk.BuildRelayRequest was not sufficient and hard to use.
// PATH doesn't even use sdk.BuildRelayRequest
// Need to understand why/how we got here and consolidate.
// Considering:
// - Moving the code below into the shannon-sdk
// - Updating this relay cmd
// - Updating PATH
// - Make everything easier.
// Look at buildUnsignedRelayRequest et al
