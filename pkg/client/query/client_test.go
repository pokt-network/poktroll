package query_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"pocket/internal/mocks/mockclient"
	"pocket/pkg/client"
	"pocket/pkg/client/query"
)

func TestQueryClient_Subscribe_Succeeds(t *testing.T) {
	readMsgTimeout := 50 * time.Millisecond
	handleMsgLimit := 100
	subscriptionsLimit := 1

	//ctx := context.Background()
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	var readMsgCounter, handleMsgCounter int

	connMock := mockclient.NewMockConnection(ctrl)
	// `Connection#Close()` should be called once for each subscription.
	connMock.EXPECT().Close().
		Return(nil).
		Times(subscriptionsLimit)
	// `Connection#WriteJSON()` should be called once for each subscription.
	connMock.EXPECT().WriteJSON(gomock.Any()).
		Return(nil).
		Times(subscriptionsLimit)
	// `Connection#ReadMessage()` should be called once for each message plus
	// one as it blocks in the loop which calls msgHandler after reading the
	// last message.
	connMock.EXPECT().ReadMessage().
		DoAndReturn(func() ([]byte, error) {
			msg := messageContent(readMsgCounter)
			readMsgCounter++
			return []byte(msg), nil
		}).
		Times(handleMsgLimit + 1)

	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, nil).
		Times(subscriptionsLimit)

	dialerOpt := query.WithDialer(dialerMock)
	queryClient := query.NewQueryClient("", dialerOpt)

	msgCh, done := make(chan []byte), make(chan struct{}, 1)
	msgHandler := func(ctx context.Context, msg []byte) error {
		msgCh <- msg
		return nil
	}

	for subscriptionIdx := 0; subscriptionIdx < subscriptionsLimit; subscriptionIdx++ {
		errCh := queryClient.Subscribe(ctx, "query ignored in client mock", msgHandler)

		go func() {
			for msg := range msgCh {
				require.Equal(t, messageContent(handleMsgCounter), string(msg))
				handleMsgCounter++

				if handleMsgCounter >= handleMsgLimit {
					done <- struct{}{}
					return
				}
			}
		}()

		select {
		case <-done:
			require.Equal(t, handleMsgLimit, handleMsgCounter)
		case err := <-errCh:
			require.NoError(t, err)
			t.Fatal("unexpected receive on subscription error channel")
		case <-time.After(readMsgTimeout):
			t.Fatalf(
				"timed out waiting for next message; expected %d messages, got %d",
				handleMsgLimit, handleMsgCounter,
			)
		}
	}

	// TODO_RESUME_HERE!!!
	//err := queryClient.Close()
	//require.NoError(t, err)

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
//	connMock.EXPECT().ReadMessage().
//		DoAndReturn(func() ([]byte, error) {
//			msg := messageContent(readMsgCounter)
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
//	errCh := queryClient.Subscribe(ctx, "query ignored in client mock", msgHandler)
//
//	//go func() {
//	//	for msg := range msgCh {
//	//		require.Equal(t, messageContent(handleMsgCounter), string(msg))
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
		DoAndReturn(func(ctx context.Context, urlStr string) (client.Connection, error) {
			return nil, errDialMock
		}).
		Times(1)

	dialerOpt := query.WithDialer(dialerMock)
	queryClient := query.NewQueryClient("", dialerOpt)

	msgHandler := func(ctx context.Context, msg []byte) error {
		// noop
		return nil
	}

	errCh := queryClient.Subscribe(ctx, "query ignored in client mock", msgHandler)

	select {
	case err := <-errCh:
		require.True(t, errors.Is(err, errDialMock))
	default:
		t.Fatalf("expected error from Subscribe errCh")
	}
}

func TestQueryClient_Subscribe_RequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	errWriteMock := errors.New("intentionally mocked write error")

	connMock := mockclient.NewMockConnection(ctrl)
	connMock.EXPECT().Close().
		Return(nil).
		Times(1)
	connMock.EXPECT().WriteJSON(gomock.Any()).
		Return(errWriteMock).
		Times(1)

	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, nil).
		Times(1)

	dialerOpt := query.WithDialer(dialerMock)
	queryClient := query.NewQueryClient("", dialerOpt)

	msgHandler := func(ctx context.Context, msg []byte) error {
		// noop
		return nil
	}

	errCh := queryClient.Subscribe(ctx, "query ignored in client mock", msgHandler)

	select {
	case err := <-errCh:
		require.True(t, errors.Is(err, errWriteMock))
	default:
		t.Fatalf("expected error from Subscribe errCh")
	}

	// cancelling the context should close the connection
	cancel()
	// closing the connection happens asynchronously, so we need to wait a bit
	// for the connection to close to satisfy the connection mock expectations.
	time.Sleep(10 * time.Millisecond)
}

func TestQueryClient_Subscribe_ConnectionClosedError(t *testing.T) {
	readMsgTimeout := 50 * time.Millisecond
	handleMsgLimit := 10
	var readMsgCounter, handleMsgCounter int

	ctx, cancel := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	errReadMock := errors.New("intentionally mocked read error")

	connMock := mockclient.NewMockConnection(ctrl)
	connMock.EXPECT().Close().
		Return(nil).
		Times(1)
	connMock.EXPECT().WriteJSON(gomock.Any()).
		Return(nil).
		Times(1)
	connMock.EXPECT().ReadMessage().
		DoAndReturn(func() ([]byte, error) {
			if readMsgCounter >= handleMsgLimit {
				return nil, errReadMock
			}

			msg := messageContent(readMsgCounter)
			readMsgCounter++
			return []byte(msg), nil
		}).
		Times(handleMsgLimit + 1)

	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, nil).
		Times(1)

	dialerOpt := query.WithDialer(dialerMock)
	queryClient := query.NewQueryClient("", dialerOpt)

	msgCh, done := make(chan []byte), make(chan struct{}, 1)
	msgHandler := func(ctx context.Context, msg []byte) error {
		msgCh <- msg
		return nil
	}
	errCh := queryClient.Subscribe(ctx, "query ignored in client mock", msgHandler)

	go func() {
		for msg := range msgCh {
			require.Equal(t, messageContent(handleMsgCounter), string(msg))
			handleMsgCounter++

			if handleMsgCounter >= handleMsgLimit {
				done <- struct{}{}
				return
			}
		}
	}()

	select {
	case <-done:
		require.Equal(t, handleMsgLimit, handleMsgCounter)
	case err := <-errCh:
		require.NoError(t, err)
		t.Fatal("unexpected receive on subscription error channel")
	case <-time.After(readMsgTimeout):
		t.Fatalf(
			"timed out waiting for next message; expected %d messages, got %d",
			handleMsgLimit, handleMsgCounter,
		)
	}

	// cancelling the context should close the connection
	cancel()
	// closing the connection happens asynchronously, so we need to wait a bit
	// for the connection to close to satisfy the connection mock expectations.
	time.Sleep(10 * time.Millisecond)
}

func messageContent(i int) string {
	return fmt.Sprintf("message-%d", i)
}
