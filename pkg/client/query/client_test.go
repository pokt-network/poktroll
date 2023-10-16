package query_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"pocket/internal/mocks/mockclient"
	"pocket/pkg/client/query"
)

var (
	errConnClosed = errors.New("connection closed")
)

func TestQueryClient_Subscribe_Succeeds(t *testing.T) {
	const queryLimit = 5
	ctx, cancel := context.WithCancel(context.Background())

	for queryIdx := 0; queryIdx < queryLimit; queryIdx++ {
		t.Run(testQuery(queryIdx), func(t *testing.T) {
			var (
				readObserverEventsTimeout = 100 * time.Millisecond
				readEventCounter          int
				// number of events to send and receive through the query client's obervable
				handleEventsLimit   = 1000
				handleEventCounter  int
				maxHandleEventCount = handleEventsLimit * 103 / 100
				queryLimit          = 1
				connClosedMu        sync.Mutex
				connClosed          bool
			)

			ctx, cancel := context.WithCancel(ctx)
			ctrl := gomock.NewController(t)
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
			// `Connection#WriteJSON()` should be called once for each subscription.
			connMock.EXPECT().WriteJSON(gomock.Any()).
				Return(nil).
				Times(1)
			// `Connection#ReadEvent()` should be called once for each message plus
			// one as it blocks in the loop which calls msgHandler after reading the
			// last message.
			connMock.EXPECT().ReadEvent().
				DoAndReturn(func() ([]byte, error) {
					connClosedMu.Lock()
					defer connClosedMu.Unlock()

					if connClosed {
						return nil, errConnClosed
					}

					event := testEvent(readEventCounter)
					readEventCounter++
					return []byte(event), nil
				}).
				MaxTimes(maxHandleEventCount).
				MinTimes(handleEventsLimit)

			dialerMock := mockclient.NewMockDialer(ctrl)
			dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
				Return(connMock, nil).
				Times(queryLimit)

			dialerOpt := query.WithDialer(dialerMock)
			queryClient := query.NewQueryClient("", dialerOpt)

			done := make(chan struct{}, 1)
			observerIdx := 0
			eventObservable, errCh := queryClient.EventsObservable(ctx, testQuery(observerIdx))
			eventObserver := eventObservable.Subscribe(ctx)
			// IMPROVE: in-lining this call to #Ch() in the go routine below
			// concurrently with usage of drainCh causes a deadlock.
			eventObserverCh := eventObserver.Ch()

			go func() {
				for event := range eventObserverCh {
					require.Equal(t, testEvent(handleEventCounter), string(event))
					handleEventCounter++

					if handleEventCounter >= handleEventsLimit {
						done <- struct{}{}
						return
					}
				}
			}()

			select {
			case <-done:
				require.Equal(t, handleEventsLimit, handleEventCounter)
			case err := <-errCh:
				require.NoError(t, err)
				t.Fatal("unexpected receive on subscription error channel")
			case <-time.After(readObserverEventsTimeout):
				t.Fatalf(
					"timed out waiting for next message; expected %d messages, got %d",
					handleEventsLimit, handleEventCounter,
				)
			}

			// cancelling the context should close the connection
			cancel()
			// closing the connection happens asynchronously, so we need to wait a bit
			// for the connection to close to satisfy the connection mock expectations.
			time.Sleep(10 * time.Millisecond)

			closed, err := drainCh(eventObserverCh)
			require.True(t, closed)
			require.NoError(t, err)
		})
	}

	// TODO_RESUME_HERE!!!
	//err := queryClient.Close()
	//require.NoError(t, err)

	//_ = cancel
	// cancelling the context should close the connection
	cancel()
	// closing the connection happens asynchronously, so we need to wait a bit
	// for the connection to close to satisfy the connection mock expectations.
	time.Sleep(10 * time.Millisecond)
}

