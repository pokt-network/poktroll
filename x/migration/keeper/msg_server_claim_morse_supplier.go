package keeper

import (
	"context"

	"cosmossdk.io/errors"
	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/pocket"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// ClaimMorseSupplier processes a Morse supplier claim migration.
//
// Preconditions:
// - The message is valid.
// - A MorseClaimableAccount exists for the given morse_src_address.
//
// Steps performed:
// - Mint and transfer all tokens (unstaked balance plus supplier stake) from the MorseClaimableAccount to the shannonDestAddress.
// - Mark the MorseClaimableAccount as claimed (i.e., set the shannon_dest_address and claimed_at_height).
// - Stake a supplier for the amount and services specified in the MorseClaimableAccount and the message.
//
// Short Circuits (these cause early exit):
// - Short circuit #1: If the Morse Supplier started unstaking before the state shift and fully unstaked after the state shift at the time of claim, mint the staked balance to the Shannon owner and exit after event emission.
// - Short circuit #2: If the Morse Supplier's stake is below the minimum required, auto-unstake, mint to owner, emit events, and exit.
//
// The function does not alter business logic and preserves all original comment content, but comments have been clarified and reformatted for improved readability.
func (k msgServer) ClaimMorseSupplier(
	ctx context.Context,
	msg *migrationtypes.MsgClaimMorseSupplier,
) (*migrationtypes.MsgClaimMorseSupplierResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger := k.Logger().With("method", "ClaimMorseSupplier")

	var (
		morseNodeClaimableAccount *migrationtypes.MorseClaimableAccount
		isFound, isAlreadyClaimed bool
		err                       error
	)
	defer k.deferAdjustWaivedGasFees(ctx, &isFound, &isAlreadyClaimed)()

	// Ensure that morse account claiming is enabled.
	morseAccountClaimingIsEnabled := k.GetParams(sdkCtx).MorseAccountClaimingEnabled
	if !morseAccountClaimingIsEnabled {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseAccountClaim.Wrapf(
				"morse account claiming is currently disabled; please contact the Pocket Network team",
			).Error(),
		)
	}

	if err = msg.ValidateBasic(); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	// DEV_NOTE: It is safe to use MustAccAddressFromBech32 here because the
	// shannonOwnerAddress and shannonOperatorAddress are validated in MsgClaimMorseSupplier#ValidateBasic().
	shannonOwnerAddr := cosmostypes.MustAccAddressFromBech32(msg.ShannonOwnerAddress)

	// Default to the shannonOwnerAddr as the shannonOperatorAddr if not provided.
	// The shannonOperatorAddr is where the Morse node/supplier unstaked balance will be minted to.
	shannonOperatorAddr := shannonOwnerAddr
	if msg.ShannonOperatorAddress != "" {
		shannonOperatorAddr = cosmostypes.MustAccAddressFromBech32(msg.ShannonOperatorAddress)
	}

	// Retrieve the MorseClaimableAccount for the given morseSrcAddress.
	morseNodeClaimableAccount, err = k.checkMorseClaimableSupplierAccount(ctx, msg.GetMorseNodeAddress())
	if err != nil {
		return nil, err
	}

	// Ensure the signer is ONE OF THE FOLLOWING:
	// - The Morse node address (i.e. operator)
	// - The Morse output address (i.e. owner)
	claimSignerType, err := checkClaimSigner(msg, morseNodeClaimableAccount)
	if err != nil {
		return nil, status.Error(
			codes.InvalidArgument,
			err.Error(),
		)
	}

	// Default shannonSigningAddress to shannonOperatorAddr because the Shannon owner defaults to the operator.
	// The shannonSigningAddress is where the node/supplier stake will be minted to and then escrowed from.
	shannonSigningAddress := shannonOperatorAddr
	switch claimSignerType {

	// ## NON-CUSTODIAL OWNER CLAIM ##
	// The Morse owner/output account is signing the claim.
	case migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_NON_CUSTODIAL_SIGNED_BY_OWNER:
		shannonSigningAddress = shannonOwnerAddr

	// ## NON-CUSTODIAL OPERATOR CLAIM ##
	// The Morse node/operator account is signing the claim.
	// This requires (i.e. pre-requisite) that:
	// 1. The Morse owner/output account has already been claimed.
	// 2. The claimed onchain Morse owner Shannon address matches the supplier claim Shannon owner address.
	case migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_NON_CUSTODIAL_SIGNED_BY_NODE_ADDR:
		// Retrieve the Morse owner claimable account for the Morse owner address.
		morseOwnerAddress := morseNodeClaimableAccount.GetMorseOutputAddress()

		// Retrieve the Morse owner claimable account.
		morseOwnerClaimableAccount, isMorseOwnerFound := k.GetMorseClaimableAccount(ctx, morseOwnerAddress)
		if !isMorseOwnerFound {
			// DEV_NOTE: THIS SHOULD NEVER HAPPEN.
			// If this occurs, it indicates that either:
			// 1. The Morse owner account was not included in the list of imported Morse claimable accounts.
			// 2. The Morse owner account was somehow removed from the list of imported Morse claimable accounts.
			return nil, status.Error(
				codes.Internal,
				migrationtypes.ErrMorseSupplierClaim.Wrapf(
					"(SHOULD NEVER HAPPEN) could not find morse claimable account for owner address (%s)",
					morseOwnerAddress,
				).Error(),
			)
		}

		// Ensure that the Morse owner account has already been claimed before migrating the Morse node/supplier to a Shannon account.
		if !morseOwnerClaimableAccount.IsClaimed() {
			return nil, status.Error(
				codes.FailedPrecondition,
				migrationtypes.ErrMorseSupplierClaim.Wrapf(
					"morse owner address (%s) MUST be claimed BEFORE migrating the Morse node/supplier to a Shannon Supplier account",
					morseOwnerAddress,
				).Error(),
			)
		}

		// Ensure that the Shannon owner address on the Morse supplier claim
		// matches the Shannon dest address of the claimed Morse owner account.
		if morseOwnerClaimableAccount.GetShannonDestAddress() != msg.GetShannonOwnerAddress() {
			return nil, status.Error(
				codes.FailedPrecondition,
				migrationtypes.ErrMorseSupplierClaim.Wrapf(
					"the Shannon owner address on the Morse supplier (%s) claim MUST match the Shannon dest address of the already claimed Morse owner account (%s)",
					msg.GetShannonOwnerAddress(),
					morseOwnerClaimableAccount.GetShannonDestAddress(),
				).Error(),
			)
		}
	}

	// Supplier Claim - Short circuit #1
	// If both of the following are true:
	// 1. The Morse Supplier started unstaking before the state shift
	// 2. The Morse Supplier fully unstaked after the state shift at the time of claim
	// Then the Shannon owner address is where the staked balance needs to be minted to.
	morseUnbondingPeriodElapsed := morseNodeClaimableAccount.HasUnbonded(ctx)

	// Supplier Claim - Short circuit #2
	// If the Morse Supplier's stake is less than the minimum stake, then the Morse Supplier should be auto-unstaked.
	minStake := k.supplierKeeper.GetParams(ctx).MinStake
	claimableSupplierStake := morseNodeClaimableAccount.GetSupplierStake()
	shouldAutoUnstake := claimableSupplierStake.Amount.LT(minStake.Amount)

	// Determine the staked tokens destination address based on the short circuit conditions above.
	var stakedTokensDestAddr sdk.AccAddress
	if morseUnbondingPeriodElapsed || shouldAutoUnstake {
		stakedTokensDestAddr = shannonOwnerAddr
	} else {
		stakedTokensDestAddr = shannonSigningAddress
	}

	// Mint the Morse node/supplier's stake to the stakedTokensDestAddr account balance.
	// The Supplier stake is subsequently escrowed from the stakedTokensDestAddr account balance
	// UNLESS it has already unbonded during the migration.
	// NOTE: The supplier module's staking fee parameter will be deducted from the claimed balance below.
	if err = k.MintClaimedMorseTokens(ctx, stakedTokensDestAddr, morseNodeClaimableAccount.GetSupplierStake()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Mint the Morse node/supplier's unstaked balance to the shannonOperatorAddress account balance.
	// The operator will always received the unstaked balance of the Morse Supplier.
	if err = k.MintClaimedMorseTokens(ctx, shannonOperatorAddr, morseNodeClaimableAccount.GetUnstakedBalance()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Set ShannonDestAddress & ClaimedAtHeight (claim).
	morseNodeClaimableAccount.ShannonDestAddress = stakedTokensDestAddr.String()
	morseNodeClaimableAccount.ClaimedAtHeight = sdkCtx.BlockHeight()

	// Update the MorseClaimableAccount.
	k.SetMorseClaimableAccount(sdkCtx, *morseNodeClaimableAccount)

	// Retrieve the shared module parameters and calculate the session end height.
	sharedParams := k.sharedKeeper.GetParams(sdkCtx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdkCtx.BlockHeight())
	currentSessionStartHeight := sharedtypes.GetSessionStartHeight(&sharedParams, sdkCtx.BlockHeight())
	previousSessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, currentSessionStartHeight-1)

	// Retrieve the claimed supplier stake and unstaked balance.
	claimedSupplierStake := morseNodeClaimableAccount.GetSupplierStake()
	claimedUnstakedBalance := morseNodeClaimableAccount.GetUnstakedBalance()

	// Collect all events for emission.
	// Always emitted:
	// - EventMorseSupplierClaimed
	// Conditionally emitted:
	// - EventSupplierUnbondingBegin
	// - EventSupplierUnbondingEnd
	// Events are appended prior to emission to allow for conditional modification prior to emission.
	events := make([]cosmostypes.Msg, 0)

	// Construct unbonded supplier for cases where it is already or will become unbonded
	// immediately (i.e. below min stake, or if unbonding period has already elapsed).
	unbondedSupplier := &sharedtypes.Supplier{
		OwnerAddress:            shannonOwnerAddr.String(),
		OperatorAddress:         shannonOperatorAddr.String(),
		Stake:                   &claimedSupplierStake,
		UnstakeSessionEndHeight: uint64(previousSessionEndHeight),
		// Services:             (intentionally omitted, no services were staked),
		// ServiceConfigHistory: (intentionally omitted, no services were staked),
	}

	// Construct the base response. It will be modified, as necessary, prior to returning.
	claimMorseSupplierResponse := &migrationtypes.MsgClaimMorseSupplierResponse{
		MorseNodeAddress:     msg.GetMorseNodeAddress(),
		MorseOutputAddress:   morseNodeClaimableAccount.GetMorseOutputAddress(),
		ClaimSignerType:      claimSignerType,
		ClaimedBalance:       claimedUnstakedBalance,
		ClaimedSupplierStake: claimedSupplierStake,
		SessionEndHeight:     sessionEndHeight,
		Supplier:             unbondedSupplier,
	}

	// Construct the base supplier claim event. It will be modified, as necessary, prior to emission.
	// ALWAYS emit an event which signals that the morse supplier has been claimed.
	morseSupplierClaimedEvent := &migrationtypes.EventMorseSupplierClaimed{
		MorseNodeAddress:     msg.GetMorseNodeAddress(),
		MorseOutputAddress:   morseNodeClaimableAccount.GetMorseOutputAddress(),
		ClaimSignerType:      claimSignerType,
		ClaimedBalance:       claimedUnstakedBalance,
		ClaimedSupplierStake: claimedSupplierStake,
		SessionEndHeight:     sessionEndHeight,
		Supplier:             unbondedSupplier,
	}

	// Conditionally emit an event which signals that the claimed Morse supplier's unbonding
	// period began on Morse, and ended while waiting to be claimed.
	morseSupplierUnbondingEndEvent := &suppliertypes.EventSupplierUnbondingEnd{
		Supplier:           unbondedSupplier,
		Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_MIGRATION,
		SessionEndHeight:   sessionEndHeight,
		UnbondingEndHeight: previousSessionEndHeight,
	}

	// Short circuit #1
	// If unbonding is complete:
	// - No further minting is needed
	// - Block time is estimated and used to set the unstake session end height
	// - Emit event to signal unbonding start
	if morseUnbondingPeriodElapsed {
		events = append(events, morseSupplierClaimedEvent)
		events = append(events, morseSupplierUnbondingEndEvent)
		if err = emitEvents(ctx, events); err != nil {
			return nil, err
		}

		return claimMorseSupplierResponse, nil
	}

	// Short circuit #2
	// If the claimed supplier stake is less than the minimum stake, the supplier is immediately unstaked.
	// - All staked tokens have already been minted to stakedTokensDestAddr account
	// - All unstaked tokens have already been minted to shannonOperatorAddr account
	if shouldAutoUnstake {
		events = append(events, morseSupplierClaimedEvent)
		events = append(events, morseSupplierUnbondingEndEvent)
		if err = emitEvents(ctx, events); err != nil {
			return nil, err
		}

		return claimMorseSupplierResponse, nil
	}

	// Aggregate (i.e. upstake) the Shannon supplier stake if we are consolidating stakes on an existing Shannon supplier.
	// Query for any existing supplier stake prior to staking.
	preClaimSupplierStake := cosmostypes.NewCoin(pocket.DenomuPOKT, math.ZeroInt())
	foundSupplier, isFound := k.supplierKeeper.GetSupplier(ctx, shannonOperatorAddr.String())
	if isFound {
		preClaimSupplierStake = *foundSupplier.Stake
	}
	postClaimSupplierStake := preClaimSupplierStake.Add(morseNodeClaimableAccount.GetSupplierStake())

	// Sanity check the service configs.
	// Quick workaround upon encountering this issue: https://gist.github.com/okdas/3328c0c507b5dba8b31ab871589f34b0
	if err = sharedtypes.ValidateSupplierServiceConfigs(msg.Services); err != nil {
		return nil, err
	}

	// Stake (or update) the supplier.
	msgStakeSupplier := suppliertypes.NewMsgStakeSupplier(
		shannonSigningAddress.String(),
		shannonOwnerAddr.String(),
		shannonOperatorAddr.String(),
		&postClaimSupplierStake,
		msg.Services,
	)

	// Stake the supplier
	supplier, err := k.supplierKeeper.StakeSupplier(ctx, logger, msgStakeSupplier)
	if err != nil {
		return nil, err
	}

	// Update the supplier claim response.
	claimMorseSupplierResponse.ClaimedBalance = morseNodeClaimableAccount.GetUnstakedBalance()
	claimMorseSupplierResponse.ClaimedSupplierStake = morseNodeClaimableAccount.GetSupplierStake()
	claimMorseSupplierResponse.Supplier = supplier

	// Update the supplier claim event.
	morseSupplierClaimedEvent.ClaimedBalance = morseNodeClaimableAccount.GetUnstakedBalance()
	morseSupplierClaimedEvent.ClaimedSupplierStake = morseNodeClaimableAccount.GetSupplierStake()
	morseSupplierClaimedEvent.Supplier = supplier

	// Emit the supplier claim event first, an unbonding begin event MAY follow.
	events = append(events, morseSupplierClaimedEvent)

	// If the claimed supplier is still unbonding:
	// - Set the unstake session end height on the supplier
	// - Emit an unbonding begin event
	if morseNodeClaimableAccount.IsUnbonding() {
		estimatedUnstakeSessionEndHeight, isUnbonded := morseNodeClaimableAccount.GetEstimatedUnbondingEndHeight(ctx, sharedParams)

		// DEV_NOTE: SHOULD NEVER happen, the check above (using #SecondsUntilUnbonded()) is the same, but in terms of time instead of block height.
		if isUnbonded {
			return nil, status.Error(
				codes.Internal,
				migrationtypes.ErrMorseSupplierClaim.Wrapf(
					"(SHOULD NEVER HAPPEN) estimated unbonding height is negative (%d)",
					estimatedUnstakeSessionEndHeight,
				).Error(),
			)
		}

		// Set the supplier's unstake session end height.
		supplier.UnstakeSessionEndHeight = uint64(estimatedUnstakeSessionEndHeight)
		k.supplierKeeper.SetAndIndexDehydratedSupplier(ctx, *supplier)

		// Emit an event which signals that the claimed Morse supplier's unbonding
		// period began on Morse and will end on Shannon at unbonding_end_height
		// (i.e. estimatedUnstakeSessionEndHeight).
		morseSupplierUnbondingBeginEvent := &suppliertypes.EventSupplierUnbondingBegin{
			Supplier:           supplier,
			Reason:             suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_MIGRATION,
			SessionEndHeight:   sessionEndHeight,
			UnbondingEndHeight: estimatedUnstakeSessionEndHeight,
		}

		// Emit the supplier unbonding begin event
		// AFTER the supplier claim event.
		events = append(events, morseSupplierUnbondingBeginEvent)
	}

	if err = emitEvents(ctx, events); err != nil {
		return nil, err
	}

	// Return the response.
	return claimMorseSupplierResponse, nil
}

// checkMorseClaimableSupplierAccount attempts to retrieve a MorseClaimableAccount for the given morseSrcAddress.
// It ensures the MorseClaimableAccount meets the following criteria:
// - It exists on-chain
// - It not already been claimed
// - It has a non-zero supplier stake
// - It has zero application stake
// If the MorseClaimableAccount does not exist, it returns an error.
// If the MorseClaimableAccount has already been claimed, any waived gas fees are charged and an error is returned.
func (k msgServer) checkMorseClaimableSupplierAccount(
	ctx context.Context,
	morseSrcAddress string,
) (*migrationtypes.MorseClaimableAccount, error) {
	// Ensure that a MorseClaimableAccount exists and has not been claimed for the given morseSrcAddress.
	morseClaimableAccount, err := k.CheckMorseClaimableAccount(ctx, morseSrcAddress, migrationtypes.ErrMorseSupplierClaim)
	if err != nil {
		return nil, err
	}

	// ONLY allow claiming as a supplier account if the MorseClaimableAccount
	// WAS staked as a supplier AND NOT as an application. A claim of staked POKT
	// from Morse to Shannon SHOULD NOT allow applications or suppliers to bypass
	// the onchain unbonding period.
	if !morseClaimableAccount.ApplicationStake.IsZero() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"Morse account %q is staked as an application, please use `pocketd tx migration claim-application` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	if !morseClaimableAccount.SupplierStake.IsPositive() {
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"Morse account %q is not staked as a supplier or application, please use `pocketd tx migration claim-account` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	return morseClaimableAccount, nil
}

// CheckMorseClaimableAccount attempts to retrieve a MorseClaimableAccount for the given morseSrcAddress.
// It ensures the MorseClaimableAccount meets the following criteria:
// - It exists on-chain
// - It not already been claimed
// - It has a non-zero supplier stake
// - It has zero application stake
// If the MorseClaimableAccount does not exist, it returns an error.
// If the MorseClaimableAccount has already been claimed, any waived gas fees are charged and an error is returned.
func (k msgServer) CheckMorseClaimableAccount(
	ctx context.Context,
	morseSrcAddress string,
	claimError *errors.Error,
) (*migrationtypes.MorseClaimableAccount, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)

	// Ensure that a MorseClaimableAccount exists for the given morseSrcAddress.
	morseClaimableAccount, isFound := k.GetMorseClaimableAccount(
		sdkCtx,
		morseSrcAddress,
	)
	if !isFound {
		return nil, status.Error(
			codes.NotFound,
			claimError.Wrapf(
				"no morse claimable account exists with address %q",
				morseSrcAddress,
			).Error(),
		)
	}

	// Ensure that the given MorseClaimableAccount has not already been claimed.
	if morseClaimableAccount.IsClaimed() {
		return nil, status.Error(
			codes.FailedPrecondition,
			claimError.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				morseClaimableAccount.GetMorseSrcAddress(),
				morseClaimableAccount.ClaimedAtHeight,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)
	}

	return &morseClaimableAccount, nil
}

