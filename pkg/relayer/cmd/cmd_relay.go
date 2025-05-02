package cmd

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/hex"
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
	sdk "github.com/pokt-network/shannon-sdk"
	"github.com/spf13/cobra"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
)

var (
	flagRelayApp      string
	flagRelaySupplier string
	flagRelayPayload  string
	flagServiceID     string
)

// relayCmd defines the `relay` subcommand for sending a relay as an application.
func relayCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "relay",
		Short: "Send a relay",
		Long:  `Send a test relay to an actively staked Application and its RelayMiner from a staked application.`,
		RunE:  runRelay,
	}

	// Cosmos flags
	cmd.Flags().StringVar(&flagNodeRPCURL, cosmosflags.FlagNode, "https://shannon-testnet-grove-rpc.beta.poktroll.com", "Cosmos node RPC URL (required)")
	cmd.Flags().StringVar(&flagNodeGRPCURL, cosmosflags.FlagGRPC, "shannon-testnet-grove-grpc.beta.poktroll.com:443", "Cosmos node GRPC URL (required)")
	cmd.Flags().BoolVar(&flagNodeGRPCInsecure, cosmosflags.FlagGRPCInsecure, false, "Used to initialize the Cosmos query context with grpc security options.")

	// Relayer flags
	// --dry-run
	// --specific-endpoint
	cmd.Flags().StringVar(&flagRelayApp, "app", "supplier", "Name of the staked application key (required)")
	cmd.Flags().StringVar(&flagRelaySupplier, "supplier", "supplier", "Supplier endpoint URL (e.g. http://localhost:8081/relay)")
	cmd.Flags().StringVar(&flagRelayPayload, "payload", "{\"jsonrpc\": \"2.0\", \"id\": 1, \"method\": \"eth_blockNumber\", \"params\": []}", "Relay payload (hex encoded)")
	cmd.Flags().StringVar(&flagServiceID, "service-id", "anvil", "Service ID (required)")
	// _ = cmd.MarkFlagRequired("app")
	// _ = cmd.MarkFlagRequired("supplier")
	// _ = cmd.MarkFlagRequired("payload")
	// _ = cmd.MarkFlagRequired("service-id")

	return cmd
}

