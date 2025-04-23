package app

import (
	storetypes "cosmossdk.io/store/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	authsigning "github.com/cosmos/cosmos-sdk/x/auth/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"

	"github.com/pokt-network/poktroll/app/volatile"
)

// newMorseClaimGasFeesWaiverAnteHandlerFn returns an AnteHandler that
// 1. lazily creates empty BaseAccounts
// 2. waives minimum gas/fees for transactions that contain ONLY morse claim messages (i.e. MsgClaimMorseAccount, MsgClaimMorseApplication, and MsgClaimMorseSupplier)
func newMorseClaimGasFeesWaiverAnteHandlerFn(app *App) cosmostypes.AnteHandler {
	return func(sdkCtx cosmostypes.Context, tx cosmostypes.Tx, simulate bool) (cosmostypes.Context, error) {
		pocketAnte, err := newPocketAnteHandler(pocketAnteHandlerOptions{
			// Default ante handler options
			HandlerOptions: ante.HandlerOptions{
				AccountKeeper:          &app.Keepers.AccountKeeper,
				BankKeeper:             app.Keepers.BankKeeper,
				ExtensionOptionChecker: nil,
				FeegrantKeeper:         app.Keepers.FeeGrantKeeper,
				SignModeHandler:        app.txConfig.SignModeHandler(),
			},
			// Pocket-specific ante handler options
			AuthAccountKeeper: app.Keepers.AccountKeeper,
			SigGasConsumer:    newSigVerificationGasConsumer(sdkCtx, app, tx),
			TxFeeChecker:      newTxFeeChecker(sdkCtx, app, tx),
		})
		if err != nil {
			return sdkCtx, err
		}

		return pocketAnte(sdkCtx, tx, simulate)
	}
}

// NewPocketAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
func newPocketAnteHandler(opts pocketAnteHandlerOptions) (cosmostypes.AnteHandler, error) {
	// basic sanity checks copied from SDK builder
	if opts.AccountKeeper == nil {
		return nil, sdkerrors.ErrLogic.Wrap("account keeper is required for ante builder")
	}
	if opts.AuthAccountKeeper == nil {
		return nil, sdkerrors.ErrLogic.Wrap("auth account keeper is required for ante builder")
	}
	if opts.BankKeeper == nil {
		return nil, sdkerrors.ErrLogic.Wrap("bank keeper is required for ante builder")
	}
	if opts.SignModeHandler == nil {
		return nil, sdkerrors.ErrLogic.Wrap("sign mode handler is required for ante builder")
	}

	anteDecorators := []cosmostypes.AnteDecorator{
		ante.NewSetUpContextDecorator(),
		ante.NewExtensionOptionsDecorator(opts.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(opts.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(opts.AccountKeeper),
		ante.NewDeductFeeDecorator(opts.AccountKeeper, opts.BankKeeper, opts.FeegrantKeeper, opts.TxFeeChecker),
		// Note that we are using `AuthAccountKeeper` here instead of `AccountKeeper` to create BaseAccounts
		autoCreateAccountDecorator{ak: opts.AuthAccountKeeper}, // autoCreateAccountDecorator must run before SetPubKeyDecorator
		ante.NewSetPubKeyDecorator(opts.AccountKeeper),         // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(opts.AccountKeeper),
		ante.NewSigGasConsumeDecorator(opts.AccountKeeper, opts.SigGasConsumer),
		ante.NewSigVerificationDecorator(opts.AccountKeeper, opts.SignModeHandler),
		ante.NewIncrementSequenceDecorator(opts.AccountKeeper),
	}

	return cosmostypes.ChainAnteDecorators(anteDecorators...), nil
}

// autoCreateAccountDecorator creates an empty BaseAccount for every signer that is unknown to x/auth.
// It **must** run *before* SetPubKeyDecorator.
type autoCreateAccountDecorator struct{ ak authkeeper.AccountKeeper }

// Pocket‑specific AnteHandler builder (fork of SDK default)
// Embeds the cosmos sdk default options with the required options
type pocketAnteHandlerOptions struct {
	ante.HandlerOptions // embed all default options
	// Explicitly passing in AuthAccountKeeper because the expected account keeper in HandlerOptions
	// does not contain `NewAccountWithAddress` needed above
	AuthAccountKeeper authkeeper.AccountKeeperI             // *required*
	SigGasConsumer    ante.SignatureVerificationGasConsumer // *required*
	TxFeeChecker      ante.TxFeeChecker                     // *required*
}

// AnteHandler that creates an empty BaseAccount for every signer that is unknown to x/auth.
// It **must** run *before* SetPubKeyDecorator.
func (d autoCreateAccountDecorator) AnteHandle(
	ctx cosmostypes.Context, tx cosmostypes.Tx, _ bool, next cosmostypes.AnteHandler,
) (cosmostypes.Context, error) {
	sigTx, ok := tx.(authsigning.SigVerifiableTx)
	if !ok {
		return ctx, sdkerrors.ErrTxDecode.Wrapf("invalid tx type %T; expected authsigning.SigVerifiableTx", tx)
	}

	signers, err := sigTx.GetSigners()
	if err != nil {
		return ctx, sdkerrors.ErrTxDecode.Wrapf("failed to get signers: %s", err)
	}

	for _, addr := range signers {
		if d.ak.GetAccount(ctx, addr) == nil {
			acc := d.ak.NewAccountWithAddress(ctx, addr)
			d.ak.SetAccount(ctx, acc) // seq=0, account number auto‑assigned
		}
	}

	return next(ctx, tx, false)
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
