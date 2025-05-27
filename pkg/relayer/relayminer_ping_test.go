package relayer_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	cometbftos "github.com/cometbft/cometbft/libs/os"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/pokt-network/poktroll/pkg/observable/channel"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/testutil/testrelayer"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

// Helper to get a unique, short, cross-test socket path in /tmp
func tempUnixSocketPath(prefix string) string {
	return filepath.Join("/tmp", fmt.Sprintf("%s-%d.sock", prefix, time.Now().UnixNano()))
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

	require.NoError(t.T(), relayminer.Start(ctx))

	socketPath := tempUnixSocketPath("relayminer-okping")
	err = relayminer.ServePing(ctx, "unix", socketPath)
	require.NoError(t.T(), err)
	require.True(t.T(), cometbftos.FileExists(socketPath))

	defer os.Remove(socketPath)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
	httpClient := http.Client{Transport: transport}

	resp, err := httpClient.Get("http://unix")
	require.NoError(t.T(), err)
	require.Equal(t.T(), http.StatusNoContent, resp.StatusCode)

	require.NoError(t.T(), relayminer.Stop(ctx))
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

	require.NoError(t.T(), relayminer.Start(ctx))

	socketPath := tempUnixSocketPath("relayminer-errping")
	err = relayminer.ServePing(ctx, "unix", socketPath)
	require.NoError(t.T(), err)
	require.True(t.T(), cometbftos.FileExists(socketPath))

	defer os.Remove(socketPath)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
	httpClient := http.Client{Transport: transport}

	resp, err := httpClient.Get("http://unix")
	require.NoError(t.T(), err)
	require.Equal(t.T(), http.StatusGatewayTimeout, resp.StatusCode)

	require.NoError(t.T(), relayminer.Stop(ctx))
}

func (t *RelayMinerPingSuite) TestNOKPingWithoutTemporaryError() {
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

	require.NoError(t.T(), relayminer.Start(ctx))

	socketPath := tempUnixSocketPath("relayminer-badping")
	err = relayminer.ServePing(ctx, "unix", socketPath)
	require.NoError(t.T(), err)
	require.True(t.T(), cometbftos.FileExists(socketPath))

	defer os.Remove(socketPath)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", socketPath)
		},
	}
	httpClient := http.Client{Transport: transport}

	resp, err := httpClient.Get("http://unix")
	require.NoError(t.T(), err)
	require.Equal(t.T(), http.StatusBadGateway, resp.StatusCode)

	require.NoError(t.T(), relayminer.Stop(ctx))
}
