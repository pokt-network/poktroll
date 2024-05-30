package testqueryclients

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// NewTestSharedQueryClient creates a mock of the SharedQueryClient.
func NewTestSharedQueryClient(
	t *testing.T,
) *mockclient.MockSharedQueryClient {
	ctrl := gomock.NewController(t)

	sharedQuerier := mockclient.NewMockSharedQueryClient(ctrl)
	params := sharedtypes.DefaultParams()

	sharedQuerier.EXPECT().GetParams(gomock.Any()).
		Return(&params, nil).
		AnyTimes()

	return sharedQuerier
}
