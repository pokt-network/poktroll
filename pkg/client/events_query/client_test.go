package eventsquery_test

import (
	"context"
	"errors"
	"fmt"
	"pocket/pkg/client"
	"pocket/pkg/observable"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"pocket/internal/mocks/mockclient"
	"pocket/internal/testchannel"
	eventsquery "pocket/pkg/client/events_query"
)

/* TODO_TECHDEBT: add test coverage for:
- [x] Multiple observers w/ different queries
- [x] Observers close when connection closes unexpectedly
- [x] Observers close when context is cancelled
- [x] Observers close on #Close()
- [ ] Returns correct error channel (*assuming no `Either`)
- [ ] Multiple observers w/ same query
- [ ] Multiple observers w/ same query, one unsubscribes
*/

func TestQueryClient_Subscribe_Succeeds(t *testing.T) {
	var (
		readObserverEventsTimeout = 300 * time.Millisecond
		queryCounter              int
		// TODO_THIS_COMMIT: increase!
		queryLimit          = 5
		connMocks           = make([]*mockclient.MockConnection, queryLimit)
		ctrl                = gomock.NewController(t)
		rootCtx, cancelRoot = context.WithCancel(context.Background())
	)
	t.Cleanup(cancelRoot)

	dialerMock := mockclient.NewMockDialer(ctrl)
	// `Dialer#DialContext()` should be called once for each subscription (subtest).
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		DoAndReturn(func(_ context.Context, _ string) (*mockclient.MockConnection, error) {
			connMock := connMocks[queryCounter]

			queryCounter++
			return connMock, nil
		}).
		Times(queryLimit)

	// set up query client
	dialerOpt := eventsquery.WithDialer(dialerMock)
	queryClient := eventsquery.NewEventsQueryClient("", dialerOpt)
	t.Cleanup(queryClient.Close)

	for queryIdx := 0; queryIdx < queryLimit; queryIdx++ {
		t.Run(testQuery(queryIdx), func(t *testing.T) {
			var (
				// readEventCounter is the  number of obsvblConns which have been
				// received from the connection since the subtest started.
				readEventCounter int
				// handleEventsLimit is the total number of obsvblConns to send and
				// receive through the query client's observable for this subtest.
				handleEventsLimit = 1000
				connClosedMu      sync.Mutex
				// TODO_THIS_COMMIT: try...
				// connClosed atomic.Bool
				connClosed            bool
				queryCtx, cancelQuery = context.WithCancel(rootCtx)
			)

			// must set up connection mock before calling EventsBytes()
			connMock := mockclient.NewMockConnection(ctrl)
			// `Connection#Close()` should be called once for each subscription.
			connMock.EXPECT().Close().
				DoAndReturn(func() error {
					connClosedMu.Lock()
					defer connClosedMu.Unlock()

					connClosed = true
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
					connClosedMu.Lock()
					defer connClosedMu.Unlock()

					if connClosed {
						return nil, client.ErrConnClosed
					}

					event := testEvent(int32(readEventCounter))
					readEventCounter++
					return event, nil
				}).
				MinTimes(handleEventsLimit)
			connMocks[queryIdx] = connMock

			// set up query observer
			eventObservable, errCh := queryClient.EventsBytes(queryCtx, testQuery(queryIdx))
			eventObserver := eventObservable.Subscribe(queryCtx)

			onDone := func() {
				// cancelling the context should close the connection
				cancelQuery()
				// closing the connection happens asynchronously, so we need to wait a bit
				// for the connection to close to satisfy the connection mock expectations.
				time.Sleep(10 * time.Millisecond)

				// drain the observer channel and assert that it's closed
				err := testchannel.DrainChannel(eventObserver.Ch())
				require.NoError(t, err, "obsvblConns observer channel should be closed")
			}

			// concurrently consume obsvblConns from the observer channel
			behavesLikeObserver(
				t, eventObserver,
				handleEventsLimit,
				errCh,
				client.ErrConnClosed,
				readObserverEventsTimeout,
				onDone,
			)
		})
	}
}

