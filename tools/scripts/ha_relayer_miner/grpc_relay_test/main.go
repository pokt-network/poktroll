// Package main provides a gRPC relay test tool for the HA RelayMiner.
//
// This tool tests the proper relay protocol by sending signed RelayRequest messages
// via the /pocket.service.RelayService/SendRelay gRPC method.
//
// Usage:
//
//	go run ./tools/scripts/ha_relayer_miner/grpc_relay_test/main.go \
//	  --relayer-addr=localhost:8080 \
//	  --grpc-addr=sauron-grpc.beta.infra.pocket.network:443 \
//	  --node=https://sauron-rpc.beta.infra.pocket.network \
//	  --app=pokt1... \
//	  --app-key-hex=... \
//	  --num-requests=5
//
// The tool:
//   - Queries the current session from the chain using the Shannon SDK
//   - Creates properly signed RelayRequest protobufs
//   - Sends them via the RelayService/SendRelay gRPC method
//   - Receives and parses RelayResponse protobufs
package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"io"
	"log"
	"net/http"
	"time"

	sdk "github.com/pokt-network/shannon-sdk"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

const (
	// RelayServiceMethodPath is the gRPC method path for the relay service
	RelayServiceMethodPath = "/pocket.service.RelayService/SendRelay"
)

var (
	flagApp           string
	flagSupplier      string
	flagNode          string
	flagGRPCAddr      string
	flagGRPCInsecure  bool
	flagRelayerAddr   string
	flagRelayInsecure bool
	flagNumRequests   int
	flagAppKeyHex     string
	flagPayload       string
	flagServiceID     string
)

func init() {
	flag.StringVar(&flagApp, "app", "", "Application address (required)")
	flag.StringVar(&flagSupplier, "supplier", "", "Supplier address (optional, picks first from session)")
	flag.StringVar(&flagNode, "node", "https://sauron-rpc.beta.infra.pocket.network", "Node RPC URL")
	flag.StringVar(&flagGRPCAddr, "grpc-addr", "sauron-grpc.beta.infra.pocket.network:443", "gRPC address for chain queries")
	flag.BoolVar(&flagGRPCInsecure, "grpc-insecure", false, "Use insecure gRPC connection for chain queries")
	flag.StringVar(&flagRelayerAddr, "relayer-addr", "localhost:8080", "HA Relayer gRPC address")
	flag.BoolVar(&flagRelayInsecure, "relay-insecure", true, "Use insecure connection to relayer")
	flag.IntVar(&flagNumRequests, "num-requests", 5, "Number of gRPC requests to make")
	flag.StringVar(&flagAppKeyHex, "app-key-hex", "", "Application private key in hex format (required)")
	flag.StringVar(&flagPayload, "payload", `{"jsonrpc":"2.0","method":"test","params":[],"id":1}`, "Payload to send")
	flag.StringVar(&flagServiceID, "service-id", "", "Service ID (optional, derived from app if not set)")
}

