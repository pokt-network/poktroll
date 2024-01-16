//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/testclient"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	createClaimTimeoutDuration   = 10 * time.Second
	eitherEventsReplayBufferSize = 100
	// Not liniting as this is a long query string.
	//nolint:lll
	msgClaimSenderQueryFmt     = "tm.event='Tx' AND message.sender='%s' AND message.action='/pocket.supplier.MsgCreateClaim'"
	testServiceId              = "anvil"
	eitherEventsBzReplayObsKey = "eitherEventsBzReplayObsKey"
	preExistingClaimsKey       = "preExistingClaimsKey"
)

func (s *suite) AfterTheSupplierCreatesAClaimForTheSessionForServiceForApplication(serviceId, appName string) {
	ctx, done := context.WithCancel(context.Background())

	// TODO_CONSIDERATION: if this test suite gets more complex, it might make
	// sense to refactor this key into a function that takes serviceId and appName
	// as arguments and returns the key.
	eitherEventsBzReplayObs := s.scenarioState[eitherEventsBzReplayObsKey].(observable.ReplayObservable[either.Bytes])

	// TODO(#220): refactor to use EventsReplayClient once available.
	channel.ForEach[either.Bytes](
		ctx, eitherEventsBzReplayObs,
		func(_ context.Context, eitherEventBz either.Bytes) {
			eventBz, err := eitherEventBz.ValueOrError()
			require.NoError(s, err)

			if strings.Contains(string(eventBz), "jsonrpc") {
				return
			}

			// Unmarshal event data into a TxEventResponse object.
			txEvent := &abci.TxResult{}
			err = json.Unmarshal(eventBz, txEvent)
			require.NoError(s, err)

			var found bool
			for _, event := range txEvent.Result.Events {
				for _, attribute := range event.Attributes {
					if attribute.Key == "action" {
						require.Equal(
							s, "/pocket.supplier.MsgCreateClaim",
							attribute.Value,
						)
						found = true
						break
					}
				}
				if found {
					break
				}
			}
			require.Truef(s, found, "unable to find event action attribute")

			done()
		},
	)

	select {
	case <-ctx.Done():
	case <-time.After(createClaimTimeoutDuration):
		s.Fatal("timed out waiting for claim to be created")
	}
}

func (s *suite) TheClaimCreatedBySupplierForServiceForApplicationShouldBePersistedOnchain(
	supplierName, serviceId, appName string,
) {
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

func (s *suite) TheSupplierHasServicedASessionWithRelaysForServiceForApplication(
	supplierName, relayCountStr, serviceId, appName string,
) {
	ctx := context.Background()

	relayCount, err := strconv.Atoi(relayCountStr)
	require.NoError(s, err)

	// Query for any existing claims so that we can compensate for them in the
	// future assertions about changes in on-chain claims.
	allClaimsRes, err := s.supplierQueryClient.AllClaims(ctx, &suppliertypes.QueryAllClaimsRequest{})
	require.NoError(s, err)
	s.scenarioState[preExistingClaimsKey] = allClaimsRes.Claim

	// Construct an events query client to listen for tx events from the supplier.
	msgSenderQuery := fmt.Sprintf(msgClaimSenderQueryFmt, accNameToAddrMap[supplierName])

	// TODO_TECHDEBT(#220): refactor to use EventsReplayClient once available.
	eventsQueryClient := events.NewEventsQueryClient(testclient.CometLocalWebsocketURL)
	eitherEventsBzObs, err := eventsQueryClient.EventsBytes(ctx, msgSenderQuery)
	require.NoError(s, err)

	eitherEventsBytesObs := observable.Observable[either.Bytes](eitherEventsBzObs)
	eitherEventsBzRelayObs := channel.ToReplayObservable(ctx, eitherEventsReplayBufferSize, eitherEventsBytesObs)
	s.scenarioState[eitherEventsBzReplayObsKey] = eitherEventsBzRelayObs

	s.sendRelaysForSession(
		appName,
		supplierName,
		testServiceId,
		relayCount,
	)
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
