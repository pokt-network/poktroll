package events_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/pokt-network/pocket/pkg/client/events"
	"github.com/pokt-network/pocket/pkg/client/events/websocket"
	"github.com/pokt-network/pocket/pkg/either"
	"github.com/pokt-network/pocket/pkg/observable"
	"github.com/pokt-network/pocket/testutil/mockclient"
	"github.com/pokt-network/pocket/testutil/testchannel"
	"github.com/pokt-network/pocket/testutil/testclient/testeventsquery"
	"github.com/pokt-network/pocket/testutil/testerrors"
)

func TestEventsQueryClient_Subscribe_Succeeds(t *testing.T) {
	var (
		readObserverEventsTimeout = time.Second
		queryCounter              int
		queryLimit                = 5
		connMocks                 = make([]*mockclient.MockConnection, queryLimit)
		ctrl                      = gomock.NewController(t)
		rootCtx, cancelRoot       = context.WithCancel(context.Background())
	)
	t.Cleanup(cancelRoot)

	dialerMock := mockclient.NewMockDialer(ctrl)
	// `Dialer#DialContext()` should be called once for each subscription (subtest).
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string) (*mockclient.MockConnection, error) {
			// Return the connection mock for the subscription with the given query.
			// It should've been created in the respective test function.
			connMock := connMocks[queryCounter]
			queryCounter++
			return connMock, nil
		}).
		Times(queryLimit)

	// Set up events query client.
	dialerOpt := events.WithDialer(dialerMock)
	queryClient := events.NewEventsQueryClient("", dialerOpt)
	t.Cleanup(queryClient.Close)

	for queryIdx := 0; queryIdx < queryLimit; queryIdx++ {
		t.Run(testQuery(queryIdx), func(t *testing.T) {
			var (
				// ReadEventCounter is the  number of eventsBytesAndConns which have been
				// received from the connection since the subtest started.
				readEventCounter int
				// HandleEventsLimit is the total number of eventsBytesAndConns to send and
				// receive through the query client's eventsBytes for this subtest.
				handleEventsLimit = 250
				// delayFirstEvent runs once (per test case) to delay the first event
				// published by the mocked connection's Receive method to give the test
				// ample time to subscribe to the events bytes observable before it
				// starts receiving events, otherwise they will be dropped.
				delayFirstEvent       sync.Once
				connClosed            atomic.Bool
				queryCtx, cancelQuery = context.WithCancel(rootCtx)
			)

			// Must set up connection mock before calling EventsBytes()
			connMock := mockclient.NewMockConnection(ctrl)
			// `Connection#Close()` should be called once for each subscription.
			connMock.EXPECT().Close().
				DoAndReturn(func() error {
					// Simulate closing the connection.
					connClosed.CompareAndSwap(false, true)
					return nil
				}).
				Times(1)
			// `Connection#Send()` should be called once for each subscription.
			connMock.EXPECT().Send(gomock.Any()).
				Return(nil).
				Times(1)
			// `Connection#Receive()` should be called once for each message plus
			// one as it blocks in the loop which calls msgHandler after reading the
			// last message.
			connMock.EXPECT().Receive().
				DoAndReturn(func() (any, error) {
					delayFirstEvent.Do(func() { time.Sleep(50 * time.Millisecond) })

					// Simulate ErrConnClosed if connection is isClosed.
					if connClosed.Load() {
						return nil, events.ErrEventsConnClosed
					}

					event := testEvent(int32(readEventCounter))
					readEventCounter++

					// Simulate IO delay between sequential events.
					time.Sleep(10 * time.Microsecond)

					return event, nil
				}).
				MinTimes(handleEventsLimit)
			connMocks[queryIdx] = connMock

			// Set up events bytes observer for this query.
			eventObservable, err := queryClient.EventsBytes(queryCtx, testQuery(queryIdx))
			require.NoError(t, err)

			eventObserver := eventObservable.Subscribe(queryCtx)

			onLimit := func() {
				// Cancelling the context should close the connection.
				cancelQuery()
				// Closing the connection happens asynchronously, so we need to wait a bit
				// for the connection to close to satisfy the connection mock expectations.
				time.Sleep(10 * time.Millisecond)

				// Drain the observer channel and assert that it's isClosed.
				err := testchannel.DrainChannel(eventObserver.Ch())
				require.NoError(t, err, "eventsBytesAndConns observer channel should be isClosed")
			}

			// Concurrently consume eventsBytesAndConns from the observer channel.
			behavesLikeEitherObserver(
				t, eventObserver,
				handleEventsLimit,
				events.ErrEventsConnClosed,
				readObserverEventsTimeout,
				onLimit,
			)
		})
	}
}

