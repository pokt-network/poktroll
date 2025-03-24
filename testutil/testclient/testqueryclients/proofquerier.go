package testqueryclients

import (
	"testing"

	"go.uber.org/mock/gomock"

	"github.com/pokt-network/pocket/pkg/client"
	"github.com/pokt-network/pocket/testutil/mockclient"
	prooftypes "github.com/pokt-network/pocket/x/proof/types"
)

// NewTestProofQueryClientWithParams creates a mock of the ProofQueryClient that
// uses the provided proof module params for its GetParams() method implementation.
func NewTestProofQueryClientWithParams(t *testing.T, params client.ProofParams) *mockclient.MockProofQueryClient {
	ctrl := gomock.NewController(t)
	proofQueryClientMock := mockclient.NewMockProofQueryClient(ctrl)
	proofQueryClientMock.EXPECT().
		GetParams(gomock.Any()).
		Return(params, nil).
		AnyTimes()

	return proofQueryClientMock
}

// NewTestProofQueryClient creates a mock of the ProofQueryClient which uses the
// default proof module params for its GetParams() method implementation.
func NewTestProofQueryClient(t *testing.T) *mockclient.MockProofQueryClient {
	defaultProofParams := prooftypes.DefaultParams()
	return NewTestProofQueryClientWithParams(t, &defaultProofParams)
}