//func TestQueryClient_Subscribe_Close(t *testing.T) {
//	readMsgTimeout := 50 * time.Millisecond
//	handleMsgLimit := 100
//
//	ctx, cancel := context.WithCancel(context.Background())
//	ctrl := gomock.NewController(t)
//	var readMsgCounter, handleMsgCounter int
//
//	connMock := mockclient.NewMockConnection(ctrl)
//	connMock.EXPECT().Close().
//		Return(nil).
//		Times(1)
//	connMock.EXPECT().WriteJSON(gomock.Any()).
//		Return(nil).
//		Times(1)
//	connMock.EXPECT().ReadEvent().
//		DoAndReturn(func() ([]byte, error) {
//			msg := testEvent(readMsgCounter)
//			readMsgCounter++
//			return []byte(msg), nil
//		}).
//		Times(handleMsgLimit + 1)
//
//	dialerMock := mockclient.NewMockDialer(ctrl)
//	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
//		Return(connMock, nil).
//		Times(1)
//
//	dialerOpt := query.WithDialer(dialerMock)
//	queryClient := query.NewQueryClient("", dialerOpt)
//
//	//msgCh, done := make(chan []byte), make(chan struct{}, 1)
//	done := make(chan []byte)
//	msgHandler := func(ctx context.Context, msg []byte) error {
//		//msgCh <- msg
//		return nil
//	}
//	errCh := queryClient.EventsObservable(ctx, "query ignored in client mock", msgHandler)
//
//	//go func() {
//	//	for msg := range msgCh {
//	//		require.Equal(t, testEvent(handleMsgCounter), string(msg))
//	//		handleMsgCounter++
//	//
//	//		if handleMsgCounter >= handleMsgLimit {
//	//			done <- struct{}{}
//	//			return
//	//		}
//	//	}
//	//}()
//
//	select {
//	case <-done:
//		require.Equal(t, handleMsgLimit, handleMsgCounter)
//	case err := <-errCh:
//		require.NoError(t, err)
//		t.Fatal("unexpected receive on subscription error channel")
//	case <-time.After(readMsgTimeout):
//		t.Fatalf(
//			"timed out waiting for next message; expected %d messages, got %d",
//			handleMsgLimit, handleMsgCounter,
//		)
//	}
//
//	// cancelling the context should close the connection
//	cancel()
//	// closing the connection happens asynchronously, so we need to wait a bit
//	// for the connection to close to satisfy the connection mock expectations.
//	time.Sleep(10 * time.Millisecond)
//}

func TestQueryClient_Subscribe_DialError(t *testing.T) {
	ctx := context.Background()
	ctrl := gomock.NewController(t)
	errDialMock := errors.New("intentionally mocked dial error")

	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(nil, errDialMock).
		Times(1)

	dialerOpt := query.WithDialer(dialerMock)
	queryClient := query.NewQueryClient("", dialerOpt)
	eventsObservable, errCh := queryClient.EventsObservable(ctx, testQuery(0))
	require.Nil(t, eventsObservable)

	select {
	case err := <-errCh:
		require.True(t, errors.Is(err, errDialMock))
	default:
		t.Fatalf("expected error from EventsObservable errCh")
	}
}

func TestQueryClient_Subscribe_RequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	errMockConnWrite := errors.New("intentionally mocked write error")

	connMock := mockclient.NewMockConnection(ctrl)
	connMock.EXPECT().Close().
		Return(nil).
		Times(1)
	connMock.EXPECT().WriteJSON(gomock.Any()).
		Return(errMockConnWrite).
		Times(1)

	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, nil).
		Times(1)

	dialerOpt := query.WithDialer(dialerMock)
	queryClient := query.NewQueryClient("", dialerOpt)
	eventsObservable, errCh := queryClient.EventsObservable(ctx, testQuery(0))
	require.Nil(t, eventsObservable)

	select {
	case err := <-errCh:
		require.True(t, errors.Is(err, errMockConnWrite))
	default:
		t.Fatalf("expected error from EventsObservable errCh")
	}

	// cancelling the context should close the connection
	cancel()
	// closing the connection happens asynchronously, so we need to wait a bit
	// for the connection to close to satisfy the connection mock expectations.
	time.Sleep(10 * time.Millisecond)
}

