//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"cosmossdk.io/depinject"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/testclient"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	createClaimTimeoutDuration       = 10 * time.Second
	submitProofTimeoutDuration       = 10 * time.Second
	testEventsReplayClientBufferSize = 100
	msgTxSenderQueryFmt              = "tm.event='Tx' AND message.sender='%s'"
	testServiceId                    = "anvil"
	eventsReplayClientKey            = "eventsReplayClientKey"
	preExistingClaimsKey             = "preExistingClaimsKey"
	preExistingProofsKey             = "preExistingProofsKey"
)

func (s *suite) AfterTheSupplierCreatesAClaimForTheSessionForServiceForApplication(serviceId, appName string) {
	s.waitForMessageAction("/pocket.supplier.MsgCreateClaim")
}

func (s *suite) TheClaimCreatedBySupplierForServiceForApplicationShouldBePersistedOnchain(supplierName, serviceId, appName string) {
	ctx := context.Background()

	allClaimsRes, err := s.supplierQueryClient.AllClaims(ctx, &suppliertypes.QueryAllClaimsRequest{
		Filter: &suppliertypes.QueryAllClaimsRequest_SupplierAddress{
			SupplierAddress: accNameToAddrMap[supplierName],
		},
	})
	require.NoError(s, err)
	require.NotNil(s, allClaimsRes)

	// Assert that the number of claims has increased by one.
	preExistingClaims := s.scenarioState[preExistingClaimsKey].([]suppliertypes.Claim)
	// NB: We are avoiding the use of require.Len here because it provides unreadable output
	// TODO_TECHDEBT: Due to the speed of the blocks of the LocalNet sequencer, along with the small number
	// of blocks per session, multiple claims may be created throughout the duration of the test. Until
	// these values are appropriately adjusted
	require.Greater(s, len(allClaimsRes.Claim), len(preExistingClaims), "number of claims must have increased")

	// TODO_IMPROVE: assert that the root hash of the claim contains the correct
	// SMST sum. The sum can be retrieved by parsing the last 8 bytes as a
	// binary-encoded uint64; e.g. something like:
	// `binary.Uvarint(claim.RootHash[len(claim.RootHash-8):])`

	// TODO_IMPROVE: add assertions about serviceId and appName and/or incorporate
	// them into the scenarioState key(s).

	claim := allClaimsRes.Claim[0]
	require.Equal(s, accNameToAddrMap[supplierName], claim.SupplierAddress)
}

func (s *suite) TheSupplierHasServicedASessionWithRelaysForServiceForApplication(supplierName, relayCountStr, serviceId, appName string) {
	ctx := context.Background()

	relayCount, err := strconv.Atoi(relayCountStr)
	require.NoError(s, err)

	// Query for any existing claims so that we can compensate for them in the
	// future assertions about changes in on-chain claims.
	allClaimsRes, err := s.supplierQueryClient.AllClaims(ctx, &suppliertypes.QueryAllClaimsRequest{})
	require.NoError(s, err)
	s.scenarioState[preExistingClaimsKey] = allClaimsRes.Claim

	// Query for any existing proofs so that we can compensate for them in the
	// future assertions about changes in on-chain proofs.
	allProofsRes, err := s.supplierQueryClient.AllProofs(ctx, &suppliertypes.QueryAllProofsRequest{})
	require.NoError(s, err)
	s.scenarioState[preExistingProofsKey] = allProofsRes.Proof

	// Construct an events query client to listen for tx events from the supplier.
	msgSenderQuery := fmt.Sprintf(msgTxSenderQueryFmt, accNameToAddrMap[supplierName])

	deps := depinject.Supply(events.NewEventsQueryClient(testclient.CometLocalWebsocketURL))
	eventsReplayClient, err := events.NewEventsReplayClient[*abci.TxResult](
		ctx,
		deps,
		msgSenderQuery,
		func(eventBz []byte) (*abci.TxResult, error) {
			if strings.Contains(string(eventBz), "jsonrpc") {
				return nil, nil
			}

			// Unmarshal event data into an ABCI TxResult object.
			txResult := &abci.TxResult{}
			err = json.Unmarshal(eventBz, txResult)
			require.NoError(s, err)

			return txResult, nil
		},
		testEventsReplayClientBufferSize,
	)
	require.NoError(s, err)

	s.scenarioState[eventsReplayClientKey] = eventsReplayClient

	s.sendRelaysForSession(
		appName,
		supplierName,
		testServiceId,
		relayCount,
	)
}

func (s *suite) AfterTheSupplierSubmitsAProofForTheSessionForServiceForApplication(a string, b string) {
	s.waitForMessageAction("/pocket.supplier.MsgSubmitProof")
}

func (s *suite) TheProofSubmittedBySupplierForServiceForApplicationShouldBePersistedOnchain(supplierName, serviceId, appName string) {
	ctx := context.Background()

	allProofsRes, err := s.supplierQueryClient.AllProofs(ctx, &suppliertypes.QueryAllProofsRequest{
		Filter: &suppliertypes.QueryAllProofsRequest_SupplierAddress{
			SupplierAddress: accNameToAddrMap[supplierName],
		},
	})
	require.NoError(s, err)
	require.NotNil(s, allProofsRes)

	// Assert that the number of proofs has increased by one.
	preExistingProofs := s.scenarioState[preExistingProofsKey].([]suppliertypes.Proof)
	// NB: We are avoiding the use of require.Len here because it provides unreadable output
	// TODO_TECHDEBT: Due to the speed of the blocks of the LocalNet sequencer, along with the small number
	// of blocks per session, multiple proofs may be created throughout the duration of the test. Until
	// these values are appropriately adjusted
	require.Greater(s, len(allProofsRes.Proof), len(preExistingProofs), "number of proofs must have increased")

	// TODO_IMPROVE: assert that the root hash of the proof contains the correct
	// SMST sum. The sum can be retrieved by parsing the last 8 bytes as a
	// binary-encoded uint64; e.g. something like:
	// `binary.Uvarint(proof.RootHash[len(proof.RootHash-8):])`

	// TODO_IMPROVE: add assertions about serviceId and appName and/or incorporate
	// them into the scenarioState key(s).

	proof := allProofsRes.Proof[0]
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
		s.TheApplicationSendsTheSupplierARequestForServiceWithData(appName, supplierName, serviceId, data)
		s.TheApplicationReceivesASuccessfulRelayResponseSignedBy(appName, supplierName)
	}
}

func (s *suite) waitForMessageAction(action string) {
	ctx, done := context.WithCancel(context.Background())

	eventsReplayClient, ok := s.scenarioState[eventsReplayClientKey].(client.EventsReplayClient[*abci.TxResult])
	require.True(s, ok, "eventsReplayClientKey not found in scenarioState")
	require.NotNil(s, eventsReplayClient)

	eventsSequenceObs := eventsReplayClient.EventsSequence(ctx)

	channel.ForEach[*abci.TxResult](
		ctx, eventsSequenceObs,
		func(_ context.Context, txEvent *abci.TxResult) {
			if txEvent == nil {
				return
			}

			var found bool
			for _, event := range txEvent.Result.Events {
				for _, attribute := range event.Attributes {
					if attribute.Key == "action" {
						if attribute.Value == action {
							found = true
							break
						}
					}
				}
				if found {
					done()
					break
				}
			}
		},
	)

	select {
	case <-ctx.Done():
		// Success; message detected before timeout.
	case <-time.After(submitProofTimeoutDuration):
		s.Fatalf("timed out waiting for message with action %q", action)
	}
}
