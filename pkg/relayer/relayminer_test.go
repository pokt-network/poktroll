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
	"github.com/golang/mock/gomock"
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
	// servedRelaysObs is NEVER published to. It exists to satisfy test mocks.
	srObs, _ := channel.NewObservable[*servicetypes.Relay]()
	servedRelaysObs := relayer.RelaysObservable(srObs)

	// minedRelaysObs is NEVER published to. It exists to satisfy test mocks.
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

	relayminerSocketPath := filepath.Join(t.TempDir(), "relayerminer.ping.sock")

	relayminer.ServePing(ctx, "unix", relayminerSocketPath)

	time.Sleep(time.Millisecond)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", relayminerSocketPath)
		},
	}

	// Override transport configuration to adapt the http client to the unix socket listener.
	httpClient := http.Client{Transport: transport}
	require.NoError(t, err)

	resp, err := httpClient.Get("http://unix")
	require.NoError(t, err)

	require.Equal(t, http.StatusNoContent, resp.StatusCode)

	err = relayminer.Stop(ctx)
	require.NoError(t, err)
}

func TestRelayMiner_NOKPingTemporaryError(t *testing.T) {
	// servedRelaysObs is NEVER published to. It exists to satisfy test mocks.
	srObs, _ := channel.NewObservable[*servicetypes.Relay]()
	servedRelaysObs := relayer.RelaysObservable(srObs)

	// minedRelaysObs is NEVER published to. It exists to satisfy test mocks.
	mrObs, _ := channel.NewObservable[*relayer.MinedRelay]()
	minedRelaysObs := relayer.MinedRelaysObservable(mrObs)

	ctx := polyzero.NewLogger().WithContext(context.Background())
	relayerProxyMock := testrelayer.NewMockOneTimeRelayerProxy(ctx, t, servedRelaysObs)

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

	relayminerSocketPath := filepath.Join(t.TempDir(), "aae252f8-b19d-4bde-bd23-d8f2f6bf4011")

	relayminer.ServePing(ctx, "unix", relayminerSocketPath)

	time.Sleep(time.Millisecond)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", relayminerSocketPath)
		},
	}

	// Override transport configuration to adapt the http client to the unix socket listener.
	httpClient := http.Client{Transport: transport}
	require.NoError(t, err)

	resp, err := httpClient.Get("http://unix")
	require.NoError(t, err)

	require.Equal(t, http.StatusGatewayTimeout, resp.StatusCode)

	err = relayminer.Stop(ctx)
	require.NoError(t, err)
}

func TestRelayMiner_NOKPingNonTemporaryError(t *testing.T) {
	// servedRelaysObs is NEVER published to. It exists to satisfy test mocks.
	srObs, _ := channel.NewObservable[*servicetypes.Relay]()
	servedRelaysObs := relayer.RelaysObservable(srObs)

	// minedRelaysObs is NEVER published to. It exists to satisfy test mocks.
	mrObs, _ := channel.NewObservable[*relayer.MinedRelay]()
	minedRelaysObs := relayer.MinedRelaysObservable(mrObs)

	ctx := polyzero.NewLogger().WithContext(context.Background())
	relayerProxyMock := testrelayer.NewMockOneTimeRelayerProxy(ctx, t, servedRelaysObs)

	relayerProxyMock.EXPECT().PingAll(gomock.Eq(ctx)).
		Times(1).Return(errors.New("fake"))

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

	relayminerSocketPath := filepath.Join(t.TempDir(), "aae252f8-b19d-4bde-bd23-d8f2f6bf4011")

	relayminer.ServePing(ctx, "unix", relayminerSocketPath)

	time.Sleep(time.Millisecond)

	transport := &http.Transport{
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			return net.Dial("unix", relayminerSocketPath)
		},
	}

	// Override transport configuration to adapt the http client to the unix socket listener.
	httpClient := http.Client{Transport: transport}
	require.NoError(t, err)

	resp, err := httpClient.Get("http://unix")
	require.NoError(t, err)

	require.Equal(t, http.StatusBadGateway, resp.StatusCode)

	err = relayminer.Stop(ctx)
	require.NoError(t, err)
}
