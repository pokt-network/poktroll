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
// that contain ONLY morse claim messages.
// I.e. MsgClaimMorseAccount, MsgClaimMorseApplication and MsgClaimMorseSupplier
func newMorseClaimGasFeesWaiverAnteHandlerFn(app *App) cosmostypes.AnteHandler {
	return func(sdkCtx cosmostypes.Context, tx cosmostypes.Tx, simulate bool) (cosmostypes.Context, error) {
		anteHandlerFn, err := ante.NewAnteHandler(ante.HandlerOptions{
			AccountKeeper:          &app.Keepers.AccountKeeper,
			BankKeeper:             app.Keepers.BankKeeper,
			ExtensionOptionChecker: nil,
			FeegrantKeeper:         app.Keepers.FeeGrantKeeper,
			SignModeHandler:        app.txConfig.SignModeHandler(),
			SigGasConsumer:         newSigVerificationGasConsumer(sdkCtx, app, tx),
			TxFeeChecker:           newTxFeeChecker(sdkCtx, app, tx),
		})
		if err != nil {
			return sdkCtx, err
		}

		return anteHandlerFn(sdkCtx, tx, simulate)
	}
}

// newSigVerificationGasConsumer returns a SignatureVerificationGasConsumer that
// returns zero gas fees for transactions which meet the following criteria:
// - Has EXACTLY one signer
// - Contains AT LEAST one message
// - Contains ONLY morse claim messages (i.e. MsgClaimMorseAccount, MsgClaimMorseApplication, and MsgClaimMorseSupplier)
func newSigVerificationGasConsumer(
	sdkCtx cosmostypes.Context,
	app *App,
	tx cosmostypes.Tx,
) ante.SignatureVerificationGasConsumer {
	// Use the freeSecp256k1SigGasConsumer if:
	// - The waive_morse_claim_gas_fees migration module param is true AND the tx:
	// - Has EXACTLY one signer
	// - Contains at least one message
	// - Contains ONLY Morse claim message(s)
	if shouldWaiveMorseClaimGasFees(sdkCtx, app, tx) {
		return freeSigGasConsumer
	}

	// Use the default signature verification gas consumer if ANY
	// non-morse-claim message type is included in the tx.
	return ante.DefaultSigVerificationGasConsumer
}

// newTxFeeChecker returns a TxFeeChecker that returns zero gas fees for
// transactions that contain ONLY morse claim messages.
// I.e. MsgClaimMorseAccount, MsgClaimMorseApplication and MsgClaimMorseSupplier
func newTxFeeChecker(
	sdkCtx cosmostypes.Context,
	app *App,
	tx cosmostypes.Tx,
) ante.TxFeeChecker {
	if shouldWaiveMorseClaimGasFees(sdkCtx, app, tx) {
		return freeTxFeeChecker
	}

	// When nil is returned, ante.checkTxFeeWithValidatorMinGasPrices is used by
	// default by cosmos-sdk.
	// See:
	// - https://github.com/cosmos/cosmos-sdk/blob/v0.50.13/x/auth/ante/fee.go#L31
	// - https://github.com/cosmos/cosmos-sdk/blob/v0.50.13/x/auth/ante/ante.go#L48
	return nil
}

// shouldWaiveMorseClaimGasFees returns true if:
//   - The waive_morse_claim_gas_fees migration module param is true
//   - The tx contains EXACTLY ONE secp256k1 signature
//   - The tx contains ONLY morse claim messages
//     I.e., MsgClaimMorseAccount, MsgClaimMorseApplication, and MsgClaimMorseSupplier
func shouldWaiveMorseClaimGasFees(sdkCtx cosmostypes.Context, app *App, tx cosmostypes.Tx) bool {
	migrationParams := app.Keepers.MigrationKeeper.GetParams(sdkCtx)

	return migrationParams.WaiveMorseClaimGasFees &&
		txHasOneSecp256k1Signature(tx) &&
		txHasOnlyMorseClaimMsgs(tx)
}

// txHasOneSecp256k1Signature returns true if the given tx contains EXACTLY ONE secp256k1 signature.
// Returns false if an error occurs while parsing the tx, or retrieving the signer public key, it returns false (to fail safely).
func txHasOneSecp256k1Signature(tx cosmostypes.Tx) bool {
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return false
	}

	pubKeys, err := sigTx.GetPubKeys()
	if err != nil {
		return false
	}

	// Ensure that the transaction has exactly one signer.
	if len(pubKeys) != 1 {
		return false
	}

	// Check if the signer's public key  is a secp256k1 public key.
	_, isSecp256k1 := pubKeys[0].(*secp256k1.PubKey)
	return isSecp256k1
}

// txHasOnlyMorseClaimMsgs returns true if the given transaction contains ONLY
// morse claim messages.
// I.e. MsgClaimMorseAccount, MsgClaimMorseApplication and MsgClaimMorseSupplier
func txHasOnlyMorseClaimMsgs(tx cosmostypes.Tx) bool {
	msgs := tx.GetMsgs()

	// At least one message must be present in the transaction.
	if len(msgs) == 0 {
		return false
	}

	// Iterate through all messages in the transaction and check if they are
	// all morse claim messages.
	for _, msg := range msgs {
		msgTypeUrl := cosmostypes.MsgTypeURL(msg)
		switch msgTypeUrl {
		case claimMorseAcctMsgTypeUrl,
			claimMorseAppMsgTypeUrl,
			claimMorseSupplierMsgTypeUrl:
			// check the remaining msgs type URLs...
			continue
		default:
			// At least one message is not a morse claim message.
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
	// Intentionally not consuming any gas; no-op.
	return nil
}

// freeTxFeeChecker is a TxFeeChecker that always returns zero gas fees
// (i.e. min gas price does not apply).
func freeTxFeeChecker(_ cosmostypes.Context, _ cosmostypes.Tx) (cosmostypes.Coins, int64, error) {
	return cosmostypes.NewCoins(cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 0)), 0, nil
}
