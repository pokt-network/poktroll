package keeper

import (
	"context"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/migration/types"
)

type msgServer struct {
	Keeper
}

// NewMsgServerImpl returns an implementation of the MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// deferAdjustWaivedGasFees ensures that gas fees are NOT waived if one of the following is true:
// - The claim is invalid
// - Morse account has already been claimed
// Claiming gas fees in the cases above ensures that we prevent spamming.
// It returns a function which is intended to be deferred.
//
// Rationale:
//  1. Morse claim txs MAY be signed by Shannon accounts which have 0upokt balances.
//     For this reason, gas fees are waived (in the ante handler) for txs which
//     contain ONLY (one or more) Morse claim messages.
//  2. This exposes a potential resource exhaustion vector (or at least extends the
//     attack surface area) where an attacker would be able to take advantage of
//     the fact that tx signature verification gas costs MAY be avoided under
//     certain conditions.
//  3. ALL Morse account claim message handlers therefore SHOULD ensure that
//     tx signature verification gas costs ARE applied if the claim is EITHER
//     invalid OR if the given Morse account has already been claimed. The latter
//     is necessary to mitigate a replay attack vector.
func (k msgServer) deferAdjustWaivedGasFees(
	ctx context.Context,
	isMorseAccountFound,
	isMorseAccountAlreadyClaimed *bool,
) func() {
	return func() {
		sdkCtx := cosmostypes.UnwrapSDKContext(ctx)
		waiveMorseClaimGasFees := k.GetParams(sdkCtx).WaiveMorseClaimGasFees
		if waiveMorseClaimGasFees && (!*isMorseAccountFound || *isMorseAccountAlreadyClaimed) {
			// Attempt to charge the waived gas fee for invalid claims.
			sdkCtx.GasMeter()
			// DEV_NOTE: Assuming that the tx containing this message was signed
			// by a non-multisig externally owned account (EOA); i.e. secp256k1,
			// conventionally. If this assumption is violated, the "wrong" gas
			// cost will be charged for the given key type.
			gas := k.accountKeeper.GetParams(ctx).SigVerifyCostSecp256k1
			sdkCtx.GasMeter().ConsumeGas(gas, "ante verify: secp256k1")
		}
	}
}
