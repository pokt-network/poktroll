package relayer_test

import (
	"context"
	"net"
	"net/http"
	"os"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

func TestRelayMiner_StartAndStop(t *testing.T) {
	srObs, _ := channel.NewObservable[*servicetypes.Relay]()
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

func TestRelayMiner_Ping(t *testing.T) {
	srObs, _ := channel.NewObservable[*servicetypes.Relay]()
	servedRelaysObs := relayer.RelaysObservable(srObs)

	mrObs, _ := channel.NewObservable[*relayer.MinedRelay]()
	minedRelaysObs := relayer.MinedRelaysObservable(mrObs)

	ctx := polyzero.NewLogger().WithContext(context.Background())
	relayerProxyMock := testrelayer.NewMockOneTimeRelayerProxyWithPing(
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

	filename := "/tmp/relayerminer.ping.sock"

	ln, err := net.Listen("unix", filename)
	require.NoError(t, err)
	defer os.Remove(filename)

	relayminer.ServePing(ctx, ln)

	time.Sleep(time.Millisecond)

	c := http.Client{
		Transport: &http.Transport{
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return net.Dial(ln.Addr().Network(), ln.Addr().String())
			},
		},
	}
	require.NoError(t, err)

	_, err = c.Get("http://unix")
	require.NoError(t, err)

	err = relayminer.Stop(ctx)
	require.NoError(t, err)
}
