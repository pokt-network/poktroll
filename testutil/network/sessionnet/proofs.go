package sessionnet

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/cosmos/cosmos-sdk/client/flags"
	testcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer"
	"github.com/pokt-network/poktroll/x/supplier/client/cli"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

var testProofPath = []byte{1, 0, 1, 0, 1, 0}

// SubmitProofs generates and submits a proof for each claim in the provided
// list of claims. Claims are implicitly paired with session trees by index but are otherwise
// arbitrary (any session tree could be used for any claim).
//
// TODO_CONSIDERATION: This method could be refactored to accept a single list of
// objects which encapsulates both the claim and session tree.
func (memnet *inMemoryNetworkWithSessions) SubmitProofs(
	t *testing.T,
	claims []types.Claim,
	sessionTrees []relayer.SessionTree,
) []types.Proof {
	t.Helper()
	require.Equal(t, len(claims), len(sessionTrees), "number of claims and session trees must be equal")

	var proofs []types.Proof
	for i, claim := range claims {
		proof := memnet.SubmitProof(t, claim, sessionTrees[i])
		proofs = append(proofs, *proof)

		// TODO_TECHDEBT(#196): Move this outside of the forloop so that the test iteration is faster
		net := memnet.GetNetwork(t)
		require.NoError(t, net.WaitForNextBlock())
	}
	return proofs
}

// SubmitProof generates and submits a proof for the given claim and session tree.
func (memnet *inMemoryNetworkWithSessions) SubmitProof(
	t *testing.T,
	claim types.Claim,
	sessionTree relayer.SessionTree,
) *types.Proof {
	t.Helper()

	closestMerkleProof, err := sessionTree.ProveClosest(testProofPath)
	require.NoError(t, err)

	closestMerkleProofBz, err := closestMerkleProof.Marshal()
	require.NoError(t, err)

	sessionHeaderEncoded := cliEncodeSessionHeader(t, claim.GetSessionHeader())
	closestMerkleProofEncoded := base64.StdEncoding.EncodeToString(closestMerkleProofBz)
	args := []string{
		sessionHeaderEncoded,
		closestMerkleProofEncoded,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, claim.GetSupplierAddress()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, memnet.NewBondDenomCoins(t, 10).String()),
	}

	ctx := memnet.GetClientCtx(t)
	responseRaw, err := testcli.ExecTestCLICmd(ctx, cli.CmdSubmitProof(), args)
	require.NoError(t, err)

	var responseJson map[string]any
	err = json.Unmarshal(responseRaw.Bytes(), &responseJson)
	require.NoError(t, err)
	require.Equal(t, float64(0), responseJson["code"], "code is not 0 in the response: %v", responseJson)

	proof := &types.Proof{
		SupplierAddress:    claim.GetSupplierAddress(),
		SessionHeader:      claim.GetSessionHeader(),
		ClosestMerkleProof: closestMerkleProofBz,
	}

	return proof
}
