//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	testutilevents "github.com/pokt-network/poktroll/testutil/events"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	// eventTimeout is the duration of time to wait after sending a valid tx
	// before the test should time out (fail).
	eventTimeout = 100 * time.Second
	// testServiceId is the service ID used for testing purposes that is
	// expected to be available in LocalNet.
	testServiceId = "0021"
	// defaultJSONPRCPath is the default path used for sending JSON-RPC relay requests.
	defaultJSONPRCPath = ""

	// eventsReplayClientBufferSize is the buffer size for the events replay client
	// for the subscriptions above.
	eventsReplayClientBufferSize = 100

	// preExistingClaimsKey is the suite#scenarioState key for any pre-existing
	// claims when querying for all claims prior to running the scenario.
	preExistingClaimsKey = "preExistingClaimsKey"
	// preExistingProofsKey is the suite#scenarioState key for any pre-existing
	// proofs when querying for all proofs prior to running the scenario.
	preExistingProofsKey = "preExistingProofsKey"
)

func (s *suite) TheUserShouldWaitForTheModuleMessageToBeSubmitted(module, msgType string) {
	event := s.waitForTxResultEvent(newEventMsgTypeMatchFn(module, msgType))

	// If the message type is "SubmitProof", save the supplier balance
	// so that next steps that assert on supplier rewards can do it without having
	// the proof submission fee skewing the results.
	switch msgType {
	case "SubmitProof":
		supplierOperatorAddress := getMsgSubmitProofSenderAddress(event)
		require.NotEmpty(s, supplierOperatorAddress)

		supplierAccName := accAddrToNameMap[supplierOperatorAddress]

		// Get current balance
		balanceKey := accBalanceKey(supplierAccName)
		currBalance := s.getAccBalance(supplierAccName)
		s.scenarioState[balanceKey] = currBalance // save the balance for later
	default:
		s.Log("no test suite state to update for message type %s", msgType)
	}

	// Rebuild actor maps after the relevant messages have been committed.
	switch module {
	case apptypes.ModuleName:
		s.buildAppMap()
	case suppliertypes.ModuleName:
		s.buildSupplierMap()
	default:
		s.Log("no test suite state to update for module %s", module)
	}
}

func (s *suite) TheUserShouldWaitForTheModuleTxEventToBeBroadcast(module, eventType string) {
	s.waitForTxResultEvent(newEventTypeMatchFn(module, eventType))

	// Rebuild actor maps after the relevant messages have been committed.
	switch module {
	case apptypes.ModuleName:
		s.buildAppMap()
	case suppliertypes.ModuleName:
		s.buildSupplierMap()
	default:
		s.Log("no test suite state to update for module %s", module)
	}
}

func (s *suite) TheUserShouldWaitForTheClaimsettledEventWithProofRequirementToBeBroadcast(proofRequirement string) {
	s.waitForNewBlockEvent(
		combineEventMatchFns(
			newEventTypeMatchFn("tokenomics", "ClaimSettled"),
			newEventModeMatchFn("EndBlock"),
			newEventAttributeMatchFn("proof_requirement", fmt.Sprintf("%q", proofRequirement)),
		),
	)

	// Update the actor maps after end block events have been emitted.
	s.buildAppMap()
	s.buildSupplierMap()
}

