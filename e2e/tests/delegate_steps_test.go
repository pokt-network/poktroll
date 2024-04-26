//go:build e2e

package e2e

import (
	"context"
	"regexp"

	"cosmossdk.io/depinject"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/tx"
	"github.com/pokt-network/poktroll/testutil/testclient"
	sessionkeeper "github.com/pokt-network/poktroll/x/session/keeper"
)

const (
	serviceId          = "anvil"
	defaultStakeAmount = 1000000
)

var (
	delegationRe = regexp.MustCompile(
		`applications:\s+-\s+address:\s+([a-zA-Z0-9]+)[\S\s]*delegatee_gateway_addresses:\s+-\s+([a-zA-Z0-9]+)`,
	)

	noDelegationRe = regexp.MustCompile(
		`applications:\s+-\s+address:\s+([a-zA-Z0-9]+)[\S\s]*delegatee_gateway_addresses:\s+\[\]`,
	)

	archivedDelegationRe = regexp.MustCompile(
		`applications:\s+-\s+address:\s+([a-zA-Z0-9]+)[\S\s]*archived_delegations:[\S\s]*-\s+gateway_addresses:[\S\s]*-\s+([a-zA-Z0-9]+)[\S\s]*last_active_block_height`,
	)
)

func (s *suite) TheApplicationIsStakedWithEnoughUpokt(accName string) {
	// TODO_TECHDEBT: This should be global to the whole feature
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.initEventsQueryClients(ctx)

	stakedAmount, ok := s.getStakedAmount("application", accName)
	if !ok {
		stakedAmount = defaultStakeAmount
	}

	s.TheUserSendsUpoktFromAccountToAccount(int64(stakedAmount+1), "pnf", accName)

	s.waitForTxResultEvent(
		"transfer",
		"recipient",
		accNameToAddrMap[accName],
	)

	s.TheUserStakesAWithUpoktForServiceFromTheAccount(
		"application",
		int64(stakedAmount+1),
		serviceId,
		accName,
	)

	s.TheUserShouldWaitForTheModuleMessageToBeSubmitted(
		"application",
		"StakeApplication",
	)
}

func (s *suite) TheGatewayIsStakedWithEnoughUpokt(accName string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.initEventsQueryClients(ctx)

	stakedAmount, ok := s.getStakedAmount("gateway", accName)
	if !ok {
		stakedAmount = defaultStakeAmount
	}

	s.TheUserSendsUpoktFromAccountToAccount(int64(stakedAmount+1), "pnf", accName)

	s.waitForTxResultEvent(
		"transfer",
		"recipient",
		accNameToAddrMap[accName],
	)

	s.TheUserStakesAWithUpoktForServiceFromTheAccount(
		"gateway",
		int64(stakedAmount+1),
		serviceId,
		accName,
	)

	s.TheUserShouldWaitForTheModuleMessageToBeSubmitted(
		"gateway",
		"StakeGateway",
	)
}

func (s *suite) TheUserDelegatesApplicationToGateway(appName, gatewayName string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.initEventsQueryClients(ctx)

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

func (s *suite) TheUserShouldSeeThatApplicationIsDelegatedToGateway(appName, gatewayName string) {
	args := []string{
		"q",
		"application",
		"list-application",
		chainIdFlag,
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	match := delegationRe.FindStringSubmatch(res.Stdout)
	require.Equal(s, len(match), 3, "app %q not delegated to gateway %q", appName, gatewayName)
	require.Equal(s, accNameToAddrMap[appName], match[1])
	require.Equal(s, accNameToAddrMap[gatewayName], match[2])
}

func (s *suite) ApplicationIsNotDelegatedToGateway(appName, gatewayName string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.initEventsQueryClients(ctx)

	args := []string{
		"q",
		"application",
		"list-application",
		chainIdFlag,
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)
	match := delegationRe.FindStringSubmatch(res.Stdout)

	if len(match) == 3 {
		s.TheUserUndelegatesApplicationFromGateway(appName, gatewayName)
		s.TheUserHasWaitedForTheBeginningOfTheNextSession()
	}

	res, err = s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	match = noDelegationRe.FindStringSubmatch(res.Stdout)
	require.Equal(s, len(match), 2, "app %q is delegated to a gateway", appName)
	require.Equal(s, accNameToAddrMap[appName], match[1])
}

func (s *suite) TheUserUndelegatesApplicationFromGateway(appName, gatewayName string) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.initEventsQueryClients(ctx)

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

	args = []string{
		"q",
		"application",
		"list-application",
		chainIdFlag,
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	match := noDelegationRe.FindStringSubmatch(res.Stdout)
	require.Equal(s, len(match), 2, "app %q is delegated to a gateway", appName)
	require.Equal(s, accNameToAddrMap[appName], match[1])
}

func (s *suite) TheUserHasWaitedForTheBeginningOfTheNextSession() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	s.initEventsQueryClients(ctx)

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

func (s *suite) TheUserShouldSeeThatApplicationHasGatewayAddressInTheArchivedDelegations(appName, gatewayName string) {
	args := []string{
		"q",
		"application",
		"list-application",
		chainIdFlag,
	}
	res, err := s.pocketd.RunCommandOnHost("", args...)
	require.NoError(s, err)

	match := archivedDelegationRe.FindStringSubmatch(res.Stdout)
	require.Equal(s, len(match), 3, "app %q does not have gateway %q in its archived delegations", appName, gatewayName)
	require.Equal(s, accNameToAddrMap[appName], match[1])
	require.Equal(s, accNameToAddrMap[gatewayName], match[2])
}

func (s *suite) initEventsQueryClients(ctx context.Context) {
	// Construct an events query client to listen for tx events from the supplier.
	deps := depinject.Supply(events.NewEventsQueryClient(testclient.CometLocalWebsocketURL))
	txEventsReplayClient, err := events.NewEventsReplayClient[*abci.TxResult](
		ctx,
		deps,
		"tm.event='Tx'",
		tx.UnmarshalTxResult,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)
	s.scenarioState[txResultEventsReplayClientKey] = txEventsReplayClient

	// Construct an events query client to listen for claim settlement or expiration events on-chain.
	blockEventsReplayClient, err := events.NewEventsReplayClient[*block.CometNewBlockEvent](
		ctx,
		deps,
		newBlockEventSubscriptionQuery,
		block.UnmarshalNewBlockEvent,
		eventsReplayClientBufferSize,
	)
	require.NoError(s, err)
	s.scenarioState[newBlockEventReplayClientKey] = blockEventsReplayClient
}
