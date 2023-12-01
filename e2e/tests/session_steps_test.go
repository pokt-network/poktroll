//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/stretchr/testify/require"

	eventsquery "github.com/pokt-network/poktroll/pkg/client/events_query"
	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/pkg/observable"
	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/testutil/testclient"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

const (
	createClaimTimeoutDuration   = 10 * time.Second
	eitherEventsReplayBufferSize = 100
)

var (
	eitherEventsBzReplayObsKey = "eitherEventsBzReplayObsKey"
	supplierAddressKey         = "supplierAddressKey"
)

func (s *suite) AfterTheSupplierCreatesAClaimForTheSession() {
	var ctx, done = context.WithCancel(context.Background())

	eitherEventsBzReplayObs := s.scenarioState[eitherEventsBzReplayObsKey].(observable.ReplayObservable[either.Bytes])

	channel.ForEach[either.Bytes](
		ctx, eitherEventsBzReplayObs,
		func(_ context.Context, eitherEventBz either.Bytes) {
			eventBz, err := eitherEventBz.ValueOrError()
			require.NoError(s, err)

			if strings.Contains(string(eventBz), "jsonrpc") {
				return
			}

			// Unmarshal byte data into a TxEvent object.
			// Try to deserialize the provided bytes into a TxEvent.
			err = json.Unmarshal(eventBz, &map[string]any{})
			require.NoError(s, err)

			var found bool
			// TODO_IN_THIS_COMMIT: improve or comment...
			for _, event := range ((map[string]any{"result": nil})["result"]).(map[string]any)["events"].([]any) {
				for _, attribute := range event.(map[string]any)["attributes"].([]any) {
					if attribute.(map[string]any)["key"] == "action" {
						require.Equal(
							s, "/pocket.supplier.MsgCreateClaim",
							attribute.(map[string]any)["value"],
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

func (s *suite) TheClaimCreatedBySupplierShouldBePersistedOnchain(supplierName string) {
	ctx := context.Background()

	// TODO_IN_THIS_COMMIT: set up in before hooks
	flagSet := testclient.NewLocalnetFlagSet(s)
	clientCtx := testclient.NewLocalnetClientCtx(s, flagSet)
	supplierQueryClient := suppliertypes.NewQueryClient(clientCtx)

	claimsRes, err := supplierQueryClient.AllClaims(ctx, &suppliertypes.QueryAllClaimsRequest{
		Filter: &suppliertypes.QueryAllClaimsRequest_SupplierAddress{
			SupplierAddress: accNameToAddrMap[supplierName],
		},
	})
	require.NoError(s, err)
	require.NotNil(s, claimsRes)

	// TODO_IN_THIS_COMMIT: query claims before this step, perhaps note the highest
	// session end height, then compare against queried claims in this step,
	// asserting the length is +1 and the highest session end height increased.
	//require.Lenf(s, claimsRes.Claim, 1, "expected 1 claim, got %d", len(claimsRes.Claim))
	require.NotEmpty(s, claimsRes.Claim)

	claim := claimsRes.Claim[0]
	require.Equal(s, accNameToAddrMap[supplierName], claim.SupplierAddress)
}

func (s *suite) TheSupplierHasServicedASessionOfRelaysForApplication(supplierName string, appName string) {
	// TODO_IN_THIS_COMMIT: use consts or something
	pocketNodeWebsocketUrl := "ws://pocket-sequencer:36657/websocket"
	msgClaimSenderQueryFmt := "tm.event='Tx' AND message.sender='%s'"
	msgSenderQuery := fmt.Sprintf(msgClaimSenderQueryFmt, accNameToAddrMap[supplierName])
	s.Logf("msgSenderQuery: %s", msgSenderQuery)
	ctx := context.Background()

	// TODO_TECHDEBT: refactor to use EventsReplayClient once available.
	eventsQueryClient := eventsquery.NewEventsQueryClient(pocketNodeWebsocketUrl)
	eitherEventsBzObs, err := eventsQueryClient.EventsBytes(ctx, msgSenderQuery)
	require.NoError(s, err)

	eitherEventsBytesObs := observable.Observable[either.Bytes](eitherEventsBzObs)
	eitherEventsBzRelayObs := channel.ToReplayObservable(ctx, eitherEventsReplayBufferSize, eitherEventsBytesObs)
	s.scenarioState[eitherEventsBzReplayObsKey] = eitherEventsBzRelayObs

	// TODO_IN_THIS_COMMMIT: use consts or something
	s.sendRelaysForSession(
		appName,
		supplierName,
		"anvil",
		5,
	)
}

// TODO_IN_THIS_COMMIT: rename
func (s *suite) sendRelaysForSession(
	appName string,
	supplierName string,
	serviceId string,
	relayLimit int,
) {
	s.TheApplicationIsStakedForService(appName, serviceId)
	s.TheSupplierIsStakedForService(supplierName, serviceId)
	s.TheSessionForApplicationAndServiceContainsTheSupplier(appName, serviceId, supplierName)

	// TODO_IN_THIS_COMMIT: something better
	data := `{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}`

	for i := 0; i < relayLimit; i++ {
		s.TheApplicationSendsTheSupplierARequestForServiceWithData(appName, supplierName, serviceId, data)
		s.TheApplicationReceivesASuccessfulRelayResponseSignedBy(appName, supplierName)
	}
}
