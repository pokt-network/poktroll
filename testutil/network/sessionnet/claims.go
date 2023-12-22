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
	"github.com/pokt-network/poktroll/testutil/network"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	"github.com/pokt-network/poktroll/x/supplier/client/cli"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// CreateClaims creates a valid claim and corresponding session tree for each
// supplier in the corresponding session for each application's first staked
// service.
func (memnet *inMemoryNetworkWithSessions) CreateClaims(
	t *testing.T,
) (claims []suppliertypes.Claim, sessionTrees []relayer.SessionTree) {
	// For NumSessions, get the session for each application in the application
	// module genesis state. Create a claim for each supplier in each session.
	for sessionIdx := 0; sessionIdx < memnet.Config.NumSessions; sessionIdx++ {
		appGenesisState := network.GetGenesisState[*apptypes.GenesisState](t, apptypes.ModuleName, memnet)

		var lastAppSession *sessiontypes.Session
		for _, application := range appGenesisState.ApplicationList {
			// Use applications first service because this in-memory network's application genesis
			// state is populated with applications whose first staked service matches that of
			// AppToSupplierPairingRatio, and whose second services matches no supplier.
			serviceId := application.GetServiceConfigs()[0].GetService().GetId()
			lastAppSession = memnet.GetSession(t, serviceId, application.GetAddress())

			for _, supplier := range lastAppSession.GetSuppliers() {
				claim, sessionTree := memnet.CreateClaim(
					t, supplier.GetAddress(),
					lastAppSession.GetHeader(),
				)
				claims = append(claims, *claim)
				sessionTrees = append(sessionTrees, sessionTree)

				// TODO_TECHDEBT(#196): Move this outside of the forloop so that the test iteration is faster
				// This interacts negatively with the session start/end heights in test scenarios. Test
				// assertions may have to account for this interaction and therefore may compromise on coverage.
				require.NoError(t, memnet.GetNetwork(t).WaitForNextBlock())
			}
		}
	}
	return claims, sessionTrees
}

// CreateClaim sends a tx using the test CLI to create an on-chain claim for the
// given supplier for the given session header.
func (memnet *inMemoryNetworkWithSessions) CreateClaim(
	t *testing.T,
	supplierAddr string,
	sessionHeader *sessiontypes.SessionHeader,
) (*suppliertypes.Claim, relayer.SessionTree) {
	t.Helper()

	clientCtx := memnet.GetClientCtx(t)
	net := memnet.GetNetwork(t)

	// Create a new session tree with NumRelaysPerSession number of relay nodes inserted.
	// Base64-encode it's root hash for use with the CLI command.
	sessionTree := newSessionTreeRoot(t, memnet.Config.NumRelaysPerSession, sessionHeader)
	rootHash, rootHashEncoded := getSessionTreeRoot(t, sessionTree)

	// Base64-encode the session header for use with the CLI command.
	sessionHeaderEncoded := cliEncodeSessionHeader(t, sessionHeader)
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
