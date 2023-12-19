package sessionnet

import (
	"encoding/json"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/client/flags"
	testcli "github.com/cosmos/cosmos-sdk/testutil/cli"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/pkg/relayer"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/supplier/client/cli"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

func (memnet *inMemoryNetworkWithSessions) CreateClaims(
	t *testing.T,
) (claims []suppliertypes.Claim, sessionTrees []relayer.SessionTree) {
	// Create numSessions * numClaimsPerSession claims for the supplier
	sessionEndHeight := 1
	for sessionIdx := 0; sessionIdx < memnet.config.NumSessions; sessionIdx++ {
		sessionEndHeight += memnet.config.NumBlocksPerSession
		appGenesisState := GetGenesisState[*apptypes.GenesisState](t, apptypes.ModuleName, memnet)
		supplierGenesisState := GetGenesisState[*suppliertypes.GenesisState](t, suppliertypes.ModuleName, memnet)
		for appIdx, application := range appGenesisState.ApplicationList {
			for supplierIdx, supplier := range supplierGenesisState.SupplierList {
				// TODO_IN_THIS_COMMIT: comment...
				if appIdx != supplierIdx {
					continue
				}

				sessionStartHeight := sessionEndHeight - memnet.config.NumBlocksPerSession
				sessionHeader, _ := NewSessionHeader(
					t, memnet,
					fmt.Sprintf("svc%d", supplierIdx),
					application.GetAddress(),
					supplier.GetAddress(),
					int64(sessionStartHeight),
				)
				sessionTree := newSessionTreeRoot(t, memnet.config.NumRelaysPerSession, sessionHeader)

				claim, sessionTree := memnet.CreateClaim(
					t, supplier.GetAddress(),
					sessionHeader,
					sessionTree,
				)
				claims = append(claims, *claim)
				sessionTrees = append(sessionTrees, sessionTree)

				net := memnet.GetNetwork(t)
				// TODO_TECHDEBT(#196): Move this outside of the forloop so that the test iteration is faster
				require.NoError(t, net.WaitForNextBlock())
			}
		}
	}
	return claims, sessionTrees
}

// createClaim sends a tx using the test CLI to create an on-chain claim
func (memnet *inMemoryNetworkWithSessions) CreateClaim(
	t *testing.T,
	supplierAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	sessionTree relayer.SessionTree,
) (*suppliertypes.Claim, relayer.SessionTree) {
	t.Helper()

	clientCtx := memnet.GetClientCtx(t)
	net := memnet.GetNetwork(t)

	sessionHeaderEncoded := cliEncodeSessionHeader(t, sessionHeader)
	rootHash, rootHashEncoded := getSessionTreeRoot(t, sessionTree)

	args := []string{
		sessionHeaderEncoded,
		rootHashEncoded,
		fmt.Sprintf("--%s=%s", flags.FlagFrom, supplierAddr),
		fmt.Sprintf("--%s=true", flags.FlagSkipConfirmation),
		fmt.Sprintf("--%s=%s", flags.FlagBroadcastMode, flags.BroadcastSync),
		fmt.Sprintf("--%s=%s", flags.FlagFees, sdk.NewCoins(sdk.NewCoin(net.Config.BondDenom, math.NewInt(10))).String()),
	}

	responseRaw, err := testcli.ExecTestCLICmd(clientCtx, cli.CmdCreateClaim(), args)
	require.NoError(t, err)
	var responseJson map[string]interface{}
	err = json.Unmarshal(responseRaw.Bytes(), &responseJson)
	require.NoError(t, err)
	require.Equal(t, float64(0), responseJson["code"], "code is not 0 in the response: %v", responseJson)

	// TODO_TECHDEBT: Forward the actual claim in the response once the response is updated to return it.
	claim := &suppliertypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader:   sessionHeader,
		RootHash:        rootHash,
	}

	return claim, sessionTree
}