func runRelay(cmd *cobra.Command, args []string) error {
	fmt.Println("Running relay...")
	// Initialize gRPC connection
	grpcConn, err := connectGRPC(GRPCConfig{
		HostPort: flagNodeGRPCURL,
		Insecure: flagNodeGRPCInsecure,
	})
	if err != nil {
		return err
	}
	defer grpcConn.Close()
	fmt.Println("\n\ngRPC connection initialized")

	// 1. Create a connection to the POKT full node
	nodeStatusFetcher, err := sdk.NewPoktNodeStatusFetcher(flagNodeRPCURL)
	if err != nil {
		fmt.Printf("Error fetching block height: %v\n", err)
		return err
	}
	fmt.Println("\n\nNode status fetcher initialized")

	// 2. Get the latest block height
	blockClient := sdk.BlockClient{
		PoktNodeStatusFetcher: nodeStatusFetcher,
	}
	blockHeight, err := blockClient.LatestBlockHeight(context.Background())
	if err != nil {
		fmt.Printf("Error fetching block height: %v\n", err)
		return err
	}
	fmt.Println("\n\nBlock height: ", blockHeight)

	// 3. Get the current session
	sessionClient := sdk.SessionClient{
		PoktNodeSessionFetcher: sdk.NewPoktNodeSessionFetcher(grpcConn),
	}
	session, err := sessionClient.GetSession(
		context.Background(),
		flagRelayApp,
		flagServiceID,
		blockHeight,
	)
	if err != nil {
		fmt.Printf("Error fetching session for app %s and service ID %s: %v\n", flagRelayApp, flagServiceID, err)
		return err
	}
	fmt.Println("\n\nSession fetched:", session)

	// 4. Select an endpoint from the session
	sessionFilter := sdk.SessionFilter{
		Session:         session,
		EndpointFilters: []sdk.EndpointFilter{},
	}
	endpoints, err := sessionFilter.FilteredEndpoints()
	if err != nil {
		fmt.Printf("Error filtering endpoints: %v\n", err)
		return err
	}
	if len(endpoints) == 0 {
		fmt.Println("No endpoints available")
		return err
	}
	fmt.Println("\n\nEndpoints fetched:", endpoints)

	// 5. Build a relay request
	relayReq, err := sdk.BuildRelayRequest(endpoints[0], []byte(flagRelayPayload))
	if err != nil {
		fmt.Printf("Error building relay request: %v\n", err)
		return err
	}
	fmt.Println("\n\nRelay request built:", relayReq)

	// 6. Create an account client for fetching public keys
	accountClient := sdk.AccountClient{
		PoktNodeAccountFetcher: sdk.NewPoktNodeAccountFetcher(grpcConn),
	}
	fmt.Println("\n\nAccount client initialized")

	// 7. Create an application client to get application details
	appClient := sdk.ApplicationClient{
		QueryClient: apptypes.NewQueryClient(grpcConn),
	}
	app, err := appClient.GetApplication(context.Background(), flagRelayApp)
	if err != nil {
		fmt.Printf("Error fetching application: %v\n", err)
		return err
	}
	fmt.Println("\n\nApplication fetched:", app)

	// 8. Create an application ring for signing
	ring := sdk.ApplicationRing{
		Application:      app,
		PublicKeyFetcher: &accountClient,
	}
	fmt.Println("\n\nApplication ring created:", ring)

	// 9. Sign the relay request
	clientCtx := client.GetClientContextFromCmd(cmd)
	privateKeyHex, err := getPrivateKeyHexFromKeyring(clientCtx.Keyring, "supplier")
	if err != nil {
		fmt.Printf("Error getting private key: %v\n", err)
		return err
	}
	signer := sdk.Signer{PrivateKeyHex: privateKeyHex}
	signedRelayReq, err := signer.Sign(context.Background(), relayReq, ring)
	if err != nil {
		fmt.Printf("Error signing relay request: %v\n", err)
		return err
	}
	fmt.Println("\n\nRelay request signed:", signedRelayReq)

	// 10. Send the relay request to the endpoint
	relayReqBz, err := signedRelayReq.Marshal()
	if err != nil {
		fmt.Printf("Error marshaling relay request: %v\n", err)
		return err
	}
	fmt.Println("\n\nRelay request marshaled:", relayReqBz)

	reqUrl, err := url.Parse(endpoints[0].Endpoint().Url)
	if err != nil {
		fmt.Printf("Error parsing endpoint URL: %v\n", err)
		return err
	}
	fmt.Println("\n\nEndpoint URL parsed:", reqUrl)

	httpReq := &http.Request{
		Method: http.MethodPost,
		URL:    reqUrl,
		Body:   io.NopCloser(bytes.NewReader(relayReqBz)),
	}

	// Send the request
	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		fmt.Printf("Error sending relay request: %v\n", err)
		return err
	}
	defer httpResp.Body.Close()

	// 11. Read the response
	respBz, err := io.ReadAll(httpResp.Body)
	if err != nil {
		fmt.Printf("Error reading response: %v\n", err)
		return err
	}
	fmt.Println("Response read:", respBz)

	// 12. Validate the relay response
	validatedResp, err := sdk.ValidateRelayResponse(
		context.Background(),
		sdk.SupplierAddress(signedRelayReq.Meta.SupplierOperatorAddress),
		respBz,
		&accountClient,
	)
	if err != nil {
		fmt.Printf("Error validating response: %v\n", err)
		return err
	}

	fmt.Printf("Relay successful: %v\n", validatedResp)

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
func getPrivateKeyHexFromKeyring(kr keyring.Keyring, keyName string) (string, error) {
	// Get the key info from the keyring
	// key, err := kr.Key(keyName)
	// if err != nil {
	// 	return "", fmt.Errorf("failed to get key: %w", err)
	// }

	// Get the address from the public key
	// address := key.GetAddress()

	// Export the private key in armored format
	armoredPrivKey, err := kr.ExportPrivKeyArmor(keyName, "") // Empty passphrase
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
		return "", fmt.Errorf("key %s is not a secp256k1 key", keyName)
	}

	// Convert to hex
	hexKey := hex.EncodeToString(secpPrivKey.Key)
	return hexKey, nil
}
