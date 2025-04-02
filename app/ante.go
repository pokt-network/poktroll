package app

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/app/volatile"
)

// newAnteHandlerFn returns an AnteHandler that waives minimum gas/fees for transactions
// that contain ONLY morse claim messages (i.e. MsgClaimMorseAccount, MsgClaimMorseApplication,
// and MsgClaimMorseSupplier).
//
// TODO_MAINNET_CRITICAL(@bryanchriswhite):
// - Add a migration module param, `WaiveMorseClaimFees`.
// - Return the default antehandler if it is false.
func newAnteHandlerFn(app *App) cosmostypes.AnteHandler {
	return func(ctx cosmostypes.Context, tx cosmostypes.Tx, simulate bool) (cosmostypes.Context, error) {
		anteHandlerFn, err := ante.NewAnteHandler(ante.HandlerOptions{
			AccountKeeper:          &app.Keepers.AccountKeeper,
			BankKeeper:             app.Keepers.BankKeeper,
			ExtensionOptionChecker: nil,
			FeegrantKeeper:         app.Keepers.FeeGrantKeeper,
			SignModeHandler:        app.txConfig.SignModeHandler(),
			SigGasConsumer:         newSigVerificationGasConsumer(ctx, app, tx),
			TxFeeChecker:           newTxFeeChecker(ctx, app, tx),
		})
		if err != nil {
			return ctx, err
		}

		return anteHandlerFn(ctx, tx, simulate)
	}
}

// newSigVerificationGasConsumer returns a SignatureVerificationGasConsumer that
// returns zero gas fees for transactions which meet the following criteria:
// - Has ONLY one signer
// - Contains at least one message
// - Contains ONLY morse claim messages (i.e. MsgClaimMorseAccount, MsgClaimMorseApplication, and MsgClaimMorseSupplier)
func newSigVerificationGasConsumer(
	sdkCtx cosmostypes.Context,
	app *App,
	tx cosmostypes.Tx,
) ante.SignatureVerificationGasConsumer {
	// Use the freeSecp256k1SigGasConsumer if the tx:
	// - Has ONLY one signer
	// - Contains at least one message
	// - Contains ONLY Morse claim message(s)
	if txHasOneSecp256k1Signature(sdkCtx, app, tx) &&
		txHasOnlyMorseClaimMsgs(tx) {
		return freeSigGasConsumer
	}

	// Use the default signature verification gas consumer if ANY
	// non-morse-claim message type is included in the tx.
	return ante.DefaultSigVerificationGasConsumer
}

// newTxFeeChecker returns a TxFeeChecker that returns zero gas fees for
// transactions that contain ONLY morse claim messages (i.e. MsgClaimMorseAccount,
// MsgClaimMorseApplication, and MsgClaimMorseSupplier).
func newTxFeeChecker(
	sdkCtx cosmostypes.Context,
	app *App,
	tx cosmostypes.Tx,
) ante.TxFeeChecker {
	if txHasOneSecp256k1Signature(sdkCtx, app, tx) &&
		txHasOnlyMorseClaimMsgs(tx) {
		return freeTxFeeChecker
	}

	// When nil is returned, ante.checkTxFeeWithValidatorMinGasPrices is used by
	// default by cosmos-sdk.
	// See:
	// - https://github.com/cosmos/cosmos-sdk/blob/v0.50.13/x/auth/ante/fee.go#L31
	// - https://github.com/cosmos/cosmos-sdk/blob/v0.50.13/x/auth/ante/ante.go#L48
	return nil
}

// txHasOneSecp256k1Signature returns true if the given transaction contains
// only one secp256k1 signature. If an error occurs while parsing the tx, or
// retrieving the signer public key, it returns false (to fail safely).
func txHasOneSecp256k1Signature(sdkCtx cosmostypes.Context, app *App, tx cosmostypes.Tx) bool {
	accountKeeper := app.Keepers.AccountKeeper

	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return false
	}

	signers, err := sigTx.GetSigners()
	if err != nil {
		return false
	}

	// Ensure that the transaction has only one signer.
	if len(signers) != 1 {
		return false
	}

	// Retrieve the signer account from the account keeper.
	signerAcc, err := ante.GetSignerAcc(sdkCtx, accountKeeper, signers[0])
	if err != nil {
		return false
	}

	// Check if the signer's public key  is a secp256k1 public key.
	switch signerAcc.GetPubKey().(type) {
	case *secp256k1.PubKey:
		return true
	default:
	}

	return false
}

// txHasOnlyMorseClaimMsgs returns true if the given transaction contains ONLY
// morse claim messages (i.e. MsgClaimMorseAccount, MsgClaimMorseApplication, and
// MsgClaimMorseSupplier).
func txHasOnlyMorseClaimMsgs(tx cosmostypes.Tx) bool {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return false
	}

	for _, msg := range msgs {
		msgTypeUrl := cosmostypes.MsgTypeURL(msg)
		switch msgTypeUrl {
		case claimMorseAcctMsgTypeUrl,
			claimMorseAppMsgTypeUrl,
			claimMorseSupplierMsgTypeUrl:
			// check the remaining messages...
			continue
		default:
			return false
		}
	}

	// All messages were Morse claim messages.
	return true
}

// freeSigGasConsumer is a signature verification gas consumer that does not
// consume any gas for signature verification. It is intended to ONLY be applied
// to txs that consist of ONLY Morse claim messages.
func freeSigGasConsumer(
	_ storetypes.GasMeter,
	_ signing.SignatureV2,
	_ authtypes.Params,
) error {
	return nil
}

// freeTxFeeChecker is a TxFeeChecker that always returns zero gas fees
// (i.e. min gas price does not apply).
func freeTxFeeChecker(_ cosmostypes.Context, _ cosmostypes.Tx) (cosmostypes.Coins, int64, error) {
	return cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)), 0, nil
}
