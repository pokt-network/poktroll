package keeper

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

func (k msgServer) ClaimMorseAccount(ctx context.Context, msg *migrationtypes.MsgClaimMorseAccount) (*migrationtypes.MsgClaimMorseAccountResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	// Ensure that gas fees are NOT waived if one of the following is true:
	// - The claim is invalid
	// - Morse account has already been claimed
	// Claiming gas fees in the cases above ensures that we prevent spamming.
	//
	// TODO_MAINNET_MIGRATION(@bryanchriswhite): Make this conditional once the WaiveMorseClaimGasFees param is available.
	//
	// Rationale:
	// 1. Morse claim txs MAY be signed by Shannon accounts which have 0upokt balances.
	//    For this reason, gas fees are waived (in the ante handler) for txs which
	//    contain ONLY (one or more) Morse claim messages.
	// 2. This exposes a potential resource exhaustion vector (or at least extends the
	//    attack surface area) where an attacker would be able to take advantage of
	//    the fact that tx signature verification gas costs MAY be avoided under
	//    certain conditions.
	// 3. ALL Morse account claim message handlers therefore SHOULD ensure that
	//    tx signature verification gas costs ARE applied if the claim is EITHER
	//    invalid OR if the given Morse account has already been claimed. The latter
	//    is necessary to mitigate a replay attack vector.
	var (
		morseClaimableAccount              migrationtypes.MorseClaimableAccount
		isFound, isValid, isAlreadyClaimed bool
	)
	defer func() {
		if !isFound || !isValid || isAlreadyClaimed {
			// Attempt to charge the waived gas fee for invalid claims.
			sdkCtx.GasMeter()
			// DEV_NOTE: Assuming that the tx containing this message was signed
			// by a non-multisig externally owned account (EOA); i.e. secp256k1,
			// conventionally. If this assumption is violated, the "wrong" gas
			// cost will be charged for the given key type.
			gas := k.accountKeeper.GetParams(ctx).SigVerifyCostSecp256k1
			sdkCtx.GasMeter().ConsumeGas(gas, "ante verify: secp256k1")
		}
	}()

	if err := msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// DEV_NOTE: It is safe to use MustAccAddressFromBech32 here because the
	// shannonDestAddress is validated in MsgClaimMorseAccount#ValidateBasic().
	shannonAccAddr := cosmostypes.MustAccAddressFromBech32(msg.ShannonDestAddress)

	// Ensure that a MorseClaimableAccount exists for the given morseSrcAddress.
	morseClaimableAccount, isFound = k.GetMorseClaimableAccount(
		sdkCtx,
		msg.GetMorseSrcAddress(),
	)
	if !isFound {
		return nil, status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"no morse claimable account exists with address %q",
				msg.GetMorseSrcAddress(),
			).Error(),
		)
	}

	// Ensure that the given MorseClaimableAccount has not already been claimed.
	if morseClaimableAccount.IsClaimed() {
		isAlreadyClaimed = true
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				msg.GetMorseSrcAddress(),
				morseClaimableAccount.ClaimedAtHeight,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)
	}

	// ONLY allow claiming as a non-actor account if the MorseClaimableAccount
	// WAS NOT staked as an application or supplier.
	// A claim of staked POKT from Morse to Shannon
	// SHOULD NOT allow applications or suppliers to bypass the onchain unbonding
	if !morseClaimableAccount.ApplicationStake.IsZero() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"Morse account %q is staked as an application, please use `pocketd tx migration claim-application` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	if !morseClaimableAccount.SupplierStake.IsZero() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"Morse account %q is staked as an supplier, please use `pocketd tx migration claim-supplier` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	// Set ShannonDestAddress & ClaimedAtHeight (claim).
	morseClaimableAccount.ShannonDestAddress = shannonAccAddr.String()
	morseClaimableAccount.ClaimedAtHeight = sdkCtx.BlockHeight()

	// Update the MorseClaimableAccount.
	k.SetMorseClaimableAccount(
		sdkCtx,
		morseClaimableAccount,
	)

	// Mint the totalTokens to the shannonDestAddress account balance.
	unstakedBalance := morseClaimableAccount.UnstakedBalance
	if err := k.MintClaimedMorseTokens(ctx, shannonAccAddr, unstakedBalance); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Emit an event which signals that the morse account has been claimed.
	sharedParams := k.sharedKeeper.GetParams(ctx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdkCtx.BlockHeight())
	event := migrationtypes.EventMorseAccountClaimed{
		SessionEndHeight:   sessionEndHeight,
		ShannonDestAddress: msg.ShannonDestAddress,
		MorseSrcAddress:    msg.GetMorseSrcAddress(),
		ClaimedBalance:     unstakedBalance,
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

	return &migrationtypes.MsgClaimMorseAccountResponse{
		MorseSrcAddress:  msg.GetMorseSrcAddress(),
		ClaimedBalance:   unstakedBalance,
		SessionEndHeight: sessionEndHeight,
	}, nil
}
