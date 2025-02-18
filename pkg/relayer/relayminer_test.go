package relayer_test

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"path/filepath"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	"github.com/cometbft/cometbft/libs/os"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

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

type RelayMinerPingSuite struct {
	suite.Suite

	servedRelaysObs relayer.RelaysObservable
	minedRelaysObs  relayer.MinedRelaysObservable
}

func TestRelayMinerPingSuite(t *testing.T) {
	suite.Run(t, new(RelayMinerPingSuite))
}

func (t *RelayMinerPingSuite) SetupTest() {
	// servedRelaysObs is NEVER published to. It exists to satisfy test mocks.
	srObs, _ := channel.NewObservable[*servicetypes.Relay]()
	t.servedRelaysObs = relayer.RelaysObservable(srObs)

	// minedRelaysObs is NEVER published to. It exists to satisfy test mocks.
	mrObs, _ := channel.NewObservable[*relayer.MinedRelay]()
	t.minedRelaysObs = relayer.MinedRelaysObservable(mrObs)
}

func (t *RelayMinerPingSuite) TestOKPingAll() {
	ctx := polyzero.NewLogger().WithContext(context.Background())
	relayerProxyMock := testrelayer.NewMockOneTimeRelayerProxyWithPing(
		ctx, t.T(),
		t.servedRelaysObs,
	)

	minerMock := testrelayer.NewMockOneTimeMiner(
		ctx, t.T(),
		t.servedRelaysObs,
		t.minedRelaysObs,
	)

	relayerSessionsManagerMock := testrelayer.NewMockOneTimeRelayerSessionsManager(
		ctx, t.T(),
		t.minedRelaysObs,
	)

	deps := depinject.Supply(
		relayerProxyMock,
		minerMock,
		relayerSessionsManagerMock,
	)

	relayminer, err := relayer.NewRelayMiner(ctx, deps)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), relayminer)

	err = relayminer.Start(ctx)
	require.NoError(t.T(), err)

	time.Sleep(time.Millisecond)

	relayminerSocketPath := filepath.Join(t.T().TempDir(), "1d031ace")

	relayminer.ServePing(ctx, "unix", relayminerSocketPath)

	time.Sleep(time.Millisecond)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", relayminerSocketPath)
		},
	}

	// Override transport configuration to adapt the http client to the unix socket listener.
	httpClient := http.Client{Transport: transport}
	require.NoError(t.T(), err)

	resp, err := httpClient.Get("http://unix")
	require.NoError(t.T(), err)

	require.Equal(t.T(), http.StatusNoContent, resp.StatusCode)

	err = relayminer.Stop(ctx)
	require.NoError(t.T(), err)
}

func (t *RelayMinerPingSuite) TestNOKPingAllWithTemporaryError() {
	ctx := polyzero.NewLogger().WithContext(context.Background())
	relayerProxyMock := testrelayer.NewMockOneTimeRelayerProxy(ctx, t.T(), t.servedRelaysObs)

	urlErr := url.Error{
		Op:  http.MethodGet,
		URL: "http://unix",
		Err: &net.DNSError{
			Err:         "fake temporary and timeout error",
			Name:        "example.com",
			Server:      "8.8.8.8",
			IsTemporary: true,
			IsTimeout:   true,
		},
	}

	relayerProxyMock.EXPECT().PingAll(gomock.Eq(ctx)).
		Times(1).Return(&urlErr)

	minerMock := testrelayer.NewMockOneTimeMiner(
		ctx, t.T(),
		t.servedRelaysObs,
		t.minedRelaysObs,
	)

	relayerSessionsManagerMock := testrelayer.NewMockOneTimeRelayerSessionsManager(
		ctx, t.T(),
		t.minedRelaysObs,
	)

	deps := depinject.Supply(
		relayerProxyMock,
		minerMock,
		relayerSessionsManagerMock,
	)

	relayminer, err := relayer.NewRelayMiner(ctx, deps)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), relayminer)

	err = relayminer.Start(ctx)
	require.NoError(t.T(), err)

	time.Sleep(time.Millisecond)

	relayminerSocketPath := filepath.Join(t.T().TempDir(), "5478a402")

	err = relayminer.ServePing(ctx, "unix", relayminerSocketPath)
	require.NoError(t.T(), err)

	require.True(t.T(), os.FileExists(relayminerSocketPath))

	time.Sleep(time.Millisecond)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", relayminerSocketPath)
		},
	}

	// Override transport configuration to adapt the http client to the unix socket listener.
	httpClient := http.Client{Transport: transport}
	require.NoError(t.T(), err)

	resp, err := httpClient.Get("http://unix")
	require.NoError(t.T(), err)

	require.Equal(t.T(), http.StatusGatewayTimeout, resp.StatusCode)

	err = relayminer.Stop(ctx)
	require.NoError(t.T(), err)
}

func (t *RelayMinerPingSuite) NOKPingWithoutTemporaryError() {
	ctx := polyzero.NewLogger().WithContext(context.Background())
	relayerProxyMock := testrelayer.NewMockOneTimeRelayerProxy(ctx, t.T(), t.servedRelaysObs)

	relayerProxyMock.EXPECT().PingAll(gomock.Eq(ctx)).
		Times(1).Return(errors.New("fake"))

	minerMock := testrelayer.NewMockOneTimeMiner(
		ctx, t.T(),
		t.servedRelaysObs,
		t.minedRelaysObs,
	)

	relayerSessionsManagerMock := testrelayer.NewMockOneTimeRelayerSessionsManager(
		ctx, t.T(),
		t.minedRelaysObs,
	)

	deps := depinject.Supply(
		relayerProxyMock,
		minerMock,
		relayerSessionsManagerMock,
	)

	relayminer, err := relayer.NewRelayMiner(ctx, deps)
	require.NoError(t.T(), err)
	require.NotNil(t.T(), relayminer)

	err = relayminer.Start(ctx)
	require.NoError(t.T(), err)

	time.Sleep(time.Millisecond)

	relayminerSocketPath := filepath.Join(t.T().TempDir(), "aae252f8")

	relayminer.ServePing(ctx, "unix", relayminerSocketPath)

	time.Sleep(time.Millisecond)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", relayminerSocketPath)
		},
	}

	// Override transport configuration to adapt the http client to the unix socket listener.
	httpClient := http.Client{Transport: transport}
	require.NoError(t.T(), err)

	resp, err := httpClient.Get("http://unix")
	require.NoError(t.T(), err)

	require.Equal(t.T(), http.StatusBadGateway, resp.StatusCode)

	err = relayminer.Stop(ctx)
	require.NoError(t.T(), err)
}
