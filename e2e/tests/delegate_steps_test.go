//go:build e2e

package e2e

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"cosmossdk.io/depinject"
	"github.com/cometbft/cometbft/libs/json"
	"github.com/stretchr/testify/require"
	"golang.org/x/exp/slices"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/testutil/testclient"
	appkeeper "github.com/pokt-network/poktroll/x/application/keeper"
	"github.com/pokt-network/poktroll/x/application/types"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

const (
	// serviceId to stake for by applications.
	serviceId = "anvil"
	// defaultStakeAmount is the default amount to stake for applications and gateways.
	defaultStakeAmount = 1000000
)

func (s *suite) TheActorTypeWithAccountIsStakedWithEnoughUpokt(accType, accName string) {
	// Get the staked amount for account, if it exists.
	// This is used to determine how much to stake for the actor and whether
	// it is already staked or not.
	stakedAmount, ok := s.getStakedAmount(accType, accName)
	if !ok {
		stakedAmount = defaultStakeAmount
	}

	// Fund the actor with enough upokt to stake for the service.
	s.TheUserSendsUpoktFromAccountToAccount(int64(stakedAmount+1), "pnf", accName)
	s.waitForTxResultEvent(
		"transfer",
		"recipient",
		accNameToAddrMap[accName],
	)

	// Stake for the service with the
	s.TheUserStakesAWithUpoktForServiceFromTheAccount(
		accType,
		int64(stakedAmount+1),
		serviceId,
		accName,
	)
	s.TheUserShouldWaitForTheModuleMessageToBeSubmitted(
		accType,
		fmt.Sprintf("Stake%s", strings.Title(accType)),
	)
}

func (s *suite) TheUserDelegatesApplicationToGateway(appName, gatewayName string) {
	args := []string{
		"tx",
		"application",
		"delegate-to-gateway",
		accNameToAddrMap[gatewayName],
		"--from",
		appName,
		keyRingFlag,
		chainIdFlag,
		"-y",
	}
	_, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	s.TheUserShouldWaitForTheModuleMessageToBeSubmitted(
		"application",
		"DelegateToGateway",
	)
}

func (s *suite) TheApplicationDoesNotHaveAnyDelegation(appName string) {
	application := s.showApplication(appName)
	undelegationWaitGroup := sync.WaitGroup{}
	undelegationWaitGroup.Add(len(application.DelegateeGatewayAddresses))

	// Concurrently undelegate the application from all gateways and wait for the
	// transactions to be committed.
	for _, gatewayAddress := range application.DelegateeGatewayAddresses {
		go func(gatewayAddress string) {
			s.TheUserUndelegatesApplicationFromGateway(appName, accAddrToNameMap[gatewayAddress])
			s.TheUserWaitsUntilTheStartOfTheNextSession()
			undelegationWaitGroup.Done()
		}(gatewayAddress)
	}
	undelegationWaitGroup.Wait()
}

func (s *suite) ApplicationIsDelegatedToGateway(appName, gatewayName string) {
	application := s.showApplication(appName)
	require.Containsf(s,
		application.DelegateeGatewayAddresses,
		accNameToAddrMap[gatewayName],
		"app %q is not delegated to gateway %q",
		appName, gatewayName,
	)
}

func (s *suite) ApplicationIsNotDelegatedToGateway(appName, gatewayName string) {
	application := s.showApplication(appName)
	require.NotContainsf(s,
		application.DelegateeGatewayAddresses,
		accNameToAddrMap[gatewayName],
		"app %q is delegated to gateway %q",
		appName, gatewayName,
	)
}

func (s *suite) TheUserUndelegatesApplicationFromGatewayBeforeTheSessionEndBlock(appName, gatewayName string) {
	s.TheUserWaitsUntilTheStartOfTheNextSession()
	s.TheUserUndelegatesApplicationFromGateway(appName, gatewayName)
}

