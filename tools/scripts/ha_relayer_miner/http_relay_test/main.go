// ha_load_test.go - Tool for sending multiple pre-signed relay requests to HA RelayMiner
//
// This tool generates a properly formatted RelayRequest once, then sends it
// many times to the HA RelayMiner for testing claim submission flows.
//
// Usage:
//   go run tools/scripts/ha_load_test/main.go [flags]
//
// Flags:
//   -n int           Number of requests to send (default 200)
//   -concurrency int Number of concurrent workers (default 10)
//   -target string   Target URL of the HA Relayer (default "http://localhost:8080")
//   -grpc string     gRPC endpoint for session queries (default "sauron-grpc.beta.infra.pocket.network:443")
//   -service string  Service ID to test (default "ha-relayminer-develop")
//   -app string      Application address
//   -app-key string  Application private key hex

package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"flag"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"sync"
	"sync/atomic"
	"time"

	sdk "github.com/pokt-network/shannon-sdk"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	numRequests     = flag.Int("n", 200, "Number of requests to send")
	concurrency     = flag.Int("concurrency", 10, "Number of concurrent workers")
	targetURL       = flag.String("target", "http://localhost:8080", "Target URL of the HA Relayer")
	grpcEndpoint    = flag.String("grpc", "sauron-grpc.beta.infra.pocket.network:443", "gRPC endpoint for session queries")
	grpcInsecure    = flag.Bool("grpc-insecure", false, "Use insecure gRPC connection")
	serviceID       = flag.String("service", "ha-relayminer-develop", "Service ID to test")
	appAddress      = flag.String("app", "pokt1ps5vduype4u6yy0unukyr4wucntj2g503l0kau", "Application address")
	appPrivKeyHex   = flag.String("app-key", "", "Application private key hex (required)")
	relayPayload    = flag.String("payload", `{"jsonrpc":"2.0","method":"test","params":[],"id":1}`, "Relay payload")
)

func main() {
	flag.Parse()

	ctx := context.Background()

	if *appPrivKeyHex == "" {
		fmt.Println("❌ Error: --app-key is required (application private key hex)")
		fmt.Println("Usage: go run tools/scripts/ha_load_test/main.go --app-key=<hex>")
		os.Exit(1)
	}

	fmt.Println("=== HA RelayMiner Load Test ===")
	fmt.Printf("Target: %s\n", *targetURL)
	fmt.Printf("Requests: %d\n", *numRequests)
	fmt.Printf("Concurrency: %d\n", *concurrency)
	fmt.Printf("Service: %s\n", *serviceID)
	fmt.Printf("App: %s\n", *appAddress)
	fmt.Println()

	// Step 1: Generate a valid, signed RelayRequest
	fmt.Println("Generating signed RelayRequest...")
	relayReqBz, sessionInfo, err := generateRelayRequest(ctx)
	if err != nil {
		fmt.Printf("❌ Error generating relay request: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✅ Generated RelayRequest (%d bytes)\n", len(relayReqBz))
	fmt.Printf("   Session ID: %s\n", sessionInfo.sessionID)
	fmt.Printf("   Session Start: %d\n", sessionInfo.startHeight)
	fmt.Printf("   Session End: %d\n", sessionInfo.endHeight)
	fmt.Println()

	// Step 2: Send requests
	fmt.Printf("Sending %d requests with %d workers...\n", *numRequests, *concurrency)
	startTime := time.Now()

	var successCount, errorCount int64
	var wg sync.WaitGroup
	requestCh := make(chan int, *numRequests)

	// Start workers
	for i := 0; i < *concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			client := &http.Client{Timeout: 30 * time.Second}

			for reqNum := range requestCh {
				err := sendRelay(client, relayReqBz)
				if err != nil {
					atomic.AddInt64(&errorCount, 1)
					if atomic.LoadInt64(&errorCount) <= 5 {
						fmt.Printf("❌ Request %d failed: %v\n", reqNum, err)
					}
				} else {
					count := atomic.AddInt64(&successCount, 1)
					if count%50 == 0 {
						fmt.Printf("✅ Sent %d/%d requests\n", count, *numRequests)
					}
				}
			}
		}(i)
	}

	// Queue requests
	for i := 1; i <= *numRequests; i++ {
		requestCh <- i
	}
	close(requestCh)

	// Wait for completion
	wg.Wait()
	elapsed := time.Since(startTime)

	fmt.Println()
	fmt.Println("=== Results ===")
	fmt.Printf("Total requests: %d\n", *numRequests)
	fmt.Printf("Successful: %d\n", successCount)
	fmt.Printf("Failed: %d\n", errorCount)
	fmt.Printf("Duration: %v\n", elapsed)
	fmt.Printf("Throughput: %.2f req/s\n", float64(*numRequests)/elapsed.Seconds())
	fmt.Println()
	fmt.Println("Check miner logs and Redis for session state.")
}

