package keeper

import (
	"context"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/pokt-network/poktroll/app/volatile"
	migrationtypes "github.com/pokt-network/poktroll/x/migration/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
)

// ClaimMorseSupplier performs the following steps, given msg is valid and a
// MorseClaimableAccount exists for the given morse_src_address:
//   - Mint and transfer all tokens (unstaked balance plus supplier stake) of the
//     MorseClaimableAccount to the shannonDestAddress.
//   - Mark the MorseClaimableAccount as claimed (i.e. adding the shannon_dest_address
//     and claimed_at_height).
//   - Stake a supplier for the amount specified in the MorseClaimableAccount,
//     and the services specified in the msg.
func (k msgServer) ClaimMorseSupplier(
	ctx context.Context,
	msg *migrationtypes.MsgClaimMorseSupplier,
) (*migrationtypes.MsgClaimMorseSupplierResponse, error) {
	sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
	logger := k.Logger().With("method", "ClaimMorseSupplier")
	waiveMorseClaimGasFees := k.GetParams(sdkCtx).WaiveMorseClaimGasFees

	// Ensure that gas fees are NOT waived if one of the following is true:
	// - The claim is invalid
	// - Morse account has already been claimed
	// Claiming gas fees in the cases above ensures that we prevent spamming.
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
		if waiveMorseClaimGasFees && (!isFound || !isValid || isAlreadyClaimed) {
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
	// shannonOwnerAddress and shannonOperatorAddress are validated in MsgClaimMorseSupplier#ValidateBasic().
	shannonOwnerAddr := cosmostypes.MustAccAddressFromBech32(msg.ShannonOwnerAddress)

	// Default to the shannonOwnerAddr as the shannonOperatorAddr if not provided.
	// The shannonOperatorAddr is where the Morse node/supplier unstaked balance will be minted to.
	shannonOperatorAddr := shannonOwnerAddr
	if msg.ShannonOperatorAddress != "" {
		shannonOperatorAddr = cosmostypes.MustAccAddressFromBech32(msg.ShannonOperatorAddress)
	}

	// Ensure that a MorseClaimableAccount exists for the given morseSrcAddress.
	morseClaimableAccount, isFound = k.GetMorseClaimableAccount(
		sdkCtx,
		msg.GetMorseNodeAddress(),
	)
	if !isFound {
		return nil, status.Error(
			codes.NotFound,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"no morse claimable account exists with address %q",
				msg.GetMorseNodeAddress(),
			).Error(),
		)
	}

	// Ensure that the given MorseClaimableAccount has not already been claimed.
	if morseClaimableAccount.IsClaimed() {
		isAlreadyClaimed = true
		return nil, status.Error(
			codes.FailedPrecondition,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"morse address %q has already been claimed at height %d by shannon address %q",
				morseClaimableAccount.GetMorseSrcAddress(),
				morseClaimableAccount.ClaimedAtHeight,
				morseClaimableAccount.ShannonDestAddress,
			).Error(),
		)
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
				"Morse account %q is not staked as an supplier or application, please use `pocketd tx migration claim-account` instead",
				morseClaimableAccount.GetMorseSrcAddress(),
			).Error(),
		)
	}

	// Ensure the signer is ONE OF THE FOLLOWING:
	// - The Morse node address (i.e. operator)
	// - The Morse output address (i.e. owner)
	claimSignerType, err := checkClaimSigner(msg, &morseClaimableAccount)
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
	case migrationtypes.MorseSupplierClaimSignerType_MORSE_SUPPLIER_CLAIM_SIGNER_TYPE_NON_CUSTODIAL_SIGNED_BY_OWNER:
		shannonSigningAddress = shannonOwnerAddr
	}

	// Mint the Morse node/supplier's stake to the shannonSigningAddress account balance.
	// The Supplier stake is subsequently escrowed from the shannonSigningAddress account balance.
	// NOTE: The supplier module's staking fee parameter will be deducted from the claimed balance below.
	if err = k.MintClaimedMorseTokens(ctx, shannonSigningAddress, morseClaimableAccount.GetSupplierStake()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Mint the Morse node/supplier's unstaked balance to the shannonOperatorAddress account balance.
	if err = k.MintClaimedMorseTokens(ctx, shannonOperatorAddr, morseClaimableAccount.GetUnstakedBalance()); err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}

	// Set ShannonDestAddress & ClaimedAtHeight (claim).
	morseClaimableAccount.ShannonDestAddress = shannonOperatorAddr.String()
	morseClaimableAccount.ClaimedAtHeight = sdkCtx.BlockHeight()

	// Update the MorseClaimableAccount.
	k.SetMorseClaimableAccount(
		sdkCtx,
		morseClaimableAccount,
	)

	// Query for any existing supplier stake prior to staking.
	preClaimSupplierStake := cosmostypes.NewCoin(volatile.DenomuPOKT, math.ZeroInt())
	foundSupplier, isFound := k.supplierKeeper.GetSupplier(ctx, shannonOperatorAddr.String())
	if isFound {
		preClaimSupplierStake = *foundSupplier.Stake
	}

	sharedParams := k.sharedKeeper.GetParams(sdkCtx)
	sessionEndHeight := sharedtypes.GetSessionEndHeight(&sharedParams, sdkCtx.BlockHeight())

	postClaimSupplierStake := preClaimSupplierStake.Add(morseClaimableAccount.GetSupplierStake())
	minStake := k.supplierKeeper.GetParams(ctx).MinStake

	// If the claimed supplier stake is less than the minimum stake, the supplier is immediately unstaked.
	if postClaimSupplierStake.Amount.LT(minStake.Amount) {
		currentSessionStart := sharedtypes.GetSessionStartHeight(&sharedParams, sdkCtx.BlockHeight())
		previousSessionEnd := sharedtypes.GetSessionEndHeight(&sharedParams, currentSessionStart-1)
		supplier := &sharedtypes.Supplier{
			OwnerAddress:            shannonOwnerAddr.String(),
			OperatorAddress:         shannonOperatorAddr.String(),
			Stake:                   &postClaimSupplierStake,
			UnstakeSessionEndHeight: uint64(previousSessionEnd),
			// Services:             (intentionally omitted, no services were staked),
			// ServiceConfigHistory: (intentionally omitted, no services were staked),
		}

		// Emit an event which signals that the morse supplier has been claimed.
		morseSupplierClaimedEvent := migrationtypes.EventMorseSupplierClaimed{
			MorseNodeAddress:     msg.GetMorseNodeAddress(),
			MorseOutputAddress:   morseClaimableAccount.GetMorseOutputAddress(),
			ClaimSignerType:      claimSignerType,
			ClaimedBalance:       morseClaimableAccount.TotalTokens(),
			ClaimedSupplierStake: cosmostypes.Coin{},
			SessionEndHeight:     sessionEndHeight,
			Supplier:             supplier,
		}
		if err = sdkCtx.EventManager().EmitTypedEvent(&morseSupplierClaimedEvent); err != nil {
			return nil, status.Error(
				codes.Internal,
				migrationtypes.ErrMorseSupplierClaim.Wrapf(
					"failed to emit event type %T: %v",
					&morseSupplierClaimedEvent,
					err,
				).Error(),
			)
		}

		// Emit an event which signals that the morse supplier was unstaked.
		morseSupplierUnstakedEvent := suppliertypes.EventSupplierUnbondingEnd{
			Supplier:         supplier,
			Reason:           suppliertypes.SupplierUnbondingReason_SUPPLIER_UNBONDING_REASON_BELOW_MIN_STAKE,
			SessionEndHeight: sessionEndHeight,
			// Unstaking when claiming Suppliers below min-stake takes effect IMMEDIATELY.
			UnbondingEndHeight: sessionEndHeight,
		}
		if err = sdkCtx.EventManager().EmitTypedEvent(&morseSupplierUnstakedEvent); err != nil {
			return nil, status.Error(
				codes.Internal,
				migrationtypes.ErrMorseSupplierClaim.Wrapf(
					"failed to emit event type %T: %v",
					&morseSupplierClaimedEvent,
					err,
				).Error(),
			)
		}

		// Claimed suppliers with less than the min stake are immediately unstaked.
		// NOTE: All stake has already been minted to shannonSignerAddr account,
		// and all unstaked tokens have already been minted to shannonOperatorAddr account.

		return &migrationtypes.MsgClaimMorseSupplierResponse{
			MorseNodeAddress:     msg.GetMorseNodeAddress(),
			MorseOutputAddress:   morseClaimableAccount.GetMorseOutputAddress(),
			ClaimSignerType:      claimSignerType,
			ClaimedBalance:       morseClaimableAccount.TotalTokens(),
			ClaimedSupplierStake: cosmostypes.Coin{},
			SessionEndHeight:     sessionEndHeight,
			Supplier:             supplier,
		}, nil
	}

	// Stake (or update) the supplier.
	msgStakeSupplier := suppliertypes.NewMsgStakeSupplier(
		shannonSigningAddress.String(),
		shannonOwnerAddr.String(),
		shannonOperatorAddr.String(),
		postClaimSupplierStake,
		msg.Services,
	)
	supplier, err := k.supplierKeeper.StakeSupplier(ctx, logger, msgStakeSupplier)
	if err != nil {
		// DEV_NOTE: StakeSupplier SHOULD ALWAYS return a gRPC status error.
		return nil, err
	}

	claimedSupplierStake := morseClaimableAccount.GetSupplierStake()
	claimedUnstakedBalance := morseClaimableAccount.GetUnstakedBalance()

	// Emit an event which signals that the morse account has been claimed.
	event := migrationtypes.EventMorseSupplierClaimed{
		MorseNodeAddress:     msg.GetMorseNodeAddress(),
		MorseOutputAddress:   morseClaimableAccount.GetMorseOutputAddress(),
		ClaimSignerType:      claimSignerType,
		ClaimedBalance:       claimedUnstakedBalance,
		ClaimedSupplierStake: claimedSupplierStake,
		SessionEndHeight:     sessionEndHeight,
		Supplier:             supplier,
	}
	if err = sdkCtx.EventManager().EmitTypedEvent(&event); err != nil {
		return nil, status.Error(
			codes.Internal,
			migrationtypes.ErrMorseSupplierClaim.Wrapf(
				"failed to emit event type %T: %v",
				&event,
				err,
			).Error(),
		)
	}

	// Return the response.
	return &migrationtypes.MsgClaimMorseSupplierResponse{
		MorseOutputAddress:   morseClaimableAccount.GetMorseOutputAddress(),
		MorseNodeAddress:     msg.GetMorseNodeAddress(),
		ClaimSignerType:      claimSignerType,
		ClaimedBalance:       claimedUnstakedBalance,
		ClaimedSupplierStake: claimedSupplierStake,
		SessionEndHeight:     sessionEndHeight,
		Supplier:             supplier,
	}, nil
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
