//go:build load

package tests

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"cosmossdk.io/math"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/load-testing/config"
	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/query"
	querycache "github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog"
	"github.com/pokt-network/poktroll/testutil/testclient/testtx"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

const (
	claimsNewBlockEventQuery = "tm.event='NewBlockHeader'"
	claimsRelayRequestMethod = "eth_blockNumber"
	claimsRelayPayloadFmt    = `{"jsonrpc":"2.0","method":"%s","params":[],"id":%d}`
)

type ClaimsTestConfig struct {
	NumApplications  int
	ServiceID        string
	GatewayURL       string
	GatewayAddress   string
	StakeAmount      int64
	DurationSessions int
	FundingAccount   string
	RPCNode          string
	GRPCAddr         string
	ChainID          string
	// Loaded from manifest
	Manifest *config.LoadTestManifestYAML
}

type Application struct {
	Address    string
	PrivateKey *secp256k1.PrivKey
	KeyName    string
}

type ClaimsStressTester struct {
	t                  *testing.T
	config             *ClaimsTestConfig
	logger             polylog.Logger
	txContext          client.TxContext
	applications       []*Application
	eventsReplayClient client.EventsReplayClient[client.Block]
	sharedQuerier      client.SharedQueryClient
	sharedParams       *sharedtypes.Params
	latestBlock        client.Block
	requestsSent       int64
	currentSession     int64
	sessionStarted     bool
	ctx                context.Context
	cancel             context.CancelFunc
}

func TestMaximizeClaims(t *testing.T) {
	// Load configuration from load test manifest
	manifest := initializeLoadTestManifest(t)

	// Skip test if no gateways available
	if len(manifest.Gateways) == 0 {
		t.Skip("No gateways available in load test manifest")
	}

	// Use first gateway from manifest
	firstGateway := manifest.Gateways[0]

	config := &ClaimsTestConfig{
		NumApplications:  3000, // Smaller number for faster testing
		DurationSessions: 5,
		ServiceID:        manifest.ServiceId,
		GatewayURL:       firstGateway.ExposedUrl,
		GatewayAddress:   firstGateway.Address,
		StakeAmount:      100000000,
		FundingAccount:   manifest.FundingAccountAddress, // Address from manifest
		RPCNode:          manifest.RPCNode,
		Manifest:         manifest,
	}

	tester := &ClaimsStressTester{
		t:      t,
		config: config,
		logger: logger, // Use the global logger from init_load_test.go
	}

	// Setup cleanup to delete application keys on test completion or failure
	t.Cleanup(func() {
		tester.cleanup()
	})

	err := tester.Run()
	require.NoError(t, err)

	t.Logf("Claims stress test completed successfully. Total requests sent: %d", tester.requestsSent)
}

func (ct *ClaimsStressTester) Run() error {
	ct.ctx, ct.cancel = context.WithCancel(context.Background())
	defer ct.cancel()

	ct.logger.Info().Msg("Starting claims stress test")

	// Setup blockchain client context
	if err := ct.setupClients(); err != nil {
		return fmt.Errorf("failed to setup clients: %w", err)
	}

	// Create applications
	if err := ct.createApplications(); err != nil {
		return fmt.Errorf("failed to create applications: %w", err)
	}

	// Fund applications
	if err := ct.fundApplications(); err != nil {
		return fmt.Errorf("failed to fund applications: %w", err)
	}

	// Stake applications
	if err := ct.stakeAndDelegateApplications(); err != nil {
		return fmt.Errorf("failed to stake applications: %w", err)
	}

	// Wait for applications to become active in next session
	//if err := ct.waitForNextSession(); err != nil {
	//	return fmt.Errorf("failed waiting for next session: %w", err)
	//}

	// Start sending requests distributed across session
	if err := ct.sendRequestsDistributed(); err != nil {
		return fmt.Errorf("failed to send requests: %w", err)
	}

	ct.logger.Info().
		Int64("total_requests", ct.requestsSent).
		Msg("Claims stress test completed successfully")

	return nil
}

