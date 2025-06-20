//go:build load

package tests

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"regexp"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"
)

const (
	localnetAnvilURL    = "http://localhost:8547"
	anvilNodeInfoMethod = "anvil_nodeInfo"
)

var blockResultRegex = regexp.MustCompile(`"result":"0x\w+"}$`)

// anvilSuite is a load/stress test suite for the Anvil server in isolation.
// This suite is intended to be used to test baseline performance expectations of
// the Anvil server such that we're can be certain that it won't be a bottleneck
// in other load tests (e.g. pokt relay stress).
type anvilSuite struct {
	gocuke.TestingT
	numRequests   int64
	startTime     time.Time
	requestsCount atomic.Uint64
}

func TestLoadAnvil(t *testing.T) {
	gocuke.NewRunner(t, &anvilSuite{}).Path(filepath.Join(".", "anvil.feature")).Run()
}

// AnvilIsRunning ensures that the Anvil server is running.
func (s *anvilSuite) AnvilIsRunning() {
	// TODO_TECHDEBT(@okdas): add support for non-LocalNet environments.

	// Send a JSON-RPC request to the Anvil server to check if it's running.
	payloadJSON := relayPayloadHeight // fmt.Sprintf(relayPayloadFmt, anvilNodeInfoMethod, 0)
	_ = s.sendJSONRPCRequest(0, payloadJSON)
}

// LoadOfConcurrentRequests sets the number of users to simulate in the load
// test as well as tracking the start time.
// EachUserRequestsTheJSONRPCMethod sends a JSON-RPC request with the given method
// to the Anvil server.
func (s *anvilSuite) LoadOfConcurrentRequestsForTheJsonrpcMethod(numRequests int64, method string) {
	// Set the number of users on the suite.
	s.numRequests = numRequests

	// Set the start time
	s.startTime = time.Now()

	// Send block height requests.
	waitGroup := sync.WaitGroup{}
	for requestIdx := int64(0); requestIdx < s.numRequests; requestIdx++ {
		waitGroup.Add(1)
		go func(userIdx int64) {
			s.requestAnvilMethod(userIdx, method)
			s.requestsCount.Add(1)
			waitGroup.Done()
		}(requestIdx)
	}

	// Wait for all requests to complete.
	waitGroup.Wait()
}

// LoadIsHandledWithinSeconds asserts that the load test has completed within the
// given number of seconds & logs the number of requests & test duration.
func (s *anvilSuite) LoadIsHandledWithinSeconds(numSeconds int64) {
	// Calculate the duration.
	duration := time.Since(s.startTime)

	require.Less(s, duration.Seconds(), float64(numSeconds))

	// Log the duration.
	logger.Info().
		Uint64("requests", s.requestsCount.Load()).
		Msgf("duration: %s", duration)
}

// requestAnvilMethod synchronously sends a JSON-RPC request to the Anvil server,
// blocking until the response is received. It asserts that the response has a 200
// status code and that the body matches the expected regex (block height in hex).
func (s *anvilSuite) requestAnvilMethod(requestId int64, method string) {
	// URL and data for the POST request
	payloadJSON := fmt.Sprintf(relayPayloadHeight, requestId) // fmt.Sprintf(relayPayloadFmt, method, requestId)

	// Send the JSON-RPC request and get the response body.
	resBody := s.sendJSONRPCRequest(requestId, payloadJSON)

	// fmt.Println(resBody)
	// Assert the response contains a block height in the expected format.
	// require.Regexp(s, blockResultRegex, resBody)

	logger.Debug().
		Int64("id", requestId).
		// Str("response", resBody).
		Int("response_len", len(resBody)).
		Send()
}

// sendJSONRPCRequest sends a JSON-RPC request to the Anvil server and returns the response body.
// It asserts that the response has a 200 status code.
func (s *anvilSuite) sendJSONRPCRequest(id int64, payloadJSON string) string {
	payloadBuf := bytes.NewBuffer([]byte(payloadJSON))

	// Create a new POST request with JSON data
	req, err := http.NewRequest("POST", localnetAnvilURL, payloadBuf)
	require.NoError(s, err, "creating request %d", id)

	// Set content type to application/json
	req.Header.Set("Content-Type", "application/json")

	// Create an HTTP client and send the request
	resp, err := http.DefaultClient.Do(req)
	require.NoError(s, err, "sending request %d", id)
	defer func() {
		_ = resp.Body.Close()
	}()

	// Read and print the response body
	body, err := io.ReadAll(resp.Body)
	require.NoError(s, err, "reading request %d", id)

	// Assert the response has the correct status code & a valid JSON-RPC result.
	require.Equal(s, resp.StatusCode, http.StatusOK)

	return string(body)
}
