package testqueryclients

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/testutil/mockclient"
)

// NewTestProofQueryClient creates a mock of the ProofQueryClient which uses the
// default proof module params for its GetParams() method implementation.
func NewTestProofQueryClient(t *testing.T) *mockclient.MockProofQueryClient {
	ctrl := gomock.NewController(t)
	defaultProofParams := proof.DefaultParams()
	proofQueryClientMock := mockclient.NewMockProofQueryClient(ctrl)
	proofQueryClientMock.EXPECT().
		GetParams(gomock.Any()).
		Return(&defaultProofParams, nil).
		AnyTimes()

	return proofQueryClientMock
}
