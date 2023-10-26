package testeventsquery

import (
	"pocket/pkg/either"
	"testing"

	"github.com/golang/mock/gomock"

	"pocket/internal/mocks/mockclient"
)

// OneTimeMockConnAndDialer returns a new mock connection and mock dialer that
// will return the mock connection when DialContext is called. The mock dialer
// will expect DialContext to be called exactly once. The connection mock will
// expect Close to be called exactly once.
// Callers must mock the Receive method with an EXPECT call before the connection
// mock can be used.
func OneTimeMockConnAndDialer(t *testing.T) (
	*mockclient.MockConnection,
	*mockclient.MockDialer,
) {
	ctrl := gomock.NewController(t)
	connMock := mockclient.NewMockConnection(ctrl)
	connMock.EXPECT().Close().
		Return(nil).
		Times(1)

	dialerMock := OneTimeMockDialer(t, either.Success(connMock))

	return connMock, dialerMock
}

// OneTimeMockDialer returns a mock dialer that will return either the given
// connection mock or error when DialContext is called. The mock dialer will
// expect DialContext to be called exactly once.
func OneTimeMockDialer(
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