func (s *suite) TheUserUndelegatesApplicationFromGateway(appName, gatewayName string) {
	args := []string{
		"tx",
		"application",
		"undelegate-from-gateway",
		accNameToAddrMap[gatewayName],
		"--from",
		appName,
		keyRingFlag,
		chainIdFlag,
		"-y",
	}
	_, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	s.TheUserShouldWaitForTheModuleMessageToBeSubmitted(
		"application",
		"UndelegateFromGateway",
	)
}

func (s *suite) ApplicationHasGatewayAddressInTheArchivedDelegations(appName, gatewayName string) {
	application := s.showApplication(appName)
	require.Truef(s,
		slices.ContainsFunc(application.ArchivedDelegations,
			func(archivedDelegations types.ArchivedDelegations) bool {
				return slices.Contains(archivedDelegations.GatewayAddresses, accNameToAddrMap[gatewayName])
			}),
		"app %q does not have gateway %q in its archived delegations",
		appName, gatewayName,
	)
}

func (s *suite) ApplicationDoesNotHaveGatewayAddressInTheArchivedDelegations(appName, gatewayName string) {
	application := s.showApplication(appName)
	require.Falsef(s,
		slices.ContainsFunc(application.ArchivedDelegations,
			func(archivedDelegations types.ArchivedDelegations) bool {
				return slices.Contains(archivedDelegations.GatewayAddresses, accNameToAddrMap[gatewayName])
			}),
		"app %q has gateway %q in its archived delegations",
		appName, gatewayName,
	)
}

func (s *suite) ThePoktrollChainIsReachable() {
	ctx := context.Background()

	// Construct an events query client to listen for tx events from the supplier.
	deps := depinject.Supply(events.NewEventsQueryClient(testclient.CometLocalWebsocketURL))
	txEventsReplayClient, err := events.NewEventsReplayClient(
		ctx,
		deps,
		"tm.event='Tx'",
		tx.UnmarshalTxResult,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)
	s.scenarioState[txResultEventsReplayClientKey] = txEventsReplayClient

	// Construct an events query client to listen for claim settlement or expiration events on-chain.
	blockEventsReplayClient, err := events.NewEventsReplayClient(
		ctx,
		deps,
		newBlockEventSubscriptionQuery,
		block.UnmarshalNewBlockEvent,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)
	s.scenarioState[newBlockEventReplayClientKey] = blockEventsReplayClient
}

func (s *suite) TheUserWaitsUntilTheStartOfTheNextSession() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	blockReplayClient := s.scenarioState[newBlockEventReplayClientKey].(client.EventsReplayClient[*block.CometNewBlockEvent])
	block := blockReplayClient.LastNEvents(ctx, 1)[0]
	nextSessionStartHeight := sessionkeeper.GetSessionEndBlockHeight(block.Height()) + 1

	blockObs := blockReplayClient.EventsSequence(ctx).Subscribe(ctx)
	for newBlock := range blockObs.Ch() {
		if newBlock.Height() >= nextSessionStartHeight {
			break
		}
	}
}

func (s *suite) TheUserWaitsUntilArchivedDelegationsArePruned() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	blockReplayClient := s.scenarioState[newBlockEventReplayClientKey].(client.EventsReplayClient[*block.CometNewBlockEvent])
	block := blockReplayClient.LastNEvents(ctx, 1)[0]
	delegationPruningBlockHeight := block.Height() + appkeeper.ArchivedDelegationsRetentionBlocks

	blockObs := blockReplayClient.EventsSequence(ctx).Subscribe(ctx)
	for newBlock := range blockObs.Ch() {
		if newBlock.Height() >= delegationPruningBlockHeight {
			break
		}
	}
}

func (s *suite) showApplication(appName string) types.Application {
	args := []string{
		"q",
		"application",
		"show-application",
		accNameToAddrMap[appName],
		chainIdFlag,
		"--output",
		"json",
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)
	var queryGetApplicationResponse types.QueryGetApplicationResponse
	err = json.Unmarshal([]byte(res.Stdout), &queryGetApplicationResponse)
	require.NoError(s, err)

	return queryGetApplicationResponse.Application
}
