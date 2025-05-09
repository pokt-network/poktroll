package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

// ImportMorseClaimableAccounts persists all MorseClaimableAccounts in the given
// MorseAccountState to the KVStore.
// This operation MAY ONLY be performed EXACTLY ONCE (per network/re-genesis),
// and ONLY by an authorized account (i.e. PNF).
func (k msgServer) ImportMorseClaimableAccounts(ctx context.Context, msg *migrationtypes.MsgImportMorseClaimableAccounts) (*migrationtypes.MsgImportMorseClaimableAccountsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("method", "ImportMorseClaimableAccounts")

	// Validate the authority.
	if msg.GetAuthority() != k.GetAuthority() {
		err := migrationtypes.ErrInvalidSigner.Wrapf("invalid authority address (%s)", msg.GetAuthority())
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	// Validate the import message.
	if err := msg.ValidateBasic(); err != nil {
		logger.Info(err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if MorseClaimableAccounts have already been imported.
	if k.HasAnyMorseClaimableAccounts(sdkCtx) {
		// Check if allow_morse_accounts_import_overwrite is enabled and return an error if not.
		shouldOverwrite := k.GetParams(sdkCtx).AllowMorseAccountImportOverwrite
		if !shouldOverwrite {
			err := migrationtypes.ErrMorseAccountsImport.Wrap("Morse claimable accounts already imported and import overwrite is disabled")
			logger.Info(err.Error())
			return nil, status.Error(codes.FailedPrecondition, err.Error())
		}
	}

	// Message handlers run during both CheckTx and DeliverTx.
	// To reduce noise and confusion, only log during DeliverTx.
	if !sdkCtx.IsCheckTx() {
		logger.Info("beginning importing morse claimable accounts...")
	}

	// Import MorseClaimableAccounts.
	k.ImportFromMorseAccountState(sdkCtx, &msg.MorseAccountState)

	// Message handlers run during both CheckTx and DeliverTx.
	// To reduce noise and confusion, only log during DeliverTx.
	if !sdkCtx.IsCheckTx() {
		logger.Info("done importing morse claimable accounts!")
	}

	// Emit the corresponding event.
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

	// Return the response.
	return &migrationtypes.MsgImportMorseClaimableAccountsResponse{
		// DEV_NOTE: The MorseAccountStateHash is validated in msg#ValidateBasic().
		StateHash:   msg.MorseAccountStateHash,
		NumAccounts: uint64(len(msg.MorseAccountState.Accounts)),
	}, nil
}