func main() {
	flag.Parse()

	if flagApp == "" {
		log.Fatal("--app is required")
	}
	if flagAppKeyHex == "" {
		log.Fatal("--app-key-hex is required")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	if err := run(ctx); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run(ctx context.Context) error {
	log.Printf("Connecting to gRPC at %s...", flagGRPCAddr)

	// Initialize gRPC connection for chain queries
	var dialOpts []grpc.DialOption
	if flagGRPCInsecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})))
	}

	grpcConn, err := grpc.NewClient(flagGRPCAddr, dialOpts...)
	if err != nil {
		return err
	}
	defer grpcConn.Close()
	log.Println("Connected to gRPC")

	// Create account client for fetching public keys
	accountClient := sdk.AccountClient{
		PoktNodeAccountFetcher: sdk.NewPoktNodeAccountFetcher(grpcConn),
	}

	// Create application client
	appClient := sdk.ApplicationClient{
		QueryClient: apptypes.NewQueryClient(grpcConn),
	}

	// Get application
	log.Printf("Fetching application %s...", flagApp)
	app, err := appClient.GetApplication(ctx, flagApp)
	if err != nil {
		return err
	}
	log.Printf("Application: %s with %d service configs", app.Address, len(app.ServiceConfigs))

	if len(app.ServiceConfigs) == 0 {
		return errors.New("application has no service configs")
	}

	serviceID := flagServiceID
	if serviceID == "" {
		serviceID = app.ServiceConfigs[0].ServiceId
	}
	log.Printf("Using service ID: %s", serviceID)

	// Create application ring for signing
	ring := sdk.ApplicationRing{
		Application:      app,
		PublicKeyFetcher: &accountClient,
	}

	// Get block height
	nodeStatusFetcher, err := sdk.NewPoktNodeStatusFetcher(flagNode)
	if err != nil {
		return err
	}

	blockClient := sdk.BlockClient{
		PoktNodeStatusFetcher: nodeStatusFetcher,
	}
	blockHeight, err := blockClient.LatestBlockHeight(ctx)
	if err != nil {
		return err
	}
	log.Printf("Block height: %d", blockHeight)

	// Get session
	sessionClient := sdk.SessionClient{
		PoktNodeSessionFetcher: sdk.NewPoktNodeSessionFetcher(grpcConn),
	}
	session, err := sessionClient.GetSession(ctx, app.Address, serviceID, blockHeight)
	if err != nil {
		return err
	}
	log.Printf("Session: %s with %d suppliers", session.SessionId, len(session.Suppliers))
	log.Printf("Session start: %d, end: %d", session.Header.SessionStartBlockHeight, session.Header.SessionEndBlockHeight)

	// Find endpoint for supplier
	sessionFilter := sdk.SessionFilter{
		Session:         session,
		EndpointFilters: []sdk.EndpointFilter{},
	}
	endpoints, err := sessionFilter.FilteredEndpoints()
	if err != nil {
		return err
	}

	var endpoint sdk.Endpoint
	if flagSupplier != "" {
		for _, e := range endpoints {
			if string(e.Supplier()) == flagSupplier {
				endpoint = e
				break
			}
		}
		if endpoint == nil {
			return errors.New("supplier not found in session: " + flagSupplier)
		}
	} else if len(endpoints) > 0 {
		endpoint = endpoints[0]
		flagSupplier = string(endpoint.Supplier())
	} else {
		return errors.New("no endpoints available")
	}
	log.Printf("Using endpoint for supplier: %s", flagSupplier)

	// Create signer directly from hex private key
	appSigner, err := sdk.NewSignerFromHex(flagAppKeyHex)
	if err != nil {
		return err
	}
	log.Printf("Created signer for app: %s", flagApp)

	// Connect to the HA Relayer
	log.Printf("Connecting to HA Relayer gRPC at %s...", flagRelayerAddr)
	var relayerDialOpts []grpc.DialOption
	if flagRelayInsecure {
		relayerDialOpts = append(relayerDialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		relayerDialOpts = append(relayerDialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{
			MinVersion: tls.VersionTLS12,
		})))
	}

	relayerConn, err := grpc.NewClient(flagRelayerAddr, relayerDialOpts...)
	if err != nil {
		return err
	}
	defer relayerConn.Close()
	log.Println("Connected to HA Relayer")

	// Get the endpoint URL for creating the HTTP request
	endpointUrl := endpoint.Endpoint().Url

	// Send relay requests
	successCount := 0
	for i := 1; i <= flagNumRequests; i++ {
		log.Printf("Sending request %d/%d...", i, flagNumRequests)

		// Create an HTTP request to serialize (like the HTTP test does)
		body := io.NopCloser(bytes.NewReader([]byte(flagPayload)))
		httpReq, err := http.NewRequest(http.MethodPost, endpointUrl, body)
		if err != nil {
			log.Printf("Request %d: Failed to create HTTP request: %v", i, err)
			continue
		}
		httpReq.Header.Set("Content-Type", "application/json")

		// Serialize to POKTHTTPRequest protobuf format
		_, payloadBz, err := sdktypes.SerializeHTTPRequest(httpReq)
		if err != nil {
			log.Printf("Request %d: Failed to serialize HTTP request: %v", i, err)
			continue
		}

		// Build relay request using the SDK with properly serialized payload
		relayReq, err := sdk.BuildRelayRequest(endpoint, payloadBz)
		if err != nil {
			log.Printf("Request %d: Failed to build relay request: %v", i, err)
			continue
		}

		// Sign the request
		signedRelayReq, err := appSigner.Sign(ctx, relayReq, &ring)
		if err != nil {
			log.Printf("Request %d: Failed to sign relay request: %v", i, err)
			continue
		}

		// Send via gRPC
		startTime := time.Now()
		relayResponse, err := sendRelayRequest(ctx, relayerConn, signedRelayReq)
		latency := time.Since(startTime)

		if err != nil {
			log.Printf("Request %d: FAILED: %v", i, err)
			continue
		}

		successCount++
		log.Printf("Request %d: SUCCESS (latency=%v)", i, latency)
		log.Printf("Request %d: Response payload size: %d bytes", i, len(relayResponse.Payload))

		if len(relayResponse.Payload) > 0 {
			payloadStr := string(relayResponse.Payload)
			if len(payloadStr) > 100 {
				payloadStr = payloadStr[:100] + "..."
			}
			log.Printf("Request %d: Payload: %s", i, payloadStr)
		}

		if relayResponse.Meta.SessionHeader != nil {
			sessionID := relayResponse.Meta.SessionHeader.SessionId
			if len(sessionID) > 16 {
				sessionID = sessionID[:16] + "..."
			}
			log.Printf("Request %d: Session ID in response: %s", i, sessionID)
		}

		if len(relayResponse.Meta.SupplierOperatorSignature) > 0 {
			log.Printf("Request %d: Response is signed by supplier", i)
		}

	}

	log.Printf("Successfully sent %d/%d gRPC relay requests", successCount, flagNumRequests)
	return nil
}

// sendRelayRequest sends a RelayRequest via the gRPC method and returns the RelayResponse.
func sendRelayRequest(ctx context.Context, conn *grpc.ClientConn, request *servicetypes.RelayRequest) (*servicetypes.RelayResponse, error) {
	response := &servicetypes.RelayResponse{}
	err := conn.Invoke(
		ctx,
		RelayServiceMethodPath,
		request,
		response,
	)
	if err != nil {
		return nil, err
	}
	return response, nil
}
