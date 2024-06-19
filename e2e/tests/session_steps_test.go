//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"cosmossdk.io/depinject"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/testclient"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	// eventTimeout is the duration of time to wait after sending a valid tx
	// before the test should time out (fail).
	eventTimeout = 60 * time.Second
	// testServiceId is the service ID used for testing purposes that is
	// expected to be available in LocalNet.
	testServiceId = "anvil"

	// txSenderEventSubscriptionQueryFmt is the format string which yields the
	// cosmos-sdk event subscription "query" string for a given sender address.
	// This is used by an events replay client to subscribe to tx events from the supplier.
	// See: https://docs.cosmos.network/v0.47/learn/advanced/events#subscribing-to-events
	txSenderEventSubscriptionQueryFmt = "tm.event='Tx' AND message.sender='%s'"
	// newBlockEventSubscriptionQuery is the query string which yields a
	// subscription query to listen for on-chain new block events.
	newBlockEventSubscriptionQuery = "tm.event='NewBlock'"
	// eventsReplayClientBufferSize is the buffer size for the events replay client
	// for the subscriptions above.
	eventsReplayClientBufferSize = 100

	// txResultEventsReplayClientKey is the suite#scenarioState key for the events replay client
	// which is subscribed to tx events where the tx sender is the scenario's supplier.
	txResultEventsReplayClientKey = "txResultEventsReplayClientKey"
	// newBlockEventReplayClientKey is the suite#scenarioState key for the events replay client
	// which is subscribed to claim settlement or expiration events on-chain.
	newBlockEventReplayClientKey = "newBlockEventReplayClientKey"

	// preExistingClaimsKey is the suite#scenarioState key for any pre-existing
	// claims when querying for all claims prior to running the scenario.
	preExistingClaimsKey = "preExistingClaimsKey"
	// preExistingProofsKey is the suite#scenarioState key for any pre-existing
	// proofs when querying for all proofs prior to running the scenario.
	preExistingProofsKey = "preExistingProofsKey"
)

func (s *suite) TheUserShouldWaitForTheModuleMessageToBeSubmitted(module, message string) {
	msgType := fmt.Sprintf("/poktroll.%s.Msg%s", module, message)
	s.waitForTxResultEvent(msgType)
}

func (s *suite) TheUserShouldWaitForTheModuleEventToBeBroadcast(module, message string) {
	eventType := fmt.Sprintf("poktroll.%s.Event%s", module, message)
	isExpectedEventFn := func(event *abci.Event) bool { return event.Type == eventType }
	s.waitForNewBlockEvent(isExpectedEventFn)
}

// TODO_FLAKY: See how 'TheClaimCreatedBySupplierForServiceForApplicationShouldBeSuccessfullySettled'
// was modified to using an event replay client, instead of a query, to eliminate the flakiness.
func (s *suite) TheClaimCreatedBySupplierForServiceForApplicationShouldBePersistedOnchain(supplierName, serviceId, appName string) {
	ctx := context.Background()

	allClaimsRes, err := s.proofQueryClient.AllClaims(ctx, &prooftypes.QueryAllClaimsRequest{
		Filter: &prooftypes.QueryAllClaimsRequest_SupplierAddress{
			SupplierAddress: accNameToAddrMap[supplierName],
		},
	})
	require.NoError(s, err)
	require.NotNil(s, allClaimsRes)

	// Assert that the number of claims has increased by one.
	preExistingClaims, ok := s.scenarioState[preExistingClaimsKey].([]prooftypes.Claim)
	require.Truef(s, ok, "%s not found in scenarioState", preExistingClaimsKey)
	// NB: We are avoiding the use of require.Len here because it provides unreadable output
	// TODO_TECHDEBT: Due to the speed of the blocks of the LocalNet validator, along with the small number
	// of blocks per session, multiple claims may be created throughout the duration of the test. Until
	// these values are appropriately adjusted
	require.Greater(s, len(allClaimsRes.Claims), len(preExistingClaims), "number of claims must have increased")

	// TODO_IMPROVE: assert that the root hash of the claim contains the correct
	// SMST sum. The sum can be retrieved by parsing the last 8 bytes as a
	// binary-encoded uint64; e.g. something like:
	// `binary.Uvarint(claim.RootHash[len(claim.RootHash-8):])`

	// TODO_IMPROVE: add assertions about serviceId and appName and/or incorporate
	// them into the scenarioState key(s).

	claim := allClaimsRes.Claims[0]
	require.Equal(s, accNameToAddrMap[supplierName], claim.SupplierAddress)
}

