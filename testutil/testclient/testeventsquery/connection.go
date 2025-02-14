package testeventsquery

import (
	"sync/atomic"
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/pokt-network/poktroll/pkg/either"
	"github.com/pokt-network/poktroll/testutil/mockclient"
)

// NewOneTimeMockConnAndDialer returns a new mock connection and mock dialer that
// will return the mock connection when DialContext is called. The mock dialer
// will expect DialContext to be called exactly once. The connection mock will
// expect Close to be called exactly once.
// Callers must mock the Receive method with an EXPECT call before the connection
// mock can be used.
func NewOneTimeMockConnAndDialer(t *testing.T) (
	*mockclient.MockConnection,
	*mockclient.MockDialer,
) {
	ctrl := gomock.NewController(t)
	connMock := mockclient.NewMockConnection(ctrl)
	connMock.EXPECT().Close().
		Return(nil).
		Times(1)

	dialerMock := NewOneTimeMockDialer(t, either.Success(connMock))

	return connMock, dialerMock
}

// NewOneTimeMockDialer returns a mock dialer that will return either the given
// connection mock or error when DialContext is called. The mock dialer will
// expect DialContext to be called exactly once.
func NewOneTimeMockDialer(
	t *testing.T,
	eitherConnMock either.Either[*mockclient.MockConnection],
) *mockclient.MockDialer {
	ctrl := gomock.NewController(t)
	dialerMock := mockclient.NewMockDialer(ctrl)

	connMock, err := eitherConnMock.ValueOrError()
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, err).
		Times(1)

	return dialerMock
}

// NewNTimesReconnectMockConnAndDialer returns a new mock connection and mock
// dialer that will return the mock connection when DialContext is called. The
// mock dialer will expect DialContext to be called any times. The connection
// mock will expect Close and Send to be called exactly N times.
func NewNTimesReconnectMockConnAndDialer(
	t *testing.T,
	n int,
	connClosed *atomic.Bool,
	delayEvent *atomic.Bool,
) (*mockclient.MockConnection, *mockclient.MockDialer) {
	connMock := NewNTimesReconnectConnectionMock(t, n, connClosed, delayEvent)
	dialerMock := NewAnyTimesMockDailer(t, connMock)
	return connMock, dialerMock
}

// NewNTimesReconnectConnectionMock returns a mock connection that will expect
// Close and Send to be called exactly N times. The connection mock will set the
// connClosed atomic to true when Close is called and false when Send is called.
// The connection mock will set the delayEvent atomic to false when Send is
// called. This is to allow the caller to subscribe to the first event emitted
func NewNTimesReconnectConnectionMock(
	t *testing.T,
	n int,
	connClosed *atomic.Bool,
	delayEvent *atomic.Bool,
) *mockclient.MockConnection {
	ctrl := gomock.NewController(t)
	connMock := mockclient.NewMockConnection(ctrl)
	// Expect the connection to be closed and the dialer to be re-established
	connMock.EXPECT().
		Close().
		DoAndReturn(func() error {
			connClosed.CompareAndSwap(false, true)
			return nil
		}).
		Times(n)
	// Expect the subscription to be re-established any number of times
	connMock.EXPECT().
		Send(gomock.Any()).
		DoAndReturn(func(eventBz []byte) error {
			if connClosed.Load() {
				connClosed.CompareAndSwap(true, false)
			}
			delayEvent.CompareAndSwap(true, false)
			return nil
		}).
		Times(n)
	return connMock
}

// NewAnyTimesMockDailer returns a mock dialer that will return the given
// connection mock when DialContext is called. The mock dialer will expect
// DialContext to be called any number of times.
func NewAnyTimesMockDailer(
	t *testing.T,
	connMock *mockclient.MockConnection,
) *mockclient.MockDialer {
	ctrl := gomock.NewController(t)
	dialerMock := mockclient.NewMockDialer(ctrl)
	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, nil).
		AnyTimes()
	return dialerMock
}
