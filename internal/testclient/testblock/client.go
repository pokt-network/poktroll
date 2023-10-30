package testblock

import (
	"context"
	"testing"

	"cosmossdk.io/depinject"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"

	"pocket/internal/mocks/mockclient"
	"pocket/internal/testclient"
	"pocket/internal/testclient/testeventsquery"
	"pocket/pkg/client"
	"pocket/pkg/client/block"
)

func NewLocalnetClient(ctx context.Context, t *testing.T) client.BlockClient {
	t.Helper()

	queryClient := testeventsquery.NewLocalnetClient(t)
	require.NotNil(t, queryClient)

	deps := depinject.Supply(queryClient)
	bClient, err := block.NewBlockClient(ctx, deps, testclient.CometLocalWebsocketURL)
	require.NoError(t, err)

	return bClient
}

func NewAnyTimeLatestBlockBlockClient(
	t *testing.T,
	blockHash []byte,
) *mockclient.MockBlockClient {
	t.Helper()
	ctrl := gomock.NewController(t)

	blockMock := mockclient.NewMockBlock(ctrl)
	blockMock.EXPECT().Height().Return(int64(1)).AnyTimes()
	blockMock.EXPECT().Hash().Return(blockHash).AnyTimes()
	blockClientMock := mockclient.NewMockBlockClient(ctrl)
	blockClientMock.EXPECT().LatestBlock(gomock.Any()).Return(blockMock)
	return blockClientMock
}
