// Package main provides a WebSocket relay test tool for the HA RelayMiner.
//
// This tool connects to the HA Relayer via WebSocket and sends signed
// RelayRequest protobuf messages. Each request/response pair is billed
// as a relay by the relayer.
//
// Usage:
//
//	go run ./tools/scripts/ws_relay_test/main.go \
//	  --app=<app_address> \
//	  --supplier=<supplier_address> \
//	  --relayer-ws-url=ws://localhost:8080 \
//	  --num-messages=5
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	sdk "github.com/pokt-network/shannon-sdk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

var (
	flagApp          string
	flagSupplier     string
	flagNode         string
	flagGRPCAddr     string
	flagGRPCInsecure bool
	flagRelayerWSURL string
	flagNumMessages  int
	flagAppKeyHex    string
	flagPayload      string
	flagServiceID    string
)

func init() {
	flag.StringVar(&flagApp, "app", "", "Application address")
	flag.StringVar(&flagSupplier, "supplier", "", "Supplier address")
	flag.StringVar(&flagNode, "node", "https://sauron-rpc.beta.infra.pocket.network", "Node RPC URL")
	flag.StringVar(&flagGRPCAddr, "grpc-addr", "sauron-grpc.beta.infra.pocket.network:443", "gRPC address")
	flag.BoolVar(&flagGRPCInsecure, "grpc-insecure", false, "Use insecure gRPC connection")
	flag.StringVar(&flagRelayerWSURL, "relayer-ws-url", "ws://localhost:8080", "Relayer WebSocket URL")
	flag.IntVar(&flagNumMessages, "num-messages", 5, "Number of messages to send")
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

	// Initialize gRPC connection
	var dialOpts []grpc.DialOption
	if flagGRPCInsecure {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		dialOpts = append(dialOpts, grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}

	grpcConn, err := grpc.NewClient(flagGRPCAddr, dialOpts...)
	if err != nil {
		return fmt.Errorf("failed to connect to gRPC: %w", err)
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
		return fmt.Errorf("failed to get application: %w", err)
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
		return fmt.Errorf("failed to create node status fetcher: %w", err)
	}

	blockClient := sdk.BlockClient{
		PoktNodeStatusFetcher: nodeStatusFetcher,
	}
	blockHeight, err := blockClient.LatestBlockHeight(ctx)
	if err != nil {
		return fmt.Errorf("failed to get block height: %w", err)
	}
	log.Printf("Block height: %d", blockHeight)

	// Get session
	sessionClient := sdk.SessionClient{
		PoktNodeSessionFetcher: sdk.NewPoktNodeSessionFetcher(grpcConn),
	}
	session, err := sessionClient.GetSession(ctx, app.Address, serviceID, blockHeight)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}
	log.Printf("Session: %s with %d suppliers", session.SessionId, len(session.Suppliers))

	// Find endpoint for supplier
	sessionFilter := sdk.SessionFilter{
		Session:         session,
		EndpointFilters: []sdk.EndpointFilter{},
	}
	endpoints, err := sessionFilter.FilteredEndpoints()
	if err != nil {
		return fmt.Errorf("failed to filter endpoints: %w", err)
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
			return fmt.Errorf("supplier %s not found in session", flagSupplier)
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
		return fmt.Errorf("failed to create signer from hex key: %w", err)
	}
	log.Printf("Created signer for app: %s", flagApp)

	// Connect to WebSocket
	log.Printf("Connecting to WebSocket at %s...", flagRelayerWSURL)

	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}

	// Add headers
	header := http.Header{}
	header.Set("Pocket-Service-Id", serviceID)
	header.Set("Pocket-Supplier-Address", flagSupplier)

	wsConn, _, err := dialer.DialContext(ctx, flagRelayerWSURL, header)
	if err != nil {
		return fmt.Errorf("failed to connect to websocket: %w", err)
	}
	defer wsConn.Close()
	log.Println("Connected to WebSocket")

	// Send messages
	for i := 1; i <= flagNumMessages; i++ {
		log.Printf("Sending message %d/%d...", i, flagNumMessages)

		// Build relay request
		relayReq, err := sdk.BuildRelayRequest(endpoint, []byte(flagPayload))
		if err != nil {
			return fmt.Errorf("failed to build relay request: %w", err)
		}

		// Sign the request
		signedRelayReq, err := appSigner.Sign(ctx, relayReq, &ring)
		if err != nil {
			return fmt.Errorf("failed to sign relay request: %w", err)
		}

		// Marshal to protobuf
		reqBytes, err := signedRelayReq.Marshal()
		if err != nil {
			return fmt.Errorf("failed to marshal relay request: %w", err)
		}

		// Send over WebSocket
		if err := wsConn.WriteMessage(websocket.BinaryMessage, reqBytes); err != nil {
			return fmt.Errorf("failed to send websocket message: %w", err)
		}
		log.Printf("Sent relay request (%d bytes)", len(reqBytes))

		// Read response
		_, respBytes, err := wsConn.ReadMessage()
		if err != nil {
			return fmt.Errorf("failed to read websocket response: %w", err)
		}
		log.Printf("Received response (%d bytes)", len(respBytes))

		// Try to unmarshal as RelayResponse
		relayResp := &servicetypes.RelayResponse{}
		if err := relayResp.Unmarshal(respBytes); err != nil {
			log.Printf("Warning: response is not a RelayResponse: %v", err)
			log.Printf("Raw response: %s", string(respBytes))
		} else {
			log.Printf("RelayResponse session: %s", relayResp.Meta.SessionHeader.SessionId)
			if len(relayResp.Payload) > 0 {
				payloadPreview := string(relayResp.Payload)
				if len(payloadPreview) > 100 {
					payloadPreview = payloadPreview[:100] + "..."
				}
				log.Printf("Payload: %s", payloadPreview)
			}
			if len(relayResp.Meta.SupplierOperatorSignature) > 0 {
				log.Printf("Response is signed by supplier")
			}
		}
	}

	log.Printf("Successfully sent %d WebSocket relay messages", flagNumMessages)
	return nil
}
