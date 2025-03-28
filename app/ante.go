package app

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/app/volatile"
)

// newAnteHandlerFn returns an AnteHandler that waives minimum gas/fees for transactions
// that contain ONLY morse claim messages (i.e. MsgClaimMorseAccount, MsgClaimMorseApplication,
// and MsgClaimMorseSupplier).
//
// TODO_MAINNET(@bryanchriswhite):
// - Add a migration module param, `WaiveMorseClaimFees`.
// - Return the default antehandler if it is false.
func newAnteHandlerFn(app *App) types.AnteHandler {
	return func(ctx types.Context, tx types.Tx, simulate bool) (types.Context, error) {
		anteHandlerFn, err := ante.NewAnteHandler(ante.HandlerOptions{
			AccountKeeper:          &app.Keepers.AccountKeeper,
			BankKeeper:             app.Keepers.BankKeeper,
			ExtensionOptionChecker: nil,
			FeegrantKeeper:         app.Keepers.FeeGrantKeeper,
			SignModeHandler:        app.txConfig.SignModeHandler(),
			SigGasConsumer:         newSigVerificationGasConsumer(tx),
			TxFeeChecker:           newTxFeeChecker(tx),
		})
		if err != nil {
			return ctx, err
		}

		return anteHandlerFn(ctx, tx, simulate)
	}
}

// newSigVerificationGasConsumer returns a SignatureVerificationGasConsumer that
// returns zero gas fees for transactions that contain ONLY morse claim messages
// (i.e. MsgClaimMorseAccount, MsgClaimMorseApplication, and MsgClaimMorseSupplier).
func newSigVerificationGasConsumer(tx types.Tx) ante.SignatureVerificationGasConsumer {
	// If the tx consists of ONLY Morse claim message(s), use the freeSigGasConsumer.
	if txHasOnlyMorseClaimMsgs(tx) {
		return freeSigGasConsumer
	}

	// Use the default signature verification gas consumer if ANY
	// non-morse-claim message type is included in the tx.
	return ante.DefaultSigVerificationGasConsumer
}

// newTxFeeChecker returns a TxFeeChecker that returns zero gas fees for
// transactions that contain ONLY morse claim messages (i.e. MsgClaimMorseAccount,
// MsgClaimMorseApplication, and MsgClaimMorseSupplier).
func newTxFeeChecker(tx types.Tx) ante.TxFeeChecker {
	if txHasOnlyMorseClaimMsgs(tx) {
		return freeTxFeeChecker
	}

	// When nil is returned, ante.checkTxFeeWithValidatorMinGasPrices is used by
	// default by cosmos-sdk.
	// See:
	// - https://github.com/cosmos/cosmos-sdk/blob/v0.50.13/x/auth/ante/fee.go#L31
	// - https://github.com/cosmos/cosmos-sdk/blob/v0.50.13/x/auth/ante/ante.go#L48
	return nil
}

// txHasOnlyMorseClaimMsgs returns true if the given transaction contains ONLY
// morse claim messages (i.e. MsgClaimMorseAccount, MsgClaimMorseApplication, and
// MsgClaimMorseSupplier).
func txHasOnlyMorseClaimMsgs(tx types.Tx) bool {
	msgs := tx.GetMsgs()
	if len(msgs) == 0 {
		return false
	}

	for _, msg := range msgs {
		msgTypeUrl := types.MsgTypeURL(msg)
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
func freeTxFeeChecker(_ types.Context, _ types.Tx) (types.Coins, int64, error) {
	return types.NewCoins(types.NewInt64Coin(volatile.DenomuPOKT, 0)), 0, nil
}