func TestQueryClient_Subscribe_Close(t *testing.T) {
	var (
		readAllEventsTimeout = 50 * time.Millisecond
		//readAllEventsTimeout = 100 * time.Millisecond
		handleEventsLimit = 10
		readEventCounter  int
		connClosedMu      sync.Mutex
		// TODO_THIS_COMMIT: try...
		//connClosed           atomic.Bool
		connClosed bool
	)

	ctx := context.Background()
	ctrl := gomock.NewController(t)

	connMock := mockclient.NewMockConnection(ctrl)
	connMock.EXPECT().Close().
		DoAndReturn(func() error {
			connClosedMu.Lock()
			defer connClosedMu.Unlock()

			connClosed = true
			return nil
		}).
		MinTimes(1)
	connMock.EXPECT().Send(gomock.Any()).
		Return(nil).
		Times(1)
	connMock.EXPECT().Receive().
		DoAndReturn(func() (any, error) {
			connClosedMu.Lock()
			defer connClosedMu.Unlock()

			if connClosed {
				return nil, client.ErrConnClosed
			}

			event := testEvent(int32(readEventCounter))
			readEventCounter++
			return event, nil
		}).
		MinTimes(handleEventsLimit)

	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, nil).
		Times(1)

	dialerOpt := eventsquery.WithDialer(dialerMock)
	queryClient := eventsquery.NewEventsQueryClient("", dialerOpt)

	// set up query observer
	eventsObservable, errCh := queryClient.EventsBytes(ctx, testQuery(0))
	eventsObserver := eventsObservable.Subscribe(ctx)

	onDone := func() {
		// cancelling the context should close the connection
		queryClient.Close()
		// closing the connection happens asynchronously, so we need to wait a bit
		// for the connection to close to satisfy the connection mock expectations.
		time.Sleep(50 * time.Millisecond)
	}

	// concurrently consume obsvblConns from the observer channel
	behavesLikeObserver(
		t, eventsObserver,
		handleEventsLimit,
		errCh,
		client.ErrConnClosed,
		readAllEventsTimeout,
		onDone,
	)
}

func TestQueryClient_Subscribe_DialError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)

	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(nil, client.ErrDial).
		Times(1)

	dialerOpt := eventsquery.WithDialer(dialerMock)
	queryClient := eventsquery.NewEventsQueryClient("", dialerOpt)
	eventsObservable, errCh := queryClient.EventsBytes(ctx, testQuery(0))
	require.Nil(t, eventsObservable)

	select {
	case err := <-errCh:
		require.True(t, errors.Is(err, client.ErrDial))
	default:
		t.Fatalf("expected error from EventsBytes errCh")
	}
}

func TestQueryClient_Subscribe_RequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)

	connMock := mockclient.NewMockConnection(ctrl)
	connMock.EXPECT().Close().
		Return(nil).
		Times(1)
	connMock.EXPECT().Send(gomock.Any()).
		Return(fmt.Errorf("mock send error")).
		Times(1)

	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, nil).
		Times(1)

	dialerOpt := eventsquery.WithDialer(dialerMock)
	queryClient := eventsquery.NewEventsQueryClient("url_ignored", dialerOpt)
	eventsObservable, errCh := queryClient.EventsBytes(ctx, testQuery(0))
	require.Nil(t, eventsObservable)

	select {
	case err := <-errCh:
		require.True(t, errors.Is(err, client.ErrSubscribe))
	default:
		t.Fatalf("expected error from EventsBytes errCh")
	}

	// cancelling the context should close the connection
	cancel()
	// closing the connection happens asynchronously, so we need to wait a bit
	// for the connection to close to satisfy the connection mock expectations.
	time.Sleep(10 * time.Millisecond)
}

