package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
)

func (k msgServer) ImportMorseClaimableAccounts(ctx context.Context, msg *migrationtypes.MsgImportMorseClaimableAccounts) (*migrationtypes.MsgImportMorseClaimableAccountsResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	logger := sdkCtx.Logger().With("method", "CreateMorseAccountState")

	if msg.GetAuthority() != k.GetAuthority() {
		err := migrationtypes.ErrUnauthorized.Wrapf("invalid authority address (%s)", msg.GetAuthority())
		return nil, status.Error(codes.PermissionDenied, err.Error())
	}

	// Validate the import message.
	if err := msg.ValidateBasic(); err != nil {
		logger.Info(err.Error())
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// Check if MorseClaimableAccounts have already been imported.
	if morseClaimableAccounts := k.GetAllMorseClaimableAccounts(sdkCtx); len(morseClaimableAccounts) > 0 {
		err := migrationtypes.ErrMorseAccountsImport.Wrap("Morse claimable accounts already imported")
		logger.Info(err.Error())
		return nil, status.Error(codes.FailedPrecondition, err.Error())
	}

	// Import MorseClaimableAccounts.
	k.ImportFromMorseAccountState(sdkCtx, &msg.MorseAccountState)

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