// TODO_FLAKY: See how 'TheClaimCreatedBySupplierForServiceForApplicationShouldBeSuccessfullySettled'
// was modified to using an event replay client, instead of a query, to eliminate the flakiness.
func (s *suite) TheClaimCreatedBySupplierForServiceForApplicationShouldBePersistedOnchain(supplierOperatorName, serviceId, appName string) {
	ctx := context.Background()

	allClaimsRes, err := s.proofQueryClient.AllClaims(ctx, &prooftypes.QueryAllClaimsRequest{
		Filter: &prooftypes.QueryAllClaimsRequest_SupplierOperatorAddress{
			SupplierOperatorAddress: accNameToAddrMap[supplierOperatorName],
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
	require.Equal(s, accNameToAddrMap[supplierOperatorName], claim.SupplierOperatorAddress)
}

func (s *suite) TheSupplierHasServicedASessionWithRelaysForServiceForApplication(supplierOperatorName, numRelaysStr, serviceId, appName string) {
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

	// Send relays for the session.
	s.sendRelaysForSession(
		appName,
		supplierOperatorName,
		testServiceId,
		numRelays,
	)
}

func (s *suite) TheClaimCreatedBySupplierForServiceForApplicationShouldBeSuccessfullySettled(supplierOperatorName, serviceId, appName string) {
	app, ok := accNameToAppMap[appName]
	require.True(s, ok, "application %s not found", appName)

	supplier, ok := operatorAccNameToSupplierMap[supplierOperatorName]
	require.True(s, ok, "supplier %s not found", supplierOperatorName)

	isValidClaimSettledEvent := func(event *abci.Event) bool {
		if event.Type != "poktroll.tokenomics.EventClaimSettled" {
			return false
		}

		// Parse the event
		testutilevents.QuoteEventMode(event)
		typedEvent, err := cosmostypes.ParseTypedEvent(*event)
		require.NoError(s, err)
		require.NotNil(s, typedEvent)
		claimSettledEvent, ok := typedEvent.(*tokenomicstypes.EventClaimSettled)
		require.True(s, ok)

		// Assert that the claim was settled for the correct application, supplier, and service.
		claim := claimSettledEvent.Claim
		require.Equal(s, app.Address, claim.SessionHeader.ApplicationAddress)
		require.Equal(s, supplier.OperatorAddress, claim.SupplierOperatorAddress)
		require.Equal(s, serviceId, claim.SessionHeader.ServiceId)
		require.Greater(s, claimSettledEvent.NumClaimedComputeUnits, uint64(0), "claimed compute units should be greater than 0")
		// TODO_FOLLOWUP: Add NumEstimatedComputeUnits and ClaimedAmountUpokt
		return true
	}

	s.waitForNewBlockEvent(isValidClaimSettledEvent)
}

func (suite *suite) TheModuleParametersAreSetAsFollows(moduleName string, params gocuke.DataTable) {
	suite.AnAuthzGrantFromTheAccountToTheAccountForTheMessageExists(
		"gov",
		"module",
		"pnf",
		"user",
		fmt.Sprintf("/poktroll.%s.MsgUpdateParams", moduleName),
	)

	suite.TheAccountSendsAnAuthzExecMessageToUpdateAllModuleParams("pnf", moduleName, params)
}

func (s *suite) sendRelaysForSession(
	appName string,
	supplierOperatorName string,
	serviceId string,
	relayLimit int,
) {
	s.TheApplicationIsStakedForService(appName, serviceId)
	s.TheSupplierIsStakedForService(supplierOperatorName, serviceId)
	s.TheSessionForApplicationAndServiceContainsTheSupplier(appName, serviceId, supplierOperatorName)

	// TODO_IMPROVE: hard-code a default set of RPC calls to iterate over for coverage.
	payload_fmt := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":%d}`

	for i := 0; i < relayLimit; i++ {
		payload := fmt.Sprintf(payload_fmt, i+1) // i+1 to avoid id=0 which is invalid

		s.TheApplicationSendsTheSupplierASuccessfulRequestForServiceWithPathAndData(appName, supplierOperatorName, serviceId, defaultJSONPRCPath, payload)
		time.Sleep(10 * time.Millisecond)
	}
}

// waitForTxResultEvent waits for an event to be observed which has the given message action.
func (s *suite) waitForTxResultEvent(eventIsMatch func(*abci.Event) bool) (matchedEvent *abci.Event) {
	ctx, cancel := context.WithCancel(s.ctx)

	// For each observed event, **asynchronously** check if it contains the given action.
	s.forEachTxResult(ctx,
		func(_ context.Context, txResult *abci.TxResult) {
			if txResult == nil {
				return
			}

			// Range over each event's attributes to find the "action" attribute
			// and compare its value to that of the action provided.
			for _, event := range txResult.Result.Events {
				if eventIsMatch(&event) {
					matchedEvent = &event
					cancel()
				}
			}
		},
	)

	return matchedEvent
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
	s.forEachBlockEvent(ctx,
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
}

// waitForBlockHeight waits for a NewBlock event to be observed whose height is
// greater than or equal to the target height.
func (s *suite) waitForBlockHeight(targetHeight int64) {
	ctx, done := context.WithCancel(s.ctx)

	// For each observed event, **asynchronously** check if it is greater than
	// or equal to the target height
	s.forEachBlockEvent(ctx,
		func(_ context.Context, newBlockEvent *block.CometNewBlockEvent) {
			if newBlockEvent == nil {
				return
			}

			if newBlockEvent.Data.Value.Block.Header.Height >= targetHeight {
				done()
				return
			}
		},
	)
}

// forEachBlockEvent calls blockEventFn for each observed block event **asynchronously**
// and blocks on waiting for the given context to be cancelled. If the context is
// not cancelled before eventTimeout, the test will fail.
func (s *suite) forEachBlockEvent(
	ctx context.Context,
	blockEventFn func(_ context.Context, newBlockEvent *block.CometNewBlockEvent),
) {
	channel.ForEach[*block.CometNewBlockEvent](ctx,
		s.newBlockEventsReplayClient.EventsSequence(ctx),
		blockEventFn,
	)

	select {
	case <-time.After(eventTimeout):
		s.Fatalf("ERROR: timed out waiting new block event")
	case <-ctx.Done():
		s.Log("Success; new block event detected before timeout.")
	}
}

// forEachTxResult calls txResult for each observed tx result **asynchronously**
// and blocks on waiting for the given context to be cancelled. If the context is
// not cancelled before eventTimeout, the test will fail.
func (s *suite) forEachTxResult(
	ctx context.Context,
	txResultFn func(_ context.Context, txResult *abci.TxResult),
) {

	channel.ForEach[*abci.TxResult](ctx,
		s.txResultReplayClient.EventsSequence(ctx),
		txResultFn,
	)

	select {
	case <-time.After(eventTimeout):
		s.Fatalf("ERROR: timed out waiting for tx result")
	case <-ctx.Done():
		s.Log("Success; tx result detected before timeout.")
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

// getMsgSubmitProofSenderAddress returns the sender address from the given event.
func getMsgSubmitProofSenderAddress(event *abci.Event) string {
	senderAttrIdx := slices.IndexFunc(event.Attributes, func(attr abci.EventAttribute) bool {
		return attr.Key == "sender"
	})

	return event.Attributes[senderAttrIdx].Value
}
