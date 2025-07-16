package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// ImportMorseClaimableAccounts
// - Persists all MorseClaimableAccounts in the provided MorseAccountState to the KVStore.
// - This operation MUST be performed EXACTLY ONCE per network/re-genesis.
// - ONLY an authorized account (e.g., PNF) may execute this operation.
// - Overwriting is only allowed if explicitly enabled in onchain governance params.
func (k msgServer) ImportMorseClaimableAccounts(
	ctx context.Context,
	msg *migrationtypes.MsgImportMorseClaimableAccounts,
) (*migrationtypes.MsgImportMorseClaimableAccountsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("method", "ImportMorseClaimableAccounts")

	// Validate authority
	// - Ensure the message is signed by the correct authority (e.g., PNF).
	if msg.GetAuthority() != k.GetAuthority() {
		err := migrationtypes.ErrInvalidSigner.Wrapf("invalid authority address (%s)", msg.GetAuthority())
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	// Validate import message
	// - Run basic validation for the import message.
	if err := msg.ValidateBasic(); err != nil {
		logger.Info(err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if MorseClaimableAccounts have already been imported
	if k.HasAnyMorseClaimableAccounts(sdkCtx) {
		// If already imported:
		// - Check if allow_morse_accounts_import_overwrite is enabled
		// - If not enabled, return an error
		shouldOverwrite := k.GetParams(sdkCtx).AllowMorseAccountImportOverwrite
		if !shouldOverwrite {
			err := migrationtypes.ErrMorseAccountsImport.Wrap("Morse claimable accounts already imported and import overwrite is disabled")
			logger.Info(err.Error())
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
		// Overwrites are enabled. Overwrite logic:
		// - Delete all existing MorseClaimableAccounts (and indices)
		// - Continue to re-import from msg
		k.resetMorseClaimableAccounts(sdkCtx)
	}
	// DEV_NOTE: This code path is reached only if ONE OF THE FOLLOWING is true:
	// - This is the first time MorseClaimableAccounts are being imported
	// - This is not the first time MorseClaimableAccounts are being imported, but overwriting is enabled

	// Only log during DeliverTx (not CheckTx) to reduce noise/confusion
	if !sdkCtx.IsCheckTx() {
		logger.Info("beginning importing morse claimable accounts...")
	}

	// Import MorseClaimableAccounts from the provided MorseAccountState
	k.ImportFromMorseAccountState(sdkCtx, &msg.MorseAccountState)

	// Only log during DeliverTx (not CheckTx) to reduce noise/confusion
	if !sdkCtx.IsCheckTx() {
		logger.Info("done importing morse claimable accounts!")
	}

	// Emit event for Morse claimable accounts import
	// - Includes: block height, MorseAccountStateHash, and number of accounts
	if err := sdkCtx.EventManager().EmitTypedEvent(
		&migrationtypes.EventImportMorseClaimableAccounts{
			CreatedAtHeight: sdkCtx.BlockHeight(),
			// DEV_NOTE: The MorseAccountStateHash is validated in msg#ValidateBasic().
			MorseAccountStateHash: msg.MorseAccountStateHash,
			NumAccounts:           uint64(len(msg.MorseAccountState.Accounts)),
		},
	); err != nil {
		logger.Error(err.Error())
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Return response
	// - Includes: MorseAccountStateHash and number of imported accounts
	return &migrationtypes.MsgImportMorseClaimableAccountsResponse{}, nil
}
