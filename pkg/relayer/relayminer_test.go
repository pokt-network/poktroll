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
	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
)

func TestRelayMiner_StartAndStop(t *testing.T) {
	srObs, _ := channel.NewObservable[*service.Relay]()
	servedRelaysObs := relayer.RelaysObservable(srObs)

	mrObs, _ := channel.NewObservable[*relayer.MinedRelay]()
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

	err = relayminer.Start(ctx)
	require.NoError(t, err)

	time.Sleep(time.Millisecond)

	err = relayminer.Stop(ctx)
	require.NoError(t, err)
}
