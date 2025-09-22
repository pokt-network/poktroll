// generate_relay.go - Tool for generating RelayRequest data and running load tests
//
// This tool generates a properly formatted RelayRequest, creates a Lua script for wrk2,
// and executes load testing against a RelayMiner endpoint using kubectl.
//
// Usage:
//   go run tools/scripts/generate_relay/main.go [flags]
//
// Flags:
//   -R int     Number of requests per second (default 512)
//   -d string  Duration of the test (default "300s")
//   -t int     Number of threads to use (default 16)
//   -c int     Number of connections to keep open (default 256)

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
	"os/exec"
	"strings"

	sdk "github.com/pokt-network/shannon-sdk"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/pokt-network/poktroll/pkg/polylog"
	sdktypes "github.com/pokt-network/shannon-sdk/types"
)

// Test configuration constants for relay generation and load testing.
// These values configure the blockchain connection and test relay parameters.
const (
	// nodeGRPCURL is the gRPC endpoint for connecting to the Poktroll node
	nodeGRPCURL = "localhost:9090"
	// serviceId identifies the service type for the relay request
	serviceId = "static"
	// appAddress is the Pokt application address used for relay authentication
	appAddress = "pokt1pn64d94e6u5g8cllsnhgrl6t96ysnjw59j5gst"
	// appPrivateKeyHex is the private key used to sign relay requests
	appPrivateKeyHex = "84e4f2257f24d9e1517d414b834bbbfa317e0d53fef21c1528a07a5fa8c70d57"
	// relayPayload is the JSON-RPC payload sent in each relay request
	relayPayload = `{"jsonrpc":"2.0","method":"eth_chainId","params":[],"id":1}`
)

// Command-line flags for configuring the wrk2 load test parameters.
var (
	// requestRate controls the number of requests per second for the load test
	requestRate = flag.Int("R", 512, "Number of requests per second")
	// duration specifies how long the load test should run
	duration = flag.String("d", "300s", "Duration of the test")
	// threads sets the number of worker threads for wrk2
	threads = flag.Int("t", 16, "Number of threads to use")
	// connections controls the number of concurrent connections to maintain
	connections = flag.Int("c", 256, "Number of connections to keep open")
)

// main orchestrates the relay load testing process by generating a RelayRequest,
// creating a wrk2 Lua script, and executing the load test.
func main() {
	flag.Parse()

	ctx := context.Background()
	logger := polylog.Ctx(ctx)

	// Step 1: Generate a valid, signed RelayRequest for load testing
	relayReqBz, err := generateRelayRequest(ctx, logger)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error generating relay request")
		os.Exit(1)
	}

	logger.Info().Msgf("✅ Generated RelayRequest (%d bytes)", len(relayReqBz))

	// ⚠️ IMPORTANT SESSION WARNING ⚠️
	fmt.Println("\n🔔 IMPORTANT: Session Validity Warning")
	fmt.Println("=======================================")
	fmt.Println("* The RelayRequest was generated for the CURRENT Session")
	fmt.Println("* Sessions change periodically (typically every few blocks)")
	fmt.Println("* If the session changes during your load test, the RelayRequest will become INVALID")
	fmt.Println("* This will cause ALL requests in the load test to FAIL validation after the session expires")
	fmt.Println("")
	fmt.Println("📋 RECOMMENDATIONS:")
	fmt.Println("* Locally set block time to 30s in config.yaml to SLOW session changes")
	fmt.Println("* Start your load test at the BEGINNING of a new Session")
	fmt.Println("* Keep test duration SHORTER than the session length")
	fmt.Println("* Monitor for session changes if running longer tests")
	fmt.Println("* If the RelayMiner starts rejecting RelayRequests, stop the test and try again")
	fmt.Println("=======================================")

	// Step 2: Create and deploy Lua script to wrk2 pod
	if err := createLuaScript(relayReqBz); err != nil {
		logger.Error().Err(err).Msg("❌ Error creating Lua script")
		os.Exit(1)
	}

	logger.Info().Msg("✅ Created wrk Lua script")

	// Step 3: Execute the wrk2 load test
	if err := executeWrkCommand(); err != nil {
		logger.Error().Err(err).Msg("❌ Error executing wrk command")
		os.Exit(1)
	}

	fmt.Println("✅ Load test completed")
}

