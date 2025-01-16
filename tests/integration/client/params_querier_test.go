package client

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"cosmossdk.io/depinject"
	comethttp "github.com/cometbft/cometbft/rpc/client/http"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/client/block"
	"github.com/pokt-network/poktroll/pkg/client/events"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/pkg/client/query/cache"
	"github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	_ "github.com/pokt-network/poktroll/pkg/polylog/polyzero"
	"github.com/pokt-network/poktroll/testutil/e2e"
	"github.com/pokt-network/poktroll/testutil/testclient"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func TestSanity3(t *testing.T) {
	ctx := context.Background()
	//app := e2e.NewE2EApp(t)
	//t.Cleanup(func() { app.Close() })

	//clientConn := app.QueryHelper()
	//require.NotNil(t, clientConn)
	clientConn := testclient.NewLocalnetClientCtx(t, testclient.NewLocalnetFlagSet(t))

	sharedQueryClient := sharedtypes.NewSharedQueryClient(clientConn)
	params, err := sharedQueryClient.GetParams(ctx)
	require.NoError(t, err)
	require.NotNil(t, params)

	eventsQueryClient := events.NewEventsQueryClient("ws://127.0.0.1:26657/websocket")
	eventsBzObs, err := eventsQueryClient.EventsBytes(ctx, "tm.event='Tx'")
	require.NoError(t, err)

	t.Log("starting goroutine")

	errCh := make(chan error, 1)
	go func() {
		t.Log("in goroutine")

		eitherEventsBzCh := eventsBzObs.Subscribe(ctx).Ch()
		first := true
		for eitherEventBz := range eitherEventsBzCh {
			eventBz, err := eitherEventBz.ValueOrError()
			if err != nil {
				errCh <- err
				return
			}

			if eventBz == nil || first {
				first = false
				continue
			}

			t.Logf(">>> eventsBz: %s", string(eventBz))
			//t.Logf(">>> eventsBz(hex): %x", eventBz)
			break
		}

		close(errCh)
	}()

	t.Log("goroutine started")

	select {
	// TODO_IN_THIS_CASE: extract to testTimeoutDuration const.
	case <-time.After(15 * time.Second):
		t.Log("timeout")
		t.Fatalf("timed out waiting for events bytest observable to receive")
	case err = <-errCh:
		t.Log("done")
		require.NoError(t, err)
	}
}

func TestSanity2(t *testing.T) {
	ctx := context.Background()
	app := e2e.NewE2EApp(t)
	t.Cleanup(func() { app.Close() })

	clientConn, err := app.GetClientConn()
	require.NoError(t, err)
	require.NotNil(t, clientConn)

	sharedQueryClient := sharedtypes.NewSharedQueryClient(clientConn)
	params, err := sharedQueryClient.GetParams(ctx)
	require.NoError(t, err)
	require.NotNil(t, params)

	eventsQueryClient := events.NewEventsQueryClient(app.GetWSEndpoint())
	eventsBzObs, err := eventsQueryClient.EventsBytes(ctx, "tm.event='Tx'")
	require.NoError(t, err)

	errCh := make(chan error, 1)
	go func() {
		eitherEventsBzCh := eventsBzObs.Subscribe(ctx).Ch()
		for eitherEventBz := range eitherEventsBzCh {
			eventBz, err := eitherEventBz.ValueOrError()
			if err != nil {
				errCh <- err
				return
			}

			if eventBz == nil {
				continue
			}

			t.Logf(">>> eventsBz: %s", string(eventBz))
			t.Logf(">>> eventsBz(hex): %x", eventBz)
			break
		}

		close(errCh)
	}()

	//defaultCUTTM := sharedtypes.DefaultParams().ComputeUnitsToTokensMultiplier
	expectedCUTTM := uint64(99)

	//sharedParams, err := sharedQueryClient.GetParams(ctx)
	//require.NoError(t, err)
	//require.Equal(t, defaultCUTTM, sharedParams.ComputeUnitsToTokensMultiplier)

	paramUpdateMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      "compute_units_to_tokens_multiplier",
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedCUTTM},
	}

	_, err = app.RunMsg(t, paramUpdateMsg)
	require.NoError(t, err)

	time.Sleep(3 * time.Second)

	select {
	// TODO_IN_THIS_CASE: extract to testTimeoutDuration const.
	case <-time.After(10 * time.Second):
		t.Fatalf("timed out waiting for events bytest observable to receive")
	case err = <-errCh:
		require.NoError(t, err)
	}
}

