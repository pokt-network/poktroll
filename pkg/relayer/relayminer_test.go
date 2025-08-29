package relayer_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	relayertypes "github.com/pokt-network/poktroll/pkg/relayer/types"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

func TestRelayMiner_StartAndStop(t *testing.T) {
	srObs, _ := channel.NewObservable[*servicetypes.Relay]()
	servedRelaysObs := relayer.RelaysObservable(srObs)

	mrObs, _ := channel.NewObservable[*relayertypes.MinedRelay]()
	minedRelaysObs := relayer.MinedRelaysObservable(mrObs)

	ctx := polyzero.NewLogger().WithContext(context.Background())
	relayerProxyMock := testrelayer.NewMockOneTimeRelayerProxy(
		ctx, t,
		servedRelaysObs,
	)

	minerMock := testrelayer.NewMockOneTimeMiner(
		ctx, t,
		servedRelaysObs,
		minedRelaysObs,
	)

	relayerSessionsManagerMock := testrelayer.NewMockOneTimeRelayerSessionsManager(
		ctx, t,
		minedRelaysObs,
	)

	deps := depinject.Supply(
		relayerProxyMock,
		minerMock,
		relayerSessionsManagerMock,
	)

	relayminer, err := relayer.NewRelayMiner(ctx, deps)
	require.NoError(t, err)
	require.NotNil(t, relayminer)

	require.NoError(t, relayminer.Start(ctx))

	time.Sleep(time.Millisecond)

	require.NoError(t, relayminer.Stop(ctx))
}
