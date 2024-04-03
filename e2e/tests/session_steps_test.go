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
	// tokenomictypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	// txEventTimeout is the duration of time to wait after sending a valid tx
	// before the test should time out (fail).
	txEventTimeout = 60 * time.Second
	// testServiceId is the service ID used for testing purposes that is
	// expected to be available in LocalNet.
	testServiceId = "anvil"

	// txSenderEventSubscriptionQueryFmt is the format string which yields the
	// cosmos-sdk event subscription "query" string for a given sender address.
	// This is used by an events replay client to subscribe to tx events from the supplier.
	// See: https://docs.cosmos.network/v0.47/learn/advanced/events#subscribing-to-events
	txSenderEventSubscriptionQueryFmt = "tm.event='Tx' AND message.sender='%s'"
	// onChainClaimEventSubscriptionQueryFmt is the format string which yields a
	// subscription query to listen for on-chain new block events.
	onChainClaimEventSubscriptionQueryFmt = "tm.event='NewBlock'"
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

func (s *suite) TheUserShouldWaitForTheMessageToBeSubmitted(module, message string) {
	s.waitForTxResultEvent(fmt.Sprintf("/poktroll.%s.Msg%s", module, message))
}

func (s *suite) TheUserShouldWaitForTheNewBlockEventToBeBroadcasted(module, message string) {
	s.waitForNewBlockEvent(fmt.Sprintf("poktroll.%s.Event%s", module, message))
}

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
	require.True(s, ok, fmt.Sprintf("%s not found in scenarioState", preExistingClaimsKey))
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
	onChainClaimEventQuery := onChainClaimEventSubscriptionQueryFmt
	onChainClaimEventsReplayClient, err := events.NewEventsReplayClient[*block.CometNewBlockEvent](
		ctx,
		deps,
		onChainClaimEventQuery,
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

func (s *suite) TheProofSubmittedBySupplierForServiceForApplicationShouldBePersistedOnchain(supplierName, serviceId, appName string) {
	ctx := context.Background()

	// Retrieve all on-chain proofs for supplierName
	allProofsRes, err := s.proofQueryClient.AllProofs(ctx, &prooftypes.QueryAllProofsRequest{
		Filter: &prooftypes.QueryAllProofsRequest_SupplierAddress{
			SupplierAddress: accNameToAddrMap[supplierName],
		},
	})
	require.NoError(s, err)
	require.NotNil(s, allProofsRes)

	// Assert that the number of proofs has increased by one.
	preExistingProofs, ok := s.scenarioState[preExistingProofsKey].([]prooftypes.Proof)
	require.True(s, ok, "preExistingProofsKey not found in scenarioState")
	// NB: We are avoiding the use of require.Len here because it provides unreadable output
	// TODO_TECHDEBT: Due to the speed of the blocks of the LocalNet validator, along with the small number
	// of blocks per session, multiple proofs may be created throughout the duration of the test. Until
	// these values are appropriately adjusted, we assert on an increase in proofs rather than +1.
	require.Greater(s, len(allProofsRes.Proofs), len(preExistingProofs), "number of proofs must have increased")

	// TODO_UPNEXT(@bryanchriswhite): assert that the root hash of the proof contains the correct
	// SMST sum. The sum can be retrieved via the `GetSum` function exposed
	// by the SMT.

	// TODO_IMPROVE: add assertions about serviceId and appName and/or incorporate
	// them into the scenarioState key(s).

	proof := allProofsRes.Proofs[0]
	require.Equal(s, accNameToAddrMap[supplierName], proof.SupplierAddress)
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
		fmt.Printf("Sending relay %d \n", i)
		s.TheApplicationSendsTheSupplierARequestForServiceWithData(appName, supplierName, serviceId, data)
		s.TheApplicationReceivesASuccessfulRelayResponseSignedBy(appName, supplierName)
	}
}

// waitForEvent waits for an event to be observed which has the given message action.
func (s *suite) waitForTxResultEvent(targetAction string) {
	ctx, done := context.WithCancel(context.Background())

	eventsReplayClientState, found := s.scenarioState[txResultEventsReplayClientKey]
	require.True(s, found, fmt.Sprintf("%s not found in scenarioState", txResultEventsReplayClientKey))

	eventsReplayClient, ok := eventsReplayClientState.(client.EventsReplayClient[*abci.TxResult])
	require.True(s, ok, fmt.Sprintf("%s not of the right type", txResultEventsReplayClientKey))
	require.NotNil(s, eventsReplayClient)

	// For each observed event, **asynchronously** check if it contains the given action.
	channel.ForEach[*abci.TxResult](
		ctx, eventsReplayClient.EventsSequence(ctx),
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
	case <-time.After(txEventTimeout):
		s.Fatalf("timed out waiting for message with action %q", targetAction)
	case <-ctx.Done():
		s.Log("Success; message detected before timeout.")
	}
}

func (s *suite) waitForNewBlockEvent(targetEvent string) {
	ctx, done := context.WithCancel(context.Background())

	newBlockEventsReplayClientState, found := s.scenarioState[newBlockEventReplayClientKey]
	require.True(s, found, fmt.Sprintf("%s not found in scenarioState", newBlockEventReplayClientKey))

	newBlockEventsReplayClient, ok := newBlockEventsReplayClientState.(client.EventsReplayClient[*block.CometNewBlockEvent])
	require.True(s, ok, fmt.Sprintf("%s not of the right type", newBlockEventReplayClientKey))
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
				// TODO_IMPROVE: We can pass in a function to do even more granular
				// checks on the event. For example, for a Claim Settlement event,
				// we can parse the claim and verify the compute units.
				if event.Type == targetEvent {
					done()
					return
				}
			}
		},
	)

	select {
	case <-time.After(txEventTimeout):
		s.Fatalf("timed out waiting for NewBlock event %q", targetEvent)
	case <-ctx.Done():
		s.Log("Success; message detected before timeout.")
	}
}
