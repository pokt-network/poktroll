package testqueryclients

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/testutil/mockclient"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
)

// NewTestProofQueryClient creates a mock of the ProofQueryClient which uses the
// default proof module params for its GetParams() method implementation.
func NewTestProofQueryClient(t *testing.T) *mockclient.MockProofQueryClient {
	ctrl := gomock.NewController(t)
	defaultProofParams := prooftypes.DefaultParams()
	proofQueryClientMock := mockclient.NewMockProofQueryClient(ctrl)
	proofQueryClientMock.EXPECT().
		GetParams(gomock.Any()).
		Return(&defaultProofParams, nil).
		AnyTimes()

	return proofQueryClientMock
}