func TestQueryClient_Subscribe_ReceiveError(t *testing.T) {
	var (
		handleEventLimit     = 10
		readAllEventsTimeout = 100 * time.Millisecond
		readEventCounter     int
	)

	ctx, cancel := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)

	connMock := mockclient.NewMockConnection(ctrl)
	connMock.EXPECT().Close().
		Return(nil).
		Times(1)
	connMock.EXPECT().Send(gomock.Any()).
		Return(nil).
		Times(1)
	connMock.EXPECT().Receive().
		DoAndReturn(func() (any, error) {
			if readEventCounter >= handleEventLimit {
				return nil, client.ErrReceive
			}

			event := testEvent(int32(readEventCounter))
			readEventCounter++

			return event, nil
		}).
		MinTimes(handleEventLimit)

	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, nil).
		Times(1)

	dialerOpt := eventsquery.WithDialer(dialerMock)
	queryClient := eventsquery.NewEventsQueryClient("", dialerOpt)

	// set up query observer
	eventsObservable, errCh := queryClient.EventsBytes(ctx, testQuery(0))
	eventsObserver := eventsObservable.Subscribe(ctx)

	onLimit := func() {
		_ = cancel
		//// cancelling the context should close the connection
		//cancel()
		//// closing the connection happens asynchronously, so we need to wait a bit
		//// for the connection to close to satisfy the connection mock expectations.
		//time.Sleep(10 * time.Millisecond)
	}

	// concurrently consume obsvblConns from the observer channel
	behavesLikeObserver(
		t, eventsObserver,
		handleEventLimit,
		errCh,
		client.ErrReceive,
		readAllEventsTimeout,
		onLimit,
	)
}

func behavesLikeObserver(
	t *testing.T,
	observer observable.Observer[[]byte],
	eventsLimit int,
	errCh <-chan error,
	expectedErr error,
	timeout time.Duration,
	onDone func(),
) {
	var (
		//eventsCounterMu sync.RWMutex
		eventsCounter int32
		// done is used to signal when the test is complete
		done = make(chan struct{}, 1)
	)

	go func() {
		//defer eventsCounterMu.Unlock()
		for event := range observer.Ch() {
			//eventsCounterMu.Lock()
			currentEventCount := atomic.LoadInt32(&eventsCounter)
			if int(currentEventCount) >= eventsLimit {
				//eventsCounterMu.Unlock()
				done <- struct{}{}
				return
			}

			expectedEvent := testEvent(currentEventCount)
			require.Equal(t, expectedEvent, event)

			//log.Printf("incrementing from %d to %d", currentEventCount, eventsCounter+1)
			atomic.AddInt32(&eventsCounter, 1)

			// unbounded consumption here can result in the condition below never
			// being met due to the connection being closed before the "last" event
			// is received
			time.Sleep(time.Microsecond)
		}
	}()

	select {
	case <-done:
		//eventsCounterMu.RLock()
		require.Equal(t, eventsLimit, int(atomic.LoadInt32(&eventsCounter)))
		//eventsCounterMu.RUnlock()

		time.Sleep(10 * time.Millisecond)

		// TODO_RESUME_HERE!!!
		// is this right?
		if onDone != nil {
			onDone()
		}
	case err := <-errCh:
		switch expectedErr {
		case nil:
			require.NoError(t, err)
		default:
			require.ErrorIs(t, err, expectedErr)
		}
	case <-time.After(timeout):
		//eventsCounterMu.RLock()
		t.Fatalf(
			"timed out waiting for next event; expected %d events, got %d",
			eventsLimit, atomic.LoadInt32(&eventsCounter),
		)
		//eventsCounterMu.RUnlock()
	}

	err := testchannel.DrainChannel(observer.Ch())
	require.NoError(t, err, "obsvblConns observer should be closed")
}

func testEvent(idx int32) []byte {
	return []byte(fmt.Sprintf("message-%d", idx))
}

func testQuery(idx int) string {
	return fmt.Sprintf("query-%d", idx)
}
