package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/pkg/encoding"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// AdminRecoverMorseAccount allows the authority to recover Morse accounts WITHOUT
// checking the allowlist. This enables faster recovery for legitimate requests that
// have been validated off-chain.
//
// SECURITY: Can ONLY be called by the module authority (PNF via authz).
// SAFETY: Still checks that account exists and hasn't been claimed already.
// SKIP: Does NOT check IsMorseAddressRecoverable (allowlist check).
func (k msgServer) AdminRecoverMorseAccount(ctx context.Context, msg *migrationtypes.MsgAdminRecoverMorseAccount) (*migrationtypes.MsgAdminRecoverMorseAccountResponse, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// 1. Validate basic message fields (address formats).
	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// 2. CRITICAL: Check if the authority is valid.
	// Only the module authority (gov module or authorized via authz) can call this.
	if k.GetAuthority() != msg.Authority {
		return nil, status.Error(
			codes.PermissionDenied,
			migrationtypes.ErrMorseRecoverableAccountClaim.Wrapf(
				"invalid authority; expected %s, got %s",
				k.GetAuthority(), msg.GetAuthority(),
			).Error(),
		)
	}

	normalizedMorseSrcAddress := encoding.NormalizeMorseAddress(msg.GetMorseSrcAddress())

	// 3. NOTE: SKIPPING ALLOWLIST CHECK
	// This is the key difference from RecoverMorseAccount!
	// We trust the authority (PNF) to validate recovery requests off-chain.
	// The regular RecoverMorseAccount would call:
	//   if !recovery.IsMorseAddressRecoverable(normalizedMorseSrcAddress) { ... }

	// 4. SAFETY CHECK: Verify the account exists on-chain.
	// The account must have been imported via MsgImportMorseClaimableAccounts.
	morseClaimableAccount, isFound := k.GetMorseClaimableAccount(
		sdkCtx,
		normalizedMorseSrcAddress,
	)
	if !isFound {
		return nil, status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseRecoverableAccountClaim.Wrapf(
				"no morse account exists with address %q (must be imported first via MsgImportMorseClaimableAccounts)",
				normalizedMorseSrcAddress,
			).Error(),
		)
	}

	// 5. SAFETY CHECK: Ensure the account has not already been claimed/recovered.
	// Each account can only be recovered once.
	if morseClaimableAccount.IsClaimed() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseRecoverableAccountClaim.Wrapf(
				"morse address %q has already been recovered at height %d onto shannon address %q",
				normalizedMorseSrcAddress,
				morseClaimableAccount.ClaimedAtHeight,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)
	}

	// 6. Perform the recovery - same as regular RecoverMorseAccount.
	recoveredBalance := morseClaimableAccount.TotalTokens()
	currentHeight := sdkCtx.BlockHeight()

	// Mark the account as claimed.
	morseClaimableAccount.ShannonDestAddress = msg.GetShannonDestAddress()
	morseClaimableAccount.ClaimedAtHeight = currentHeight

	// Update the on-chain state.
	k.SetMorseClaimableAccount(
		sdkCtx,
		morseClaimableAccount,
	)

	// Mint the recovered balance to the destination Shannon account.
	shannonAccAddr := sdk.MustAccAddressFromBech32(msg.GetShannonDestAddress())
	if err := k.MintClaimedMorseTokens(ctx, shannonAccAddr, recoveredBalance); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// 7. Emit event with admin flag for audit trail.
	// This distinct event allows monitoring/filtering of admin recoveries.
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentHeight)
	event := migrationtypes.EventMorseAccountRecoveredAdmin{
		SessionEndHeight:   sessionEndHeight,
		RecoveredBalance:   recoveredBalance.String(),
		ShannonDestAddress: msg.GetShannonDestAddress(),
		MorseSrcAddress:    normalizedMorseSrcAddress,
		IsAdminRecovery:    true, // Always true for admin recovery
	}
	if err := sdkCtx.EventManager().EmitTypedEvent(&event); err != nil {
		return nil, status.Error(
			codes.Internal,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"failed to emit event type %T: %v",
				&event,
				err,
			).Error(),
		)
	}

	return &migrationtypes.MsgAdminRecoverMorseAccountResponse{}, nil
}
