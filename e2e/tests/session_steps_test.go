//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strconv"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/proto/types/tokenomics"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
)

const (
	// eventTimeout is the duration of time to wait after sending a valid tx
	// before the test should time out (fail).
	eventTimeout = 100 * time.Second
	// testServiceId is the service ID used for testing purposes that is
	// expected to be available in LocalNet.
	testServiceId = "anvil"
	// defaultJSONPRCPath is the default path used for sending JSON-RPC relay requests.
	defaultJSONPRCPath = ""

	// txSenderEventSubscriptionQueryFmt is the format string which yields the
	// cosmos-sdk event subscription "query" string for a given sender address.
	// This is used by an events replay client to subscribe to tx events from the supplier.
	// See: https://docs.cosmos.network/v0.47/learn/advanced/events#subscribing-to-events
	txSenderEventSubscriptionQueryFmt = "tm.event='Tx' AND message.sender='%s'"

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

func (s *suite) TheUserShouldWaitForTheModuleMessageToBeSubmitted(module, msgType string) {
	s.waitForTxResultEvent(newEventMsgTypeMatchFn(module, msgType))
}

func (s *suite) TheUserShouldWaitForTheModuleTxEventToBeBroadcast(module, eventType string) {
	s.waitForTxResultEvent(newEventTypeMatchFn(module, eventType))
}

func (s *suite) TheUserShouldWaitForTheModuleEndBlockEventToBeBroadcast(module, eventType string) {
	s.waitForNewBlockEvent(
		combineEventMatchFns(
			newEventTypeMatchFn(module, eventType),
			newEventModeMatchFn("EndBlock"),
		),
	)
}

// TODO_FLAKY: See how 'TheClaimCreatedBySupplierForServiceForApplicationShouldBeSuccessfullySettled'
// was modified to using an event replay client, instead of a query, to eliminate the flakiness.
func (s *suite) TheClaimCreatedBySupplierForServiceForApplicationShouldBePersistedOnchain(supplierName, serviceId, appName string) {
	ctx := context.Background()

	allClaimsRes, err := s.proofQueryClient.AllClaims(ctx, &proof.QueryAllClaimsRequest{
		Filter: &proof.QueryAllClaimsRequest_SupplierAddress{
			SupplierAddress: accNameToAddrMap[supplierName],
		},
	})
	require.NoError(s, err)
	require.NotNil(s, allClaimsRes)

	// Assert that the number of claims has increased by one.
	preExistingClaims, ok := s.scenarioState[preExistingClaimsKey].([]proof.Claim)
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
	allClaimsRes, err := s.proofQueryClient.AllClaims(ctx, &proof.QueryAllClaimsRequest{})
	require.NoError(s, err)
	s.scenarioState[preExistingClaimsKey] = allClaimsRes.Claims

	// Query for any existing proofs so that we can compare against them in
	// future assertions about changes in on-chain proofs.
	allProofsRes, err := s.proofQueryClient.AllProofs(ctx, &proof.QueryAllProofsRequest{})
	require.NoError(s, err)
	s.scenarioState[preExistingProofsKey] = allProofsRes.Proofs

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

		// Parse the event
		testutilevents.QuoteEventMode(event)
		typedEvent, err := cosmostypes.ParseTypedEvent(*event)
		require.NoError(s, err)
		require.NotNil(s, typedEvent)
		claimSettledEvent, ok := typedEvent.(*tokenomics.EventClaimSettled)
		require.True(s, ok)

		// Assert that the claim was settled for the correct application, supplier, and service.
		claim := claimSettledEvent.Claim
		require.Equal(s, app.Address, claim.SessionHeader.ApplicationAddress)
		require.Equal(s, supplier.Address, claim.SupplierAddress)
		require.Equal(s, serviceId, claim.SessionHeader.Service.Id)
		require.Greater(s, claimSettledEvent.NumComputeUnits, uint64(0), "compute units should be greater than 0")
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
		s.TheApplicationSendsTheSupplierARequestForServiceWithPathAndData(appName, supplierName, serviceId, defaultJSONPRCPath, data)
		s.TheApplicationReceivesASuccessfulRelayResponseSignedBy(appName, supplierName)
	}
}

// waitForTxResultEvent waits for an event to be observed which has the given message action.
func (s *suite) waitForTxResultEvent(eventIsMatch func(*abci.Event) bool) {
	ctx, cancel := context.WithCancel(s.ctx)

	// For each observed event, **asynchronously** check if it contains the given action.
	channel.ForEach[*abci.TxResult](
		ctx, s.txResultReplayClient.EventsSequence(ctx),
		func(_ context.Context, txResult *abci.TxResult) {
			if txResult == nil {
				return
			}

			// Range over each event's attributes to find the "action" attribute
			// and compare its value to that of the action provided.
			for _, event := range txResult.Result.Events {
				if eventIsMatch(&event) {
					cancel()
				}
			}
		},
	)

	select {
	case <-time.After(eventTimeout):
		s.Fatalf("ERROR: timed out waiting for tx result event")
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
	ctx, done := context.WithCancel(s.ctx)

	// For each observed event, **asynchronously** check if it contains the given action.
	channel.ForEach[*block.CometNewBlockEvent](
		ctx, s.newBlockEventsReplayClient.EventsSequence(ctx),
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

// newEventTypeMatchFn returns a function that matches an event based on its type
// field. The type URL is constructed from the given module and eventType arguments
// where module is the module name and eventType is the protobuf message type name
// without the "Event" prefix; e.g., pass "tokenomics" and "ClaimSettled" to match
// the "poktroll.tokenomics.EventClaimSettled" event.
func newEventTypeMatchFn(module, eventType string) func(*abci.Event) bool {
	targetEventType := fmt.Sprintf("poktroll.%s.Event%s", module, eventType)
	return func(event *abci.Event) bool {
		if event == nil {
			return false
		}

		if event.Type == targetEventType {
			return true
		}
		return false
	}
}

// newEventMsgTypeMatchFn returns a function that matches an event based on the
// "action" attribute in its attributes field, which is populated with the message
// type URL of the message to which a given event corresponds. The target action
// is constructed from the given module and msgType arguments where module is the
// module name and msgType is the protobuf message type name without the "Msg" prefix;
// e.g., pass "proof" and "CreateClaim" to match the "poktroll.proof.MsgCreateClaim" message.
func newEventMsgTypeMatchFn(module, msgType string) func(event *abci.Event) bool {
	targetMsgType := fmt.Sprintf("/poktroll.%s.Msg%s", module, msgType)
	return newEventAttributeMatchFn("action", targetMsgType)
}

// newEventModeMatchFn returns a function that matches an event based on the
// "mode" attribute in its attributes field. The target mode value is the given
// mode string.
func newEventModeMatchFn(mode string) func(event *abci.Event) bool {
	return newEventAttributeMatchFn("mode", mode)
}

// newEventAttributeMatchFn returns a function that matches an event based on the
// presence of an attribute with the given key and value.
func newEventAttributeMatchFn(key, value string) func(event *abci.Event) bool {
	return func(event *abci.Event) bool {
		if event == nil {
			return false
		}

		for _, attribute := range event.Attributes {
			if attribute.Key == key && attribute.Value == value {
				return true
			}
		}
		return false
	}
}

// combineEventMatchFns returns a function that matches an event based on the
// conjunction of multiple event match functions. The returned function will
// return true only if all the given functions return true.
func combineEventMatchFns(fns ...func(*abci.Event) bool) func(*abci.Event) bool {
	return func(event *abci.Event) bool {
		for _, fn := range fns {
			if !fn(event) {
				return false
			}
		}
		return true
	}
}