// TODO_IN_THIS_COMMIT: godoc...
type MockParamsCache[P any] struct {
	mock.Mock
}

func (m *MockParamsCache[P]) GetLatestVersion(key string) (P, error) {
	args := m.Called(key)
	return args.Get(0).(P), args.Error(1)
}

func (m *MockParamsCache[P]) GetVersion(key string, version int64) (P, error) {
	args := m.Called(key, version)
	return args.Get(0).(P), args.Error(1)
}

func (m *MockParamsCache[P]) SetVersion(key string, value P, version int64) error {
	args := m.Called(key, value, version)
	return args.Error(0)
}

func TestSanity(t *testing.T) {
	ctx := context.Background()
	app := e2e.NewE2EApp(t)
	t.Cleanup(func() { app.Close() })

	clientConn, err := app.GetClientConn()
	require.NoError(t, err)
	require.NotNil(t, clientConn)

	eventsQueryClient := events.NewEventsQueryClient(app.GetWSEndpoint())
	require.NotNil(t, eventsQueryClient)

	// TODO_IN_THIS_COMMIT: add E2EApp#GetGRPCEndpoint() method.
	blockQueryClient, err := comethttp.New("tcp://127.0.0.1:42070", "/websocket")
	require.NoError(t, err)

	deps := depinject.Supply(
		eventsQueryClient,
		blockQueryClient,
	)
	blockClient, err := block.NewBlockClient(ctx, deps)
	require.NoError(t, err)

	logBuffer := new(bytes.Buffer)
	logger := polyzero.NewLogger(
		polyzero.WithLevel(polyzero.DebugLevel),
		polyzero.WithOutput(logBuffer),
	)

	paramsCache, err := cache.NewHistoricalInMemoryCache[*sharedtypes.Params]()
	require.NoError(t, err)

	// TODO_IN_THIS_COMMIT: replace polylog.Ctx with logger arg...
	ctx = logger.WithContext(ctx)
	deps = depinject.Configs(
		deps,
		depinject.Supply(
			logger,
			clientConn,
			//new(MockParamsCache[*sharedtypes.Params]),
			paramsCache,
			blockClient,
		),
	)

	moduleInfoOpt := query.WithModuleInfo(ctx, sharedtypes.ModuleName, sharedtypes.ErrSharedParamInvalid)
	paramsQuerier, err := query.NewCachedParamsQuerier[*sharedtypes.Params, sharedtypes.SharedQueryClient](
		ctx, deps,
		sharedtypes.NewSharedQueryClient,
		moduleInfoOpt,
	)

	require.NoError(t, err)

	defaultCUTTM := sharedtypes.DefaultParams().ComputeUnitsToTokensMultiplier
	expectedCUTTM := uint64(99)

	sharedParams, err := paramsQuerier.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, defaultCUTTM, sharedParams.ComputeUnitsToTokensMultiplier)

	paramUpdateMsg := &sharedtypes.MsgUpdateParam{
		Authority: authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		Name:      "compute_units_to_tokens_multiplier",
		AsType:    &sharedtypes.MsgUpdateParam_AsUint64{AsUint64: expectedCUTTM},
	}

	res, err := app.RunMsg(t, paramUpdateMsg)
	require.NoError(t, err)

	t.Logf("res: %+v", res)

	sharedParams, err = paramsQuerier.GetParams(ctx)
	require.NoError(t, err)
	require.Equal(t, int64(expectedCUTTM), int64(sharedParams.ComputeUnitsToTokensMultiplier))

	// Wait a tick to ensure the events query client observed the param update tx result.
	time.Sleep(100 * time.Millisecond)

	// TODO_IN_THIS_COMMIT: find a better way to assert that the cache was updated...
	// Consider mocking the cache implementation...
	t.Logf("\n%s", logBuffer.String())
	logLines := strings.Split(strings.Trim(logBuffer.String(), "\n"), "\n")
	require.Equal(t, 3, len(logLines))
	require.Contains(t, logLines[0], "cache miss")
	require.Contains(t, logLines[1], "cache hit")
	require.Contains(t, logLines[2], "cache hit")
}