func TestEventsQueryClient_Subscribe_Close(t *testing.T) {
	var (
		firstEventDelay      = 50 * time.Millisecond
		readAllEventsTimeout = 50*time.Millisecond + firstEventDelay
		handleEventsLimit    = 10
		readEventCounter     int
		// delayFirstEvent runs once (per test case) to delay the first event
		// published by the mocked connection's Receive method to give the test
		// ample time to subscribe to the events bytes observable before it
		// starts receiving events, otherwise they will be dropped.
		delayFirstEvent sync.Once
		connClosed      atomic.Bool
		ctx             = context.Background()
	)

	connMock, dialerMock := testeventsquery.NewOneTimeMockConnAndDialer(t)
	connMock.EXPECT().Send(gomock.Any()).Return(nil).
		Times(1)
	connMock.EXPECT().Receive().
		DoAndReturn(func() (any, error) {
			delayFirstEvent.Do(func() { time.Sleep(firstEventDelay) })

			if connClosed.Load() {
				return nil, events.ErrEventsConnClosed
			}

			event := testEvent(int32(readEventCounter))
			readEventCounter++

			// Simulate IO delay between sequential events.
			time.Sleep(10 * time.Microsecond)

			return event, nil
		}).
		MinTimes(handleEventsLimit)

	dialerOpt := events.WithDialer(dialerMock)
	queryClient := events.NewEventsQueryClient("", dialerOpt)

	// set up query observer
	eventsObservable, err := queryClient.EventsBytes(ctx, testQuery(0))
	require.NoError(t, err)

	eventsObserver := eventsObservable.Subscribe(ctx)

	onLimit := func() {
		// cancelling the context should close the connection
		queryClient.Close()
		// closing the connection happens asynchronously, so we need to wait a bit
		// for the connection to close to satisfy the connection mock expectations.
		time.Sleep(10 * time.Millisecond)
	}

	// concurrently consume eventsBytesAndConns from the observer channel
	behavesLikeEitherObserver(
		t, eventsObserver,
		handleEventsLimit,
		events.ErrEventsConnClosed,
		readAllEventsTimeout,
		onLimit,
	)
}

func TestEventsQueryClient_Subscribe_DialError(t *testing.T) {
	ctx := context.Background()

	eitherErrDial := either.Error[*mockclient.MockConnection](events.ErrEventsDial)
	dialerMock := testeventsquery.NewOneTimeMockDialer(t, eitherErrDial)

	dialerOpt := events.WithDialer(dialerMock)
	queryClient := events.NewEventsQueryClient("", dialerOpt)
	eventsObservable, err := queryClient.EventsBytes(ctx, testQuery(0))
	require.Nil(t, eventsObservable)
	require.True(t, errors.Is(err, events.ErrEventsDial))
}

func TestEventsQueryClient_Subscribe_RequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())

	connMock, dialerMock := testeventsquery.NewOneTimeMockConnAndDialer(t)
	connMock.EXPECT().Send(gomock.Any()).
		Return(fmt.Errorf("mock send error")).
		Times(1)

	dialerOpt := events.WithDialer(dialerMock)
	queryClient := events.NewEventsQueryClient("url_ignored", dialerOpt)
	eventsObservable, err := queryClient.EventsBytes(ctx, testQuery(0))
	require.Nil(t, eventsObservable)
	require.True(t, errors.Is(err, events.ErrEventsSubscribe))

	// cancelling the context should close the connection
	cancel()
	// closing the connection happens asynchronously, so we need to wait a bit
	// for the connection to close to satisfy the connection mock expectations.
	time.Sleep(10 * time.Millisecond)
}