type sessionInfo struct {
	sessionID   string
	startHeight int64
	endHeight   int64
}

func generateRelayRequest(ctx context.Context) ([]byte, *sessionInfo, error) {
	// Connect to gRPC
	var grpcConn *grpc.ClientConn
	var err error

	if *grpcInsecure {
		grpcConn, err = grpc.NewClient(*grpcEndpoint,
			grpc.WithTransportCredentials(insecure.NewCredentials()))
	} else {
		grpcConn, err = grpc.NewClient(*grpcEndpoint,
			grpc.WithTransportCredentials(credentials.NewTLS(&tls.Config{})))
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect to gRPC: %w", err)
	}
	defer grpcConn.Close()

	// Get current session
	sessionClient := sdk.SessionClient{
		PoktNodeSessionFetcher: sdk.NewPoktNodeSessionFetcher(grpcConn),
	}
	session, err := sessionClient.GetSession(ctx, *appAddress, *serviceID, 0)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get session: %w", err)
	}

	info := &sessionInfo{
		sessionID:   session.SessionId,
		startHeight: session.Header.SessionStartBlockHeight,
		endHeight:   session.Header.SessionEndBlockHeight,
	}

	// Create an account client for fetching public keys
	accountClient := sdk.AccountClient{
		PoktNodeAccountFetcher: sdk.NewPoktNodeAccountFetcher(grpcConn),
	}

	// Create an application ring for signing
	ring := &sdk.ApplicationRing{
		Application:      *session.Application,
		PublicKeyFetcher: &accountClient,
	}

	// Select an endpoint from the session
	sessionFilter := sdk.SessionFilter{
		Session:         session,
		EndpointFilters: []sdk.EndpointFilter{},
	}
	endpoints, err := sessionFilter.FilteredEndpoints()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to filter endpoints: %w", err)
	}
	if len(endpoints) == 0 {
		return nil, nil, fmt.Errorf("no endpoints available in session")
	}

	endpoint := endpoints[rand.Intn(len(endpoints))]
	endpointUrl := endpoint.Endpoint().Url

	// Initialize the signer from the private key hex
	appSigner, err := sdk.NewSignerFromHex(*appPrivKeyHex)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create signer: %w", err)
	}

	// Parse and modify the URL to use our target
	reqUrl, err := url.Parse(*targetURL)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to parse target URL: %w", err)
	}

	// Prepare the JSON-RPC request payload
	body := io.NopCloser(bytes.NewReader([]byte(*relayPayload)))
	httpReq, err := http.NewRequest(http.MethodPost, endpointUrl, body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create HTTP request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	_, payloadBz, err := sdktypes.SerializeHTTPRequest(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to serialize HTTP request: %w", err)
	}

	// Build relay request
	relayReq, err := sdk.BuildRelayRequest(endpoint, payloadBz)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to build relay request: %w", err)
	}

	// Sign the relay request
	signedRelayReq, err := appSigner.Sign(ctx, relayReq, ring)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to sign relay request: %w", err)
	}

	// Marshal
	relayReqBz, err := signedRelayReq.Marshal()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to marshal relay request: %w", err)
	}

	_ = reqUrl // Silence unused variable warning

	return relayReqBz, info, nil
}

func sendRelay(client *http.Client, relayReqBz []byte) error {
	resp, err := client.Post(*targetURL, "application/octet-stream", bytes.NewReader(relayReqBz))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read and discard response
	_, _ = io.Copy(io.Discard, resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	return nil
}