// checkClaimSigner verifies that the msg was signed by an authorized Morse private key.
//
// It compares the msg's signer address to the morseClaimableAccount's addresses.

// The signers can be found of:
// - morse_node_address (operator; owner IFF output/owner is nil)
// - morse_output_address (owner; operator is most likely a different offchain identity)
//
// morse_node_address:
// - Morse node account is claiming itself (i.e. the operator)
// - Account remains custodial (if output/owner is nil)
// - Account remains non-custodial (if output/owner is non-nil)
//
// morse_output_address:
// - Morse output/owner account is claiming the Morse node account
// - Account remains non-custodial (under output/owner account control)
//
// If the Morse node is claiming itself, checks whether a Morse output address exists to distinguish:
//   - Operator claiming a custodial account
//   - Operator claiming a non-custodial account
//
// Returns:
//   - claimSignerType: Enum describing the signer type
//   - err: Error if signer is not authorized
func checkClaimSigner(
	msg *migrationtypes.MsgClaimMorseSupplier,
	morseClaimableAccount *migrationtypes.MorseClaimableAccount,
) (claimSignerType migrationtypes.MorseSupplierClaimSignerType, err error) {
	// node addr === operator
	nodeAddr := morseClaimableAccount.GetMorseSrcAddress()
	// output addr === owner (MAY be empty)
	outputAddr := morseClaimableAccount.GetMorseOutputAddress()

	// Check the message signer address
	switch msg.GetMorseSignerAddress() {

	// signer === owner && owner !== operator
	// Owner claim (a.k.a non-custodial claim)
	// This is the owner claiming the Morse node/servicer/supplier account
	case outputAddr:
		claimSignerType = migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_NON_CUSTODIAL_SIGNED_BY_OWNER

	// Operator claim
	// I.e., the signer of the message IS NOT the output/owner account.
	// May be custodial or non-custodial depending on whether the output (i.e. owner) is set.
	case nodeAddr:
		switch outputAddr {

		// signer === node addr === operator === owner
		// Custodial claim: No output address is set so the operator === owner === signer
		case "":
			claimSignerType = migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_CUSTODIAL_SIGNED_BY_NODE_ADDR

		// signer === operator === node addr && owner !== operator
		// Non-custodial claim: Output address exists so the operator is claiming the account on behalf of the owner.
		default:
			claimSignerType = migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_NON_CUSTODIAL_SIGNED_BY_NODE_ADDR
		}

	// Signer does not match either the operator or owner address
	default:
		return migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_UNSPECIFIED,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"morse signer address (%s) doesn't match the operator (%s) or owner (%s) address",
				msg.GetMorseSignerAddress(),
				nodeAddr,
				outputAddr,
			)
	}

	return claimSignerType, nil
}