func TestQueryClient_Subscribe_ConnectionClosedError(t *testing.T) {
	var (
		readAllEventsTimeout = 100 * time.Millisecond
		handleEventLimit     = 10
		readEventCounter     int
		handleEventCounter   int
		errMockConnClosed    = errors.New("intentionally mocked connection closed error")
		connClosedMu         sync.Mutex
		connClosed           bool
	)

	ctx, cancel := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)

	connMock := mockclient.NewMockConnection(ctrl)
	connMock.EXPECT().Close().
		DoAndReturn(func() error {
			connClosedMu.Lock()
			defer connClosedMu.Unlock()

			connClosed = true
			return nil
		}).
		Times(1)
	connMock.EXPECT().WriteJSON(gomock.Any()).
		Return(nil).
		Times(1)
	connMock.EXPECT().ReadEvent().
		DoAndReturn(func() ([]byte, error) {
			connClosedMu.Lock()
			defer connClosedMu.Unlock()

			if connClosed {
				return nil, errConnClosed
			}
			//_ = connClosed

			if readEventCounter >= handleEventLimit {
				return nil, errMockConnClosed
			}

			event := testEvent(readEventCounter)
			readEventCounter++
			return []byte(event), nil
		}).
		MinTimes(handleEventLimit)

	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, nil).
		Times(1)

	dialerOpt := query.WithDialer(dialerMock)
	queryClient := query.NewQueryClient("", dialerOpt)

	done := make(chan struct{}, 1)
	eventsObservable, errCh := queryClient.EventsObservable(ctx, testQuery(0))
	eventsObserver := eventsObservable.Subscribe(ctx)
	// IMPROVE: in-lining this call to #Ch() in the go routine below
	// concurrently with usage of drainCh causes a deadlock.
	eventsObserverCh := eventsObserver.Ch()

	go func() {
		for event := range eventsObserverCh {
			require.Equal(t, testEvent(handleEventCounter), string(event))
			handleEventCounter++

			if handleEventCounter >= handleEventLimit {
				done <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-done:
		fmt.Println("done")
		require.Equal(t, handleEventLimit, handleEventCounter)

		time.Sleep(10 * time.Millisecond)

		fmt.Println("selecting...")
		select {
		case err := <-errCh:
			fmt.Println("...errCh")
			require.True(t, errors.Is(err, errMockConnClosed))
		case <-time.After(readAllEventsTimeout):
			fmt.Println("...time.After")
			t.Fatalf("expected error: %s", errMockConnClosed.Error())
		}

		_ = readAllEventsTimeout
		//case <-time.After(readAllEventsTimeout):
		//	t.Fatalf(
		//		"timed out waiting for next message; expected %d messages, got %d",
		//		handleEventLimit, handleEventCounter,
		//	)
	}

	_ = cancel
	//// cancelling the context should close the connection
	//cancel()
	//// closing the connection happens asynchronously, so we need to wait a bit
	//// for the connection to close to satisfy the connection mock expectations.
	//time.Sleep(10 * time.Millisecond)

	fmt.Println("pre-drain")
	closed, err := drainCh(eventsObserverCh)
	fmt.Println("post-drain")
	require.Truef(t, closed, "events observer channel is not closed")
	require.NoError(t, err)
}

/* TODO_THIS_COMMIT: add test coverage for:
- [x] Multiple observers w/ different queries
- [ ] Multiple observers w/ same query
- [ ] Multiple observers w/ same query, one unsubscribes
- [x] Observers close when connection closes
- [ ] Observers close when context is cancelled
- [ ] Observers close on #Close()
- [ ] Returns correct error channel (*assuming no `Maybe`)
*/

func testEvent(idx int) string {
	return fmt.Sprintf("message-%d", idx)
}

func testQuery(idx int) string {
	return fmt.Sprintf("query-%d", idx)
}

// TODO_THIS_COMMIT: move & de-dup
func drainCh[V any](ch <-chan V) (closed bool, err error) {
	fmt.Println("draining")
	for {
		select {
		case _, ok := <-ch:
			if !ok {
				return true, nil
			}
			continue
		default:
			return false, fmt.Errorf("observer channel left open")
		}
	}
}