func (s *suite) TheSupplierHasServicedASessionWithRelaysForServiceForApplication(supplierName, numRelaysStr, serviceId, appName string) {
	ctx := context.Background()

	numRelays, err := strconv.Atoi(numRelaysStr)
	require.NoError(s, err)

	// Query for any existing claims so that we can compare against them in
	// future assertions about changes in on-chain claims.
	allClaimsRes, err := s.proofQueryClient.AllClaims(ctx, &prooftypes.QueryAllClaimsRequest{})
	require.NoError(s, err)
	s.scenarioState[preExistingClaimsKey] = allClaimsRes.Claims

	// Query for any existing proofs so that we can compare against them in
	// future assertions about changes in on-chain proofs.
	allProofsRes, err := s.proofQueryClient.AllProofs(ctx, &prooftypes.QueryAllProofsRequest{})
	require.NoError(s, err)
	s.scenarioState[preExistingProofsKey] = allProofsRes.Proofs

	// Construct an events query client to listen for tx events from the supplier.
	msgSenderQuery := fmt.Sprintf(txSenderEventSubscriptionQueryFmt, accNameToAddrMap[supplierName])
	deps := depinject.Supply(events.NewEventsQueryClient(testclient.CometLocalWebsocketURL))
	txSendEventsReplayClient, err := events.NewEventsReplayClient[*abci.TxResult](
		ctx,
		deps,
		msgSenderQuery,
		tx.UnmarshalTxResult,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)
	s.scenarioState[txResultEventsReplayClientKey] = txSendEventsReplayClient

	// Construct an events query client to listen for claim settlement or expiration events on-chain.
	onChainClaimEventsReplayClient, err := events.NewEventsReplayClient[*block.CometNewBlockEvent](
		ctx,
		deps,
		newBlockEventSubscriptionQuery,
		block.UnmarshalNewBlockEvent,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)
	s.scenarioState[newBlockEventReplayClientKey] = onChainClaimEventsReplayClient

	// Send relays for the session.
	s.sendRelaysForSession(
		appName,
		supplierName,
		testServiceId,
		numRelays,
	)
}

func (s *suite) TheClaimCreatedBySupplierForServiceForApplicationShouldBeSuccessfullySettled(supplierName, serviceId, appName string) {
	app, ok := accNameToAppMap[appName]
	require.True(s, ok, "application %s not found", appName)

	supplier, ok := accNameToSupplierMap[supplierName]
	require.True(s, ok, "supplier %s not found", supplierName)

	isValidClaimSettledEvent := func(event *abci.Event) bool {
		if event.Type != "poktroll.tokenomics.EventClaimSettled" {
			return false
		}
		claimSettledEvent := s.abciToClaimSettledEvent(event)
		claim := claimSettledEvent.Claim
		require.Equal(s, app.Address, claim.SessionHeader.ApplicationAddress)
		require.Equal(s, supplier.Address, claim.SupplierAddress)
		require.Equal(s, serviceId, claim.SessionHeader.Service.Id)
		require.Greater(s, claimSettledEvent.ComputeUnits, uint64(0), "compute units should be greater than 0")
		s.Logf("Claim settled for %d compute units w/ proof requirement: %t\n", claimSettledEvent.ComputeUnits, claimSettledEvent.ProofRequired)
		return true
	}

	s.waitForNewBlockEvent(isValidClaimSettledEvent)
}

