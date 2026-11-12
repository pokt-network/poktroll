package app_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	cometed25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/flags"
	clienttx "github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app"
	"github.com/pokt-network/poktroll/app/pocket"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

const anteTestChainID = "pocket"

func signZeroFeeTx(
	t *testing.T,
	ctx context.Context,
	txCfg client.TxConfig,
	priv *secp256k1.PrivKey,
	accNum uint64,
	msgs ...sdk.Msg,
) sdk.Tx {
	t.Helper()

	builder := txCfg.NewTxBuilder()
	require.NoError(t, builder.SetMsgs(msgs...))
	builder.SetGasLimit(200_000)
	builder.SetFeeAmount(sdk.NewCoins())

	signMode := signing.SignMode_SIGN_MODE_DIRECT
	require.NoError(t, builder.SetSignatures(signing.SignatureV2{
		PubKey: priv.PubKey(),
		Data:   &signing.SingleSignatureData{SignMode: signMode},
	}))
	signerData := authsigning.SignerData{
		ChainID:       anteTestChainID,
		AccountNumber: accNum,
		PubKey:        priv.PubKey(),
		Address:       sdk.AccAddress(priv.PubKey().Address()).String(),
	}
	sig, err := clienttx.SignWithPrivKey(ctx, signMode, signerData, builder, priv, txCfg, 0)
	require.NoError(t, err)
	require.NoError(t, builder.SetSignatures(sig))

	return builder.GetTx()
}

