package testeventsquery

import (
	"pocket/pkg/either"
	"testing"

	"github.com/golang/mock/gomock"

	"pocket/internal/mocks/mockclient"
)

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

func OneTimeMockDialer(
	t *testing.T,
	eitherConnMock either.Either[*mockclient.MockConnection],
) *mockclient.MockDialer {
	connMock, err := eitherConnMock.ValueOrError()

	ctrl := gomock.NewController(t)
	dialerMock := mockclient.NewMockDialer(ctrl)

	dialerMock.EXPECT().DialContext(gomock.Any(), gomock.Any()).
		Return(connMock, err).
		Times(1)

	return dialerMock
}