// generateRelayRequest creates a valid, signed RelayRequest using real session data
// from the blockchain. It connects to the Poktroll node, fetches an active session,
// builds a properly formatted relay request, and signs it with the application's
// private key. This ensures the generated RelayRequest matches production format.
func generateRelayRequest(ctx context.Context, logger polylog.Logger) ([]byte, error) {

	// Initialize gRPC connection
	grpcConn, err := connectGRPC(GRPCConfig{
		HostPort: nodeGRPCURL,
		Insecure: true,
	})
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error connecting to gRPC")
		return nil, err
	}
	defer grpcConn.Close()
	logger.Info().Msgf("✅ gRPC connection initialized: %s", nodeGRPCURL)

	// Get the current session
	sessionClient := sdk.SessionClient{
		PoktNodeSessionFetcher: sdk.NewPoktNodeSessionFetcher(grpcConn),
	}
	session, err := sessionClient.GetSession(ctx, appAddress, serviceId, 0)
	if err != nil {
		logger.Error().Err(err).Msgf("❌ Error fetching session for app %s and service ID %s", appAddress, serviceId)
		return nil, err
	}
	logger.Info().Msgf("✅ Session with id %s fetched for app %s and service ID %s with %d suppliers", session.SessionId, appAddress, serviceId, len(session.Suppliers))

	// Create an account client for fetching public keys
	accountClient := sdk.AccountClient{
		PoktNodeAccountFetcher: sdk.NewPoktNodeAccountFetcher(grpcConn),
	}
	logger.Info().Msg("✅ Account client initialized")

	// Create an application ring for signing
	ring := sdk.ApplicationRing{
		Application:      *session.Application,
		PublicKeyFetcher: &accountClient,
	}
	logger.Info().Msg("✅ Application ring created")

	// Select an endpoint from the session
	sessionFilter := sdk.SessionFilter{
		Session:         session,
		EndpointFilters: []sdk.EndpointFilter{},
	}
	endpoints, err := sessionFilter.FilteredEndpoints()
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error filtering endpoints")
		return nil, err
	}
	if len(endpoints) == 0 {
		logger.Error().Msg("❌ No endpoints available")
		return nil, fmt.Errorf("no endpoints available in the current session")
	}
	logger.Info().Msgf("✅ %d endpoints fetched", len(endpoints))

	endpoint := endpoints[rand.Intn(len(endpoints))]
	logger.Info().Msgf("✅ No supplier specified, randomly selected endpoint: %v", endpoint)

	// Get the endpoint URL
	endpointUrl := endpoint.Endpoint().Url

	appSigner := sdk.Signer{PrivateKeyHex: appPrivateKeyHex}

	// Parse the endpoint URL
	reqUrl, err := url.Parse(endpointUrl)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error parsing endpoint URL")
		return nil, err
	}
	logger.Info().Msgf("✅ Endpoint URL parsed: %v", reqUrl)

	// Prepare the JSON-RPC request payload
	body := io.NopCloser(bytes.NewReader([]byte(relayPayload)))
	jsonRpcServiceReq, err := http.NewRequest(http.MethodPost, endpointUrl, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new HTTP request for url %s: %w", endpointUrl, err)
	}
	jsonRpcServiceReq.Header.Set("Content-Type", "application/json")
	_, payloadBz, err := sdktypes.SerializeHTTPRequest(jsonRpcServiceReq)

	if err != nil {
		return nil, fmt.Errorf("failed to Serialize HTTP Request for URL %s: %w", endpointUrl, err)
	}
	logger.Info().Msg("✅ JSON-RPC request payload serialized.")

	// Build a relay request
	relayReq, err := sdk.BuildRelayRequest(endpoint, payloadBz)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error building relay request")
		return nil, err
	}
	logger.Info().Msg("✅ Relay request built.")

	signedRelayReq, err := appSigner.Sign(ctx, relayReq, ring)
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error signing relay request")
		return nil, err
	}
	logger.Info().Msg("✅ Relay request signed.")

	// Marshal the signed relay request
	relayReqBz, err := signedRelayReq.Marshal()
	if err != nil {
		logger.Error().Err(err).Msg("❌ Error marshaling relay request")
		return nil, err
	}
	logger.Info().Msg("✅ Relay request marshaled.")

	return relayReqBz, nil
}