// The Morse claim fee waiver only applies to claims that can succeed: claiming
// enabled, and each msg claiming a distinct, existing, unclaimed Morse account.
func TestAnteHandler_MorseClaimFeeWaiver(t *testing.T) {
	appOptions := make(simtestutil.AppOptionsMap, 0)
	appOptions[flags.FlagHome] = t.TempDir()
	pocketApp, err := app.New(log.NewNopLogger(), dbm.NewMemDB(), nil, true, appOptions)
	require.NoError(t, err)

	ctx := pocketApp.NewUncachedContext(false, cmtproto.Header{
		ChainID: anteTestChainID,
		Height:  10,
		Time:    time.Unix(1_800_000_000, 0),
	})
	keepers := pocketApp.Keepers
	require.NoError(t, keepers.AccountKeeper.Params.Set(ctx, authtypes.DefaultParams()))
	require.NoError(t, keepers.BankKeeper.SetParams(ctx, banktypes.DefaultParams()))

	signerKey := secp256k1.GenPrivKey()
	signerAddr := sdk.AccAddress(signerKey.PubKey().Address())
	signerAcc := keepers.AccountKeeper.NewAccountWithAddress(ctx, signerAddr)
	keepers.AccountKeeper.SetAccount(ctx, signerAcc)

	newClaimMsg := func(morseKey cometed25519.PrivKey) *migrationtypes.MsgClaimMorseAccount {
		msg, err := migrationtypes.NewMsgClaimMorseAccount(signerAddr.String(), morseKey, signerAddr.String())
		require.NoError(t, err)
		return msg
	}
	importMorseAccount := func(morseKey cometed25519.PrivKey, claimedAtHeight int64) {
		keepers.MigrationKeeper.SetMorseClaimableAccount(ctx, migrationtypes.MorseClaimableAccount{
			MorseSrcAddress: newClaimMsg(morseKey).GetMorseSignerAddress(),
			UnstakedBalance: sdk.NewInt64Coin(pocket.DenomuPOKT, 1),
			ClaimedAtHeight: claimedAtHeight,
		})
	}

	unclaimedKey, claimedKey, unknownKey := cometed25519.GenPrivKey(), cometed25519.GenPrivKey(), cometed25519.GenPrivKey()
	importMorseAccount(unclaimedKey, 0)
	importMorseAccount(claimedKey, 5)

	minGasPrices := sdk.NewDecCoins(sdk.NewDecCoinFromDec(pocket.DenomuPOKT, math.LegacyMustNewDecFromStr("0.001")))
	checkCtx := ctx.WithIsCheckTx(true).WithExecMode(sdk.ExecModeCheck).WithMinGasPrices(minGasPrices)
	// Each tx runs on its own cache so an admitted tx does not bump the signer's sequence.
	runAnte := func(msgs ...sdk.Msg) error {
		txCtx, _ := checkCtx.CacheContext()
		_, err := pocketApp.AnteHandler()(txCtx, signZeroFeeTx(t, ctx, pocketApp.TxConfig(), signerKey, signerAcc.GetAccountNumber(), msgs...), false)
		return err
	}

	setParams := func(waive, claimingEnabled bool) {
		require.NoError(t, keepers.MigrationKeeper.SetParams(ctx, migrationtypes.NewParams(waive, false, claimingEnabled)))
	}
	setParams(true, true)

	require.NoError(t, runAnte(newClaimMsg(unclaimedKey)), "unclaimed Morse account: fee waived")
	require.ErrorContains(t, runAnte(banktypes.NewMsgSend(signerAddr, signerAddr, sdk.NewCoins())), "insufficient fee")
	require.ErrorContains(t, runAnte(newClaimMsg(unknownKey)), "insufficient fee", "unknown Morse account")
	require.ErrorContains(t, runAnte(newClaimMsg(claimedKey)), "insufficient fee", "already claimed Morse account")
	require.ErrorContains(t, runAnte(newClaimMsg(unclaimedKey), newClaimMsg(unclaimedKey)), "insufficient fee", "repeated Morse account")

	// Supplier claims are keyed by the Morse node address, and may all be signed
	// by the same Morse owner key on behalf of several nodes.
	morseOwnerKey := cometed25519.GenPrivKey()
	newSupplierClaimMsg := func(morseNodeAddress string) *migrationtypes.MsgClaimMorseSupplier {
		msg, err := migrationtypes.NewMsgClaimMorseSupplier(signerAddr.String(), signerAddr.String(), morseNodeAddress, morseOwnerKey, nil, signerAddr.String())
		require.NoError(t, err)
		return msg
	}
	nodeClaimA := newSupplierClaimMsg(cometed25519.GenPrivKey().PubKey().Address().String())
	nodeClaimB := newSupplierClaimMsg(cometed25519.GenPrivKey().PubKey().Address().String())
	for _, nodeClaim := range []*migrationtypes.MsgClaimMorseSupplier{nodeClaimA, nodeClaimB} {
		keepers.MigrationKeeper.SetMorseClaimableAccount(ctx, migrationtypes.MorseClaimableAccount{
			MorseSrcAddress:    nodeClaim.GetMorseNodeAddress(),
			SupplierStake:      sdk.NewInt64Coin(pocket.DenomuPOKT, 1),
			MorseOutputAddress: nodeClaim.GetMorseSignerAddress(),
		})
	}
	require.NoError(t, runAnte(nodeClaimA, nodeClaimB), "owner-signed supplier claims for two nodes: fee waived")
	unknownNodeClaim := newSupplierClaimMsg(cometed25519.GenPrivKey().PubKey().Address().String())
	require.ErrorContains(t, runAnte(unknownNodeClaim), "insufficient fee", "unknown Morse node")

	// The Morse signer of a supplier claim must be the claimed node or its output
	// address.
	foreignNodeClaim := newSupplierClaimMsg(cometed25519.GenPrivKey().PubKey().Address().String())
	keepers.MigrationKeeper.SetMorseClaimableAccount(ctx, migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:    foreignNodeClaim.GetMorseNodeAddress(),
		SupplierStake:      sdk.NewInt64Coin(pocket.DenomuPOKT, 1),
		MorseOutputAddress: cometed25519.GenPrivKey().PubKey().Address().String(),
	})
	require.ErrorContains(t, runAnte(foreignNodeClaim), "insufficient fee", "supplier claim signed by another key")
	require.ErrorContains(t, runAnte(nodeClaimA, foreignNodeClaim), "insufficient fee", "one such claim disables the waiver")

	// A custodial node (no output address) signing for itself is waived.
	morseNodeKey := cometed25519.GenPrivKey()
	custodialClaim, custodialErr := migrationtypes.NewMsgClaimMorseSupplier(signerAddr.String(), signerAddr.String(), morseNodeKey.PubKey().Address().String(), morseNodeKey, nil, signerAddr.String())
	require.NoError(t, custodialErr)
	keepers.MigrationKeeper.SetMorseClaimableAccount(ctx, migrationtypes.MorseClaimableAccount{
		MorseSrcAddress: custodialClaim.GetMorseNodeAddress(),
		SupplierStake:   sdk.NewInt64Coin(pocket.DenomuPOKT, 1),
	})
	require.NoError(t, runAnte(custodialClaim), "node-signed custodial supplier claim: fee waived")

	// A node signing for itself while a distinct output address is set (non-custodial) is waived too.
	nodeKey2 := cometed25519.GenPrivKey()
	nodeSignedClaim, nodeSignedErr := migrationtypes.NewMsgClaimMorseSupplier(signerAddr.String(), signerAddr.String(), nodeKey2.PubKey().Address().String(), nodeKey2, nil, signerAddr.String())
	require.NoError(t, nodeSignedErr)
	keepers.MigrationKeeper.SetMorseClaimableAccount(ctx, migrationtypes.MorseClaimableAccount{
		MorseSrcAddress:    nodeSignedClaim.GetMorseNodeAddress(),
		SupplierStake:      sdk.NewInt64Coin(pocket.DenomuPOKT, 1),
		MorseOutputAddress: cometed25519.GenPrivKey().PubKey().Address().String(),
	})
	require.NoError(t, runAnte(nodeSignedClaim), "node-signed non-custodial supplier claim: fee waived")

	setParams(true, false)
	require.ErrorContains(t, runAnte(newClaimMsg(unclaimedKey)), "insufficient fee", "claiming disabled")

	setParams(false, true)
	require.ErrorContains(t, runAnte(newClaimMsg(unclaimedKey)), "insufficient fee", "waiver disabled")
}