// TODO_INVESTIGATE: why this test fails?
func TestEventsQueryClient_Subscribe_ReceiveError(t *testing.T) {
	t.Skip("TODO_INVESTIGATE: why this test fails")

	var (
		handleEventLimit     = 10
		readAllEventsTimeout = 100 * time.Millisecond
		readEventCounter     int
	)

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	connMock, dialerMock := testeventsquery.NewOneTimeMockConnAndDialer(t)
	connMock.EXPECT().Send(gomock.Any()).Return(nil).
		Times(1)
	connMock.EXPECT().Receive().
		DoAndReturn(func() (any, error) {
			if readEventCounter >= handleEventLimit {
				return nil, websocket.ErrEventsWebsocketReceive
			}

			event := testEvent(int32(readEventCounter))
			readEventCounter++
			time.Sleep(10 * time.Microsecond)

			return event, nil
		}).
		MinTimes(handleEventLimit)

	dialerOpt := events.WithDialer(dialerMock)
	queryClient := events.NewEventsQueryClient("", dialerOpt)

	// set up query observer
	eventsObservable, err := queryClient.EventsBytes(ctx, testQuery(0))
	require.NoError(t, err)

	eventsObserver := eventsObservable.Subscribe(ctx)
	// concurrently consume eventsBytesAndConns from the observer channel
	behavesLikeEitherObserver(
		t, eventsObserver,
		handleEventLimit,
		websocket.ErrEventsWebsocketReceive,
		readAllEventsTimeout,
		nil,
	)
}

// TODO_TECHDEBT: add test coverage for multiple observers with distinct and overlapping queries
func TestEventsQueryClient_EventsBytes_MultipleObservers(t *testing.T) {
	t.Skip("TODO_TECHDEBT: add test coverage for multiple observers with distinct and overlapping queries")
}

// behavesLikeEitherObserver asserts that the given observer behaves like an
// observable.Observer[either.Either[V]] by consuming notifications from the
// observer channel and asserting that they match the expected notification.
// It also asserts that the observer channel is isClosed after the expected number
// of eventsBytes have been received.
// If onLimit is not nil, it is called when the expected number of events have
// been received.
// Otherwise, the observer channel is drained and the test fails if it is not
// isClosed after the timeout duration.
func behavesLikeEitherObserver[V any](
	t *testing.T,
	observer observable.Observer[either.Either[V]],
	notificationsLimit int,
	expectedErr error,
	timeout time.Duration,
	onLimit func(),
) {
	t.Helper()

	var (
		// eventsCounter is the number of events which have been received from the
		// eventsBytes since this function was called.
		eventsCounter int32
		// errCh is used to signal when the test completes and/or produces an error
		errCh = make(chan error, 1)
	)

	go func() {
		for eitherEvent := range observer.Ch() {
			event, err := eitherEvent.ValueOrError()
			if err != nil {
				switch expectedErr {
				case nil:
					if !assert.NoError(t, err) {
						errCh <- testerrors.ErrAsync
						return
					}
				default:
					if !assert.ErrorIs(t, err, expectedErr) {
						errCh <- testerrors.ErrAsync
						return
					}
				}
			}

			currentEventCount := atomic.LoadInt32(&eventsCounter)
			if int(currentEventCount) >= notificationsLimit {
				// signal completion
				errCh <- nil
				return
			}

			// TODO_IMPROVE: to make this test helper more generic, it should accept
			// a generic function which generates the expected event for the given
			// index. Perhaps this function could use an either type which could be
			// used to consolidate the expectedErr and expectedEvent arguments.
			expectedEvent := testEvent(currentEventCount)
			// Require calls t.Fatal internally, which shouldn't happen in a
			// goroutine other than the test function's.
			// Use assert instead and stop the test by sending on errCh and
			// returning.
			if !assert.Equal(t, expectedEvent, event) {
				errCh <- testerrors.ErrAsync
				return
			}

			atomic.AddInt32(&eventsCounter, 1)

			// unbounded consumption here can result in the condition below never
			// being met due to the connection being isClosed before the "last" event
			// is received
			time.Sleep(10 * time.Microsecond)
		}
	}()

	select {
	case err := <-errCh:
		require.NoError(t, err)
		require.Equal(t, notificationsLimit, int(atomic.LoadInt32(&eventsCounter)))

		time.Sleep(10 * time.Millisecond)

		if onLimit != nil {
			onLimit()
		}
	case <-time.After(timeout):
		t.Fatalf(
			"timed out waiting for next event; expected %d events, got %d",
			notificationsLimit, atomic.LoadInt32(&eventsCounter),
		)
	}

	err := testchannel.DrainChannel(observer.Ch())
	require.NoError(t, err, "eventsBytesAndConns observer should be isClosed")
}

func testEvent(idx int32) []byte {
	return []byte(fmt.Sprintf("message_%d", idx))
}

func testQuery(idx int) string {
	return fmt.Sprintf("query_%d", idx)
}
