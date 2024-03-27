package tests

import (
	"bytes"
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

const localnetAnvilURL = "http://localhost:8547"

var blockResultRegex = regexp.MustCompile(`"result":"0x\w+"}$`)

type anvilSuite struct {
	gocuke.TestingT
	numUsers      int64
	startTime     time.Time
	requestsCount atomic.Uint64
}

func TestLoadAnvil(t *testing.T) {
	gocuke.NewRunner(t, &anvilSuite{}).Path(filepath.Join(".", "anvil.feature")).Run()
}

func (s *anvilSuite) AnvilIsRunning() {
	// TODO_TECHDEBT: add support for non-localnet environments.
}

func (s *anvilSuite) LoadOfConcurrentUsers(numUsers int64) {
	// Set the number of users on the suite.
	s.numUsers = numUsers

	// Set the start time
	s.startTime = time.Now()
}

func (s *anvilSuite) EachUserRequestsTheEthereumBlockHeight() {
	// Request block height for each user.
	waitGroup := sync.WaitGroup{}
	for userIdx := int64(0); userIdx < s.numUsers; userIdx++ {
		waitGroup.Add(1)
		go func(userIdx int64) {
			s.requestAnvilBlockHeight(userIdx)
			s.requestsCount.Add(1)
			waitGroup.Done()
		}(userIdx)
	}

	// Wait for logging to complete.
	waitGroup.Wait()
}

func (s *anvilSuite) LoadIsHandledWithinSeconds(numSeconds int64) {
	// Calculate the duration.
	duration := time.Now().Sub(s.startTime)

	require.Less(s, duration.Seconds(), float64(numSeconds))

	// Log the duration.
	logger.Info().
		Uint64("requests", s.requestsCount.Load()).
		Msgf("duration: %s", duration)
}

func (s *anvilSuite) requestAnvilBlockHeight(userIdx int64) {
	// URL and data for the POST request
	jsonData := bytes.NewBuffer([]byte(`{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`))

	// Create a new POST request with JSON data
	req, err := http.NewRequest("POST", localnetAnvilURL, jsonData)
	if err != nil {
		logger.Error().
			Err(err).
			Int64("user", userIdx).
			Msg("creating request")
		return
	}

	// Set content type to application/json
	req.Header.Set("Content-Type", "application/json")

	// Create an HTTP client and send the request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		logger.Error().
			Err(err).
			Int64("user", userIdx).
			Msg("sending request")
		return
	}
	defer resp.Body.Close()

	// Read and print the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error().
			Err(err).
			Int64("user", userIdx).
			Msg("reading response")
		return
	}

	// Assert the response has the correct status code & a valid JSON-RPC result.
	require.Equal(s, resp.StatusCode, http.StatusOK)
	require.Regexp(s, blockResultRegex, string(body))

	logger.Debug().
		Int64("user", userIdx).
		Str("response", string(body)).
		Send()
}