// createLuaScript generates a wrk2 Lua script containing the binary RelayRequest data
// and deploys it to the wrk2 pod. The script formats the relay bytes as a Lua string
// with proper escaping, then copies the script to the pod's /tmp directory for use
// during load testing.
func createLuaScript(relayReqBz []byte) error {
	// Format relayReqBz as Lua string literal with escaped bytes
	var luaString strings.Builder
	luaString.WriteString("\"")
	for _, b := range relayReqBz {
		luaString.WriteString(fmt.Sprintf("\\%d", b))
	}
	luaString.WriteString("\"")

	luaScript := fmt.Sprintf(`-- wrk2 Lua script for RelayRequest load testing
-- RelayRequest data as binary string
local relay_data = %s

-- Headers
local headers = {}
headers["Content-Type"] = "application/json"

-- Request function
function request()
    return wrk.format("POST", nil, headers, relay_data)
end
`, luaString.String())

	// Create Lua script in temp directory
	tempDir := os.TempDir()
	luaFilePath := fmt.Sprintf("%s/wrk_relay.lua", tempDir)

	if err := os.WriteFile(luaFilePath, []byte(luaScript), 0644); err != nil {
		return fmt.Errorf("failed to write Lua script to temp dir: %w", err)
	}
	defer os.Remove(luaFilePath)

	// Identify the actual pod name for the wrk2 deployment
	getPodCmd := exec.Command("kubectl", "get", "pods", "-l", "app=wrk2", "-o", "jsonpath={.items[0].metadata.name}")
	podNameBytes, err := getPodCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get wrk2 pod name: %w", err)
	}

	podName := string(podNameBytes)
	if podName == "" {
		return fmt.Errorf("no wrk2 pod found")
	}

	// Copy the Lua script to the specific pod's /tmp directory
	copyCmd := exec.Command("kubectl", "cp", luaFilePath, fmt.Sprintf("%s:/tmp/wrk_relay.lua", podName))
	copyCmd.Stdout = os.Stdout
	copyCmd.Stderr = os.Stderr

	if err := copyCmd.Run(); err != nil {
		return fmt.Errorf("failed to copy Lua script to wrk2 pod: %w", err)
	}

	return nil
}

// executeWrkCommand executes the wrk2 load test against the RelayMiner endpoint.
// It builds the command arguments from the configured flags and runs wrk2 inside
// the Kubernetes cluster using the previously deployed Lua script.
func executeWrkCommand() error {
	// Build wrk2 command arguments
	args := []string{"exec", "deployment/wrk2", "--", "wrk", "-R", fmt.Sprintf("%d", *requestRate), "-L"}
	args = append(args, "-d", *duration, "-t", fmt.Sprintf("%d", *threads), "-c", fmt.Sprintf("%d", *connections),
		"-s", "/tmp/wrk_relay.lua", "http://relayminer1:8545/")

	// Execute wrk2 command
	wrkCmd := exec.Command("kubectl", args...)

	wrkCmd.Stdout = os.Stdout
	wrkCmd.Stderr = os.Stderr

	if err := wrkCmd.Run(); err != nil {
		return fmt.Errorf("wrk2 command failed: %w", err)
	}

	return nil
}

// GRPCConfig holds configuration options for establishing a gRPC connection.
//
// Fields:
// - HostPort: gRPC host:port string
// - Insecure: Use insecure credentials
type GRPCConfig struct {
	HostPort string `yaml:"host_port"`
	Insecure bool   `yaml:"insecure"`
}

// connectGRPC establishes a gRPC client connection using the provided GRPCConfig.
//
// - Returns a grpc.ClientConn or error
// - Uses insecure credentials if config.Insecure is true
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