func (s *suite) sendRelaysForSession(
	appName string,
	supplierName string,
	serviceId string,
	relayLimit int,
) {
	s.TheApplicationIsStakedForService(appName, serviceId)
	s.TheSupplierIsStakedForService(supplierName, serviceId)
	s.TheSessionForApplicationAndServiceContainsTheSupplier(appName, serviceId, supplierName)

	// TODO_IMPROVE/TODO_COMMUNITY: hard-code a default set of RPC calls to iterate over for coverage.
	data := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`

	for i := 0; i < relayLimit; i++ {
		s.Logf("Sending relay %d \n", i)
		s.TheApplicationSendsTheSupplierARequestForServiceWithData(appName, supplierName, serviceId, data)
		s.TheApplicationReceivesASuccessfulRelayResponseSignedBy(appName, supplierName)
	}
}

// waitForTxResultEvent waits for an event to be observed which has the given message action.
func (s *suite) waitForTxResultEvent(targetAction string) {
	ctx, done := context.WithCancel(context.Background())

	txResultEventsReplayClientState, ok := s.scenarioState[txResultEventsReplayClientKey]
	require.Truef(s, ok, "%s not found in scenarioState", txResultEventsReplayClientKey)

	txResultEventsReplayClient, ok := txResultEventsReplayClientState.(client.EventsReplayClient[*abci.TxResult])
	require.True(s, ok, "%q not of the right type; expected client.EventsReplayClient[*abci.TxResult], got %T", txResultEventsReplayClientKey, txResultEventsReplayClientState)
	require.NotNil(s, txResultEventsReplayClient)

	// For each observed event, **asynchronously** check if it contains the given action.
	channel.ForEach[*abci.TxResult](
		ctx, txResultEventsReplayClient.EventsSequence(ctx),
		func(_ context.Context, txResult *abci.TxResult) {
			if txResult == nil {
				return
			}

			// Range over each event's attributes to find the "action" attribute
			// and compare its value to that of the action provided.
			for _, event := range txResult.Result.Events {
				for _, attribute := range event.Attributes {
					if attribute.Key == "action" {
						if attribute.Value == targetAction {
							done()
							return
						}
					}
				}
			}
		},
	)

	select {
	case <-time.After(eventTimeout):
		s.Fatalf("ERROR: timed out waiting for message with action %q", targetAction)
	case <-ctx.Done():
		s.Log("Success; message detected before timeout.")
	}
}

// waitForNewBlockEvent waits for an event to be observed whose type and data
// match the conditions specified by isEventMatchFn.
// isEventMatchFn is a function that receives an abci.Event and returns a boolean
// indicating whether the event matches the desired conditions.
func (s *suite) waitForNewBlockEvent(
	isEventMatchFn func(*abci.Event) bool,
) {
	ctx, done := context.WithCancel(context.Background())

	newBlockEventsReplayClientState, ok := s.scenarioState[newBlockEventReplayClientKey]
	require.Truef(s, ok, "%s not found in scenarioState", newBlockEventReplayClientKey)

	newBlockEventsReplayClient, ok := newBlockEventsReplayClientState.(client.EventsReplayClient[*block.CometNewBlockEvent])
	require.True(s, ok, "%q not of the right type; expected client.EventsReplayClient[*block.CometNewBlockEvent], got %T", newBlockEventReplayClientKey, newBlockEventsReplayClientState)
	require.NotNil(s, newBlockEventsReplayClient)

	// For each observed event, **asynchronously** check if it contains the given action.
	channel.ForEach[*block.CometNewBlockEvent](
		ctx, newBlockEventsReplayClient.EventsSequence(ctx),
		func(_ context.Context, newBlockEvent *block.CometNewBlockEvent) {
			if newBlockEvent == nil {
				return
			}

			// Range over each event's attributes to find the "action" attribute
			// and compare its value to that of the action provided.
			for _, event := range newBlockEvent.Data.Value.ResultFinalizeBlock.Events {
				// Checks on the event. For example, for a Claim Settlement event,
				// we can parse the claim and verify the compute units.
				if isEventMatchFn(&event) {
					done()
					return
				}
			}
		},
	)

	select {
	case <-time.After(eventTimeout):
		s.Fatalf("ERROR: timed out waiting for NewBlock event")
	case <-ctx.Done():
		s.Log("Success; message detected before timeout.")
	}
}

// abciToClaimSettledEvent converts an abci.Event to a tokenomics.EventClaimSettled
// NB: This was a ChatGPT generated function.
func (s *suite) abciToClaimSettledEvent(event *abci.Event) *tokenomicstypes.EventClaimSettled {
	var claimSettledEvent tokenomicstypes.EventClaimSettled

	// TODO_TECHDEBT: Investigate why `cosmostypes.ParseTypedEvent(*event)` throws
	// an error where cosmostypes is imported from "github.com/cosmos/cosmos-sdk/types"
	// resulting in the following error:
	// 'json: error calling MarshalJSON for type json.RawMessage: invalid character 'E' looking for beginning of value'
	// typedEvent, err := cosmostypes.ParseTypedEvent(*event)

	for _, attr := range event.Attributes {
		switch string(attr.Key) {
		case "claim":
			var claim prooftypes.Claim
			if err := s.cdc.UnmarshalJSON([]byte(attr.Value), &claim); err != nil {
				s.Fatalf("ERROR: failed to unmarshal claim: %v", err)
			}
			claimSettledEvent.Claim = &claim
		case "compute_units":
			value := string(attr.Value)
			value = value[1 : len(value)-1] // Remove surrounding quotes
			computeUnits, err := strconv.ParseUint(value, 10, 64)
			if err != nil {
				s.Fatalf("ERROR: failed to parse compute_units: %v", err)
			}
			claimSettledEvent.ComputeUnits = computeUnits
		case "proof_required":
			proofRequired, err := strconv.ParseBool(string(attr.Value))
			if err != nil {
				s.Fatalf("ERROR: failed to parse proof_required: %v", err)
			}
			claimSettledEvent.ProofRequired = proofRequired
		}
	}
	return &claimSettledEvent
}