func (ct *ClaimsStressTester) setupClients() error {
	// Setup transaction context using test helpers
	ct.txContext = testtx.NewLocalnetContext(ct.t)

	var err error

	// Set up events replay client using block client (same pattern as relays_stress_helpers_test.go)

	cometClient, err := sdkclient.NewClientFromNode(ct.config.RPCNode)
	require.NoError(ct.t, err)

	err = cometClient.Start()
	require.NoError(ct.t, err)

	// Create deps for events replay client
	eventsDeps := depinject.Supply(
		cometClient,
		logger,
	)

	ct.eventsReplayClient, err = events.NewEventsReplayClient(
		ct.ctx,
		eventsDeps,
		claimsNewBlockEventQuery,
		block.UnmarshalNewBlock,
		10, // replay limit
	)
	require.NoError(ct.t, err)

	// Set up shared query client using the same pattern as relays_stress_helpers_test.go
	sharedParamsCache := querycache.NewNoOpParamsCache[sharedtypes.Params]()
	blockhashCache := querycache.NewNoOpKeyValueCache[query.BlockHash]()
	deps := depinject.Supply(
		ct.txContext.GetClientCtx(),
		logger,
		ct.eventsReplayClient,
		sharedParamsCache,
		blockhashCache,
	)

	blockQueryClient, err := sdkclient.NewClientFromNode(ct.config.RPCNode)
	require.NoError(ct.t, err)
	deps = depinject.Configs(deps, depinject.Supply(blockQueryClient))

	ct.sharedQuerier, err = query.NewSharedQuerier(deps)
	require.NoError(ct.t, err)

	// Query shared parameters from the blockchain
	ct.sharedParams, err = ct.sharedQuerier.GetParams(ct.ctx)
	if err != nil {
		return fmt.Errorf("failed to get shared params: %w", err)
	}

	ct.logger.Info().
		Uint64("num_blocks_per_session", ct.sharedParams.GetNumBlocksPerSession()).
		Msg("Retrieved shared parameters from blockchain")

	// Start monitoring blocks
	go ct.monitorBlocks()

	// Wait for first block event
	for ct.latestBlock == nil {
		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func (ct *ClaimsStressTester) monitorBlocks() {
	// Use events replay client to monitor blocks (same pattern as relays_stress_test.go)
	ct.logger.Info().Msg("Starting to monitor block events")

	channel.ForEach(
		ct.ctx,
		ct.eventsReplayClient.EventsSequence(ct.ctx),
		func(ctx context.Context, blockEvent client.Block) {
			blockHeight := blockEvent.Height()

			// Update latest block
			ct.latestBlock = blockEvent

			// Calculate session information using shared params
			sessionNumber := sharedtypes.GetSessionNumber(ct.sharedParams, blockHeight)
			sessionStartHeight := sharedtypes.GetSessionStartHeight(ct.sharedParams, blockHeight)

			// Check if we're at session start
			if blockHeight == sessionStartHeight {
				prevSession := ct.currentSession
				ct.currentSession = sessionNumber
				ct.sessionStarted = true

				ct.logger.Info().
					Int64("block_height", blockHeight).
					Int64("session_number", sessionNumber).
					Int64("session_start_height", sessionStartHeight).
					Int64("prev_session", prevSession).
					Msg("New session started")
			}

			ct.logger.Debug().
				Int64("block_height", blockHeight).
				Int64("session_number", sessionNumber).
				Msg("Received block event")
		},
	)
}

func (ct *ClaimsStressTester) createApplications() error {
	ct.logger.Info().
		Int("num_apps", ct.config.NumApplications).
		Msg("Creating applications")

	ct.applications = make([]*Application, ct.config.NumApplications)

	for i := 0; i < ct.config.NumApplications; i++ {
		// Generate new private key
		privKey := secp256k1.GenPrivKey()
		keyName := fmt.Sprintf("claimstest-app-%d", i+1)

		// Check if key already exists and delete it first (cleanup from previous test runs)
		if existingKey, err := ct.txContext.GetKeyring().Key(keyName); err == nil {
			existingAddr, _ := existingKey.GetAddress()
			ct.txContext.GetKeyring().DeleteByAddress(existingAddr)
		}

		// Import key into keyring
		privKeyHex := fmt.Sprintf("%x", privKey)
		err := ct.txContext.GetKeyring().ImportPrivKeyHex(keyName, privKeyHex, "secp256k1")
		if err != nil {
			return fmt.Errorf("failed to import private key for app %d: %w", i+1, err)
		}

		// Get address
		keyRecord, err := ct.txContext.GetKeyring().Key(keyName)
		if err != nil {
			return fmt.Errorf("failed to get key record for app %d: %w", i+1, err)
		}

		address, err := keyRecord.GetAddress()
		if err != nil {
			return fmt.Errorf("failed to get address for app %d: %w", i+1, err)
		}

		ct.applications[i] = &Application{
			Address:    address.String(),
			PrivateKey: privKey,
			KeyName:    keyName,
		}

		ct.logger.Debug().
			Str("address", address.String()).
			Str("key_name", keyName).
			Msgf("Created application %d", i+1)
	}

	return nil
}

func (ct *ClaimsStressTester) fundApplications() error {
	ct.logger.Info().Msg("Funding applications")

	// Calculate funding amount (stake + some extra for fees)
	fundingAmount := sdk.NewCoin("upokt", math.NewInt(ct.config.StakeAmount*2))

	// Get funding account using address from manifest (same pattern as relays stress test)
	fundingAccAddress, err := sdk.AccAddressFromBech32(ct.config.FundingAccount)
	if err != nil {
		return fmt.Errorf("failed to parse funding account address: %w", err)
	}

	fundingKey, err := ct.txContext.GetKeyring().KeyByAddress(fundingAccAddress)
	if err != nil {
		return fmt.Errorf("failed to get funding account key: %w", err)
	}

	fundingAddr, err := fundingKey.GetAddress()
	if err != nil {
		return fmt.Errorf("failed to get funding address: %w", err)
	}

	// Create funding transaction for all applications in a single transaction
	var msgs []sdk.Msg
	for _, app := range ct.applications {
		appAddr := sdk.MustAccAddressFromBech32(app.Address)

		// Create bank send message for each application
		msgs = append(msgs, &banktypes.MsgSend{
			FromAddress: fundingAddr.String(),
			ToAddress:   appAddr.String(),
			Amount:      sdk.NewCoins(fundingAmount),
		})
	}

	// Build and send single transaction with all funding messages
	txBuilder := ct.txContext.NewTxBuilder()
	err = txBuilder.SetMsgs(msgs...)
	if err != nil {
		return fmt.Errorf("failed to set messages for funding transaction: %w", err)
	}

	// Set gas limit higher for multiple messages
	txBuilder.SetGasLimit(200000 * uint64(len(ct.applications)))
	if ct.latestBlock != nil {
		txBuilder.SetTimeoutHeight(uint64(ct.latestBlock.Height() + 10))
	}

	// Sign transaction using funding account key name
	err = ct.txContext.SignTx(fundingKey.Name, txBuilder, false, false, false)
	if err != nil {
		return fmt.Errorf("failed to sign funding transaction: %w", err)
	}

	// Broadcast transaction
	txBytes, err := ct.txContext.EncodeTx(txBuilder)
	if err != nil {
		return fmt.Errorf("failed to encode funding transaction: %w", err)
	}

	txResp, err := ct.txContext.BroadcastTx(txBytes)
	if err != nil {
		return fmt.Errorf("failed to broadcast funding transaction: %w", err)
	}

	// Check if transaction was successful
	if txResp.Code != 0 {
		return fmt.Errorf("funding transaction failed with code %d: %s", txResp.Code, txResp.RawLog)
	}

	ct.logger.Info().
		Int("num_applications", len(ct.applications)).
		Str("funding_amount", fundingAmount.String()).
		Str("tx_hash", txResp.TxHash).
		Uint32("tx_code", txResp.Code).
		Msg("All applications funded successfully in single transaction")

	// Wait for funding transaction to be processed
	time.Sleep(30 * time.Second)

	return nil
}

func (ct *ClaimsStressTester) stakeAndDelegateApplications() error {
	ct.logger.Info().Msg("Staking applications")

	stakeAmount := sdk.NewCoin("upokt", math.NewInt(ct.config.StakeAmount))
	serviceConfig := []*sharedtypes.ApplicationServiceConfig{
		{ServiceId: ct.config.ServiceID},
	}

	ct.logger.Info().
		Str("stake_amount", stakeAmount.String()).
		Str("service_id", ct.config.ServiceID).
		Int("num_applications", len(ct.applications)).
		Msg("Starting to stake applications with configuration")

	for i, app := range ct.applications {
		ct.logger.Debug().
			Str("app_address", app.Address).
			Str("key_name", app.KeyName).
			Msgf("Processing application %d for staking", i+1)

		// Create stake message
		stakeMsg := apptypes.NewMsgStakeApplication(
			app.Address,
			stakeAmount,
			serviceConfig,
		)

		ct.logger.Debug().
			Str("app_address", app.Address).
			Str("stake_amount", stakeAmount.String()).
			Str("service_id", ct.config.ServiceID).
			Msgf("Creating stake message for application %d", i+1)

		msgs := []sdk.Msg{
			stakeMsg,
			apptypes.NewMsgDelegateToGateway(
				app.Address,
				ct.config.GatewayAddress,
			),
		}

		// Build transaction
		txBuilder := ct.txContext.NewTxBuilder()
		err := txBuilder.SetMsgs(msgs...)
		if err != nil {
			return fmt.Errorf("failed to set messages for staking app %d: %w", i+1, err)
		}

		txBuilder.SetGasLimit(200000)
		if ct.latestBlock != nil {
			txBuilder.SetTimeoutHeight(uint64(ct.latestBlock.Height() + 10))
		}

		// Sign transaction
		err = ct.txContext.SignTx(app.KeyName, txBuilder, false, false, false)
		if err != nil {
			return fmt.Errorf("failed to sign staking tx for app %d: %w", i+1, err)
		}

		// Broadcast transaction
		txBytes, err := ct.txContext.EncodeTx(txBuilder)
		if err != nil {
			return fmt.Errorf("failed to encode staking tx for app %d: %w", i+1, err)
		}

		txResp, err := ct.txContext.BroadcastTx(txBytes)
		if err != nil {
			return fmt.Errorf("failed to broadcast staking tx for app %d: %w", i+1, err)
		}

		// Check if transaction was successful
		if txResp.Code != 0 {
			return fmt.Errorf("staking tx for app %d failed with code %d: %s", i+1, txResp.Code, txResp.RawLog)
		}
	}

	// Wait for staking transactions to be processed (increased wait time)
	ct.logger.Info().Msg("Waiting for staking transactions to be processed...")
	time.Sleep(30 * time.Second)

	return nil
}

func (ct *ClaimsStressTester) sendRequestsDistributed() error {
	// Calculate requests per session based on session length and number of apps
	// Each app sends one request per session
	numBlocksPerSession := int64(ct.sharedParams.GetNumBlocksPerSession())
	requestsPerSession := len(ct.applications) // One request per app per session

	ct.logger.Info().
		Int64("num_blocks_per_session", numBlocksPerSession).
		Int("requests_per_session", requestsPerSession).
		Int("duration_sessions", ct.config.DurationSessions).
		Int("total_apps", len(ct.applications)).
		Msg("Starting to send requests distributed across sessions")

	sessionsCompleted := 0

	for sessionsCompleted < ct.config.DurationSessions {
		// Wait for session start
		ct.sessionStarted = false
		for !ct.sessionStarted {
			select {
			case <-ct.ctx.Done():
				return fmt.Errorf("context cancelled while waiting for session")
			default:
				time.Sleep(100 * time.Millisecond)
			}
		}

		ct.logger.Info().
			Int64("session_number", ct.currentSession).
			Int("session_count", sessionsCompleted+1).
			Int("total_sessions", ct.config.DurationSessions).
			Msg("Starting requests for session")

		// Send requests distributed across the session
		if err := ct.sendRequestsForSession(numBlocksPerSession); err != nil {
			return fmt.Errorf("failed to send requests for session: %w", err)
		}

		sessionsCompleted++
	}

	return nil
}

func (ct *ClaimsStressTester) sendRequestsForSession(numBlocksPerSession int64) error {
	// Calculate interval to distribute requests across the session
	// Each app sends one request per session, distributed across session blocks
	sessionDuration := time.Duration(numBlocksPerSession) * 30 * time.Second // Assuming ~30s block time
	requestInterval := (sessionDuration) / time.Duration(len(ct.applications))

	var wg sync.WaitGroup

	for i, app := range ct.applications {
		wg.Add(1)
		go func(appIndex int, application *Application) {
			defer wg.Done()

			// Distribute requests across the session
			delay := time.Duration(appIndex) * requestInterval
			time.Sleep(delay)

			// Send single request for this app in this session
			ct.sendRelayRequest(application, int(ct.requestsSent))
			ct.requestsSent++
		}(i, app)
	}

	// Wait for all requests in this session to complete
	wg.Wait()

	return nil
}

func (ct *ClaimsStressTester) cleanup() {
	if ct.applications == nil {
		return
	}

	ct.logger.Info().Msg("Cleaning up application keys from keyring")

	for i, app := range ct.applications {
		if app == nil {
			continue
		}

		// Delete the application key from the keyring
		if ct.txContext != nil && ct.txContext.GetKeyring() != nil {
			accAddress := sdk.MustAccAddressFromBech32(app.Address)
			err := ct.txContext.GetKeyring().DeleteByAddress(accAddress)
			if err != nil {
				ct.logger.Warn().
					Err(err).
					Str("app_address", app.Address).
					Msgf("Failed to delete application %d key from keyring", i+1)
			} else {
				ct.logger.Debug().
					Str("app_address", app.Address).
					Msgf("Deleted application %d key from keyring", i+1)
			}
		}
	}
}

func (ct *ClaimsStressTester) sendRelayRequest(app *Application, requestID int) {
	// Create JSON-RPC payload
	payload := fmt.Sprintf(claimsRelayPayloadFmt, claimsRelayRequestMethod, requestID+1)

	// Create HTTP request
	req, err := http.NewRequest("POST", ct.config.GatewayURL, strings.NewReader(payload))
	if err != nil {
		ct.logger.Error().
			Err(err).
			Str("app_address", app.Address).
			Int("request_id", requestID).
			Msg("Failed to create HTTP request")
		return
	}

	// Set required headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("App-Address", app.Address)
	req.Header.Set("Target-Service-Id", ct.config.ServiceID)

	// Send request
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		ct.logger.Error().
			Err(err).
			Str("app_address", app.Address).
			Int("request_id", requestID).
			Msg("Failed to send relay request")
		return
	}
	defer resp.Body.Close()

	// Log successful request
	if resp.StatusCode == http.StatusOK {
		ct.logger.Info().
			Str("app_address", app.Address).
			Int("request_id", requestID).
			Int("status_code", resp.StatusCode).
			Msg("Relay request successful")
	} else {
		ct.logger.Warn().
			Str("app_address", app.Address).
			Int("request_id", requestID).
			Int("status_code", resp.StatusCode).
			Msg("Relay request failed")
	}
}

// initializeLoadTestManifest reads and parses the load test manifest file
// using the same logic as the existing relay stress tests.
func initializeLoadTestManifest(t *testing.T) *config.LoadTestManifestYAML {
	workingDirectory, err := os.Getwd()
	require.NoError(t, err)

	manifestPath := filepath.Join(workingDirectory, "..", "..", flagManifestFilePath)
	loadTestManifestContent, err := os.ReadFile(manifestPath)
	require.NoError(t, err)

	loadTestManifest, err := config.ParseLoadTestManifest(loadTestManifestContent)
	require.NoError(t, err)

	return loadTestManifest
}
