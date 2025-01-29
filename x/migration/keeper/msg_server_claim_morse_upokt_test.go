package keeper_test

import (
	"encoding/binary"
	"testing"

	"cosmossdk.io/depinject"
	cometcrypto "github.com/cometbft/cometbft/crypto/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/client/query"
	"github.com/pokt-network/poktroll/testutil/integration"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/testutil/testmigration"
	"github.com/pokt-network/poktroll/x/migration/types"
)

func TestMsgServer_ClaimMorsePokt(t *testing.T) {
	app := integration.NewCompleteIntegrationApp(t)

	numAccounts := 10
	_, accountState := testmigration.NewMorseStateExportAndAccountState(t, numAccounts)

	// Ensure that the MorseAccountState is set initially.
	resAny, err := app.RunMsg(t, &types.MsgCreateMorseAccountState{
		Authority:         authtypes.NewModuleAddress(govtypes.ModuleName).String(),
		MorseAccountState: *accountState,
	})
	require.NoError(t, err)

	createStateRes, ok := resAny.(*types.MsgCreateMorseAccountStateResponse)
	require.True(t, ok)
	t.Logf("createStateRes: %+v", createStateRes)

	shannonDestAddr := sample.AccAddress()

	deps := depinject.Supply(app.QueryHelper())
	bankClient, err := query.NewBankQuerier(deps)
	require.NoError(t, err)

	balance, err := bankClient.GetBalance(app.GetSdkCtx(), shannonDestAddr)
	require.NoError(t, err)
	require.Equal(t, int64(0), balance.Amount.Int64())

	// TODO_IN_THIS_COMMIT: comment or refactor testutil...
	seedBz := make([]byte, 8)
	binary.LittleEndian.PutUint64(seedBz, uint64(1))
	privKey := cometcrypto.GenPrivKeyFromSecret(seedBz)
	morseDestAddr := privKey.PubKey().Address().String()
	require.Equal(t, morseDestAddr, accountState.Accounts[0].Address.String())

	morseClaimMsg := types.NewMsgClaimMorsePokt(shannonDestAddr, morseDestAddr, nil)
	morseClaimMsgUnsignedBz, err := proto.Marshal(morseClaimMsg)
	require.NoError(t, err)

	signature, err := privKey.Sign(morseClaimMsgUnsignedBz)
	require.NoError(t, err)

	morseClaimMsg.MorseSignature = signature
	resAny, err = app.RunMsg(t, morseClaimMsg)
	require.NoError(t, err)

	expectedBalance := sdk.NewInt64Coin(volatile.DenomuPOKT, 1110111)
	expectedStateHash, err := accountState.GetHash()
	require.NoError(t, err)

	claimAccountRes, ok := resAny.(*types.MsgClaimMorsePoktResponse)
	require.True(t, ok)
	require.Equal(t, expectedStateHash, claimAccountRes.GetStateHash())
	require.Equal(t, expectedBalance, claimAccountRes.GetBalance())

	balance, err = bankClient.GetBalance(app.GetSdkCtx(), shannonDestAddr)
	require.NoError(t, err)
	require.Equal(t, expectedBalance, balance)
}
