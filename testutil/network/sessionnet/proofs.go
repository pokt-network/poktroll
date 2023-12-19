package sessionnet

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/testutil/cli"
	types2 "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer"
	cli2 "github.com/pokt-network/poktroll/x/supplier/client/cli"
	"github.com/pokt-network/poktroll/x/supplier/types"
)

func (memnet *inMemoryNetworkWithSessions) SubmitProofs(
	t *testing.T,
	claims []types.Claim,
	sessionTrees []relayer.SessionTree,
) []types.Proof {
	t.Helper()

	var proofs []types.Proof
	for i, claim := range claims {
		proof := memnet.SubmitProof(t, claim, sessionTrees[i])
		proofs = append(proofs, *proof)

		net := memnet.GetNetwork(t)
		require.NoError(t, net.WaitForNextBlock())
	}
	return proofs
}

func (memnet *inMemoryNetworkWithSessions) SubmitProof(
	t *testing.T,
	claim types.Claim,
	sessionTree relayer.SessionTree,
) *types.Proof {
	t.Helper()

	merkelProof, err := sessionTree.ProveClosest(testProofPath)
	require.NoError(t, err)

	proofBz, err := merkelProof.Marshal()
	require.NoError(t, err)

	sessionHeaderEncoded := cliEncodeSessionHeader(t, claim.GetSessionHeader())
	proofEncoded := base64.StdEncoding.EncodeToString(proofBz)

	bondDenom := memnet.GetNetwork(t).Config.BondDenom
	args := []string{
		sessionHeaderEncoded,
		proofEncoded,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, claim.GetSupplierAddress()),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, types2.NewCoins(types2.NewCoin(bondDenom, math.NewInt(10))).String()),
	}

	ctx := memnet.GetClientCtx(t)
	responseRaw, err := cli.ExecTestCLICmd(ctx, cli2.CmdCreateClaim(), args)
	require.NoError(t, err)
	var responseJson map[string]interface{}
	err = json.Unmarshal(responseRaw.Bytes(), &responseJson)
	require.NoError(t, err)
	require.Equal(t, float64(0), responseJson["code"], "code is not 0 in the response: %v", responseJson)

	proof := &types.Proof{
		SupplierAddress: claim.GetSupplierAddress(),
		SessionHeader:   claim.GetSessionHeader(),
		MerkleProof:     proofBz,
	}

	return proof
}
