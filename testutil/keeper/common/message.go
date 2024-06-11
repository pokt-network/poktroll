package common

import (
	"testing"

	"github.com/pokt-network/smt"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer"
	testsession "github.com/pokt-network/poktroll/testutil/session"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// NewTestClaimMsg returns a new MsgCreateClaim that can be submitted
// to be validated and stored on-chain.
func NewTestClaimMsg(
	t *testing.T,
	sessionStartHeight int64,
	sessionId string,
	supplierAddr string,
	appAddr string,
	service *sharedtypes.Service,
	merkleRoot smt.MerkleRoot,
) *prooftypes.MsgCreateClaim {
	t.Helper()

	return prooftypes.NewMsgCreateClaim(
		supplierAddr,
		&sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			Service:                 service,
			SessionId:               sessionId,
			SessionStartBlockHeight: sessionStartHeight,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(sessionStartHeight),
		},
		merkleRoot,
	)
}

// NewTestProofMsg creates a new submit proof message that can be submitted
// to be validated and stored on-chain.
func NewTestProofMsg(
	t *testing.T,
	supplierAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	sessionTree relayer.SessionTree,
	closestProofPath []byte,
) *prooftypes.MsgSubmitProof {
	t.Helper()

	// Generate a closest proof from the session tree using closestProofPath.
	merkleProof, err := sessionTree.ProveClosest(closestProofPath)
	require.NoError(t, err)
	require.NotNil(t, merkleProof)

	// Serialize the closest merkle proof.
	merkleProofBz, err := merkleProof.Marshal()
	require.NoError(t, err)

	return &prooftypes.MsgSubmitProof{
		SupplierAddress: supplierAddr,
		SessionHeader:   sessionHeader,
		Proof:           merkleProofBz,
	}
}
