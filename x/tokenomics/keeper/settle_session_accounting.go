package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	"github.com/pokt-network/poktroll/telemetry"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// SettleSessionAccounting is responsible for all of the post-session accounting
// necessary to burn, mint or transfer tokens depending on the amount of work
// done. The amount of "work done" complete is dictated by `sum` of `root`.
//
// ASSUMPTION: It is assumed the caller of this function validated the claim
// against a proof BEFORE calling this function.
//
// TODO_MAINNET(@Olshansk): Research if there's a way to limit who can call this function?
func (k Keeper) SettleSessionAccounting(
	ctx context.Context,
	claim *prooftypes.Claim,
) (err error) {
	logger := k.Logger().With("method", "SettleSessionAccounting")

	// Declaring variables that will be emitted by telemetry
	settlementCoin := cosmostypes.NewCoin("upokt", math.NewInt(0))
	isSuccessful := false

	// This is emitted only when the function returns (successful or not)
	defer telemetry.EventSuccessCounter(
		"settle_session_accounting",
		func() float32 {
			if settlementCoin.Amount.BigInt() == nil {
				return 0
			}
			return float32(settlementCoin.Amount.Int64())
		},
		func() bool { return isSuccessful },
	)

	// Ensure the claim is not nil
	if claim == nil {
		logger.Error("received a nil claim")
		return tokenomicstypes.ErrTokenomicsClaimNil
	}

	// Retrieve & validate the session header
	sessionHeader := claim.GetSessionHeader()
	if sessionHeader == nil {
		logger.Error("received a nil session header")
		return tokenomicstypes.ErrTokenomicsSessionHeaderNil
	}
	if err = sessionHeader.ValidateBasic(); err != nil {
		logger.Error("received an invalid session header", "error", err)
		return tokenomicstypes.ErrTokenomicsSessionHeaderInvalid
	}

	// Retrieve the supplier address that will be getting rewarded; providing services
	supplierAddr, err := cosmostypes.AccAddressFromBech32(claim.GetSupplierAddress())
	if err != nil || supplierAddr == nil {
		return tokenomicstypes.ErrTokenomicsSupplierAddressInvalid
	}

	// Retrieve the application address that is being charged; getting services
	applicationAddress, err := cosmostypes.AccAddressFromBech32(sessionHeader.GetApplicationAddress())
	if err != nil || applicationAddress == nil {
		return tokenomicstypes.ErrTokenomicsApplicationAddressInvalid
	}

	// Retrieve the root of the claim to determine the amount of work done
	root := (smt.MerkleSumRoot)(claim.GetRootHash())

	// Ensure the root hash is valid
	if !root.HasDigestSize(protocol.TrieHasherSize) {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf(
			"root hash has invalid digest size (%d), expected (%d)",
			root.DigestSize(), protocol.TrieHasherSize,
		)
	}

	// Retrieve the sum of the root hash to determine the amount of work (compute units) done
	claimComputeUnits, err := root.Sum()
	if err != nil {
		return tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrapf("%v", err)
	}

	// Helpers for logging the same metadata throughout this function calls
	logger = logger.With(
		"compute_units", claimComputeUnits,
		"session_id", sessionHeader.GetSessionId(),
		"supplier", supplierAddr,
		"application", applicationAddress,
	)
	logger.Info("About to start session settlement accounting")

	// Retrieve the on-chain staked application record
	application, foundApplication := k.applicationKeeper.GetApplication(ctx, applicationAddress.String())
	if !foundApplication {
		logger.Warn(fmt.Sprintf("application for claim with address %q not found", applicationAddress))
		return tokenomicstypes.ErrTokenomicsApplicationNotFound
	}

	// Retrieve the on-chain staked application record
	supplier, foundSupplier := k.supplierKeeper.GetSupplier(ctx, supplierAddr.String())
	if !foundSupplier {
		logger.Warn(fmt.Sprintf("supplier for claim with address %q not found", supplierAddr))
		return tokenomicstypes.ErrTokenomicsSupplierNotFound
	}

	// Determine the total number of tokens that'll be used for settling the session.
	// When the network achieves equilibrium, this will be the mint & burn.
	// TODO_IN_THIS_PR: Simplify: this function should just take in a ServiceId and the one below should take in the root
	computeUnitsPerRelay, err := k.getComputUnitsPerRelayFromApplication(application, sessionHeader.Service.Id)
	if err != nil {
		return err
	}
	computeUnitsToTokensMultiplier := k.GetParams(ctx).ComputeUnitsToTokensMultiplier
	settlementCoin, err = relayCountToCoin(claimComputeUnits, computeUnitsPerRelay, computeUnitsToTokensMultiplier)
	if err != nil {
		return err
	}

	// Start claiming log line!
	logger.Info(fmt.Sprintf(
		"About to start claiming (%d) compute units equate to (%s) coins for session (%s) with ComputeUnitsPerRelay (%d) and ComputeUnitsToTokenMultiplier (%d)",
		claimComputeUnits,
		settlementCoin,
		sessionHeader.SessionId,
		computeUnitsPerRelay,
		computeUnitsToTokensMultiplier,
	))

	//
	if err := k.ProcessTokenLogicModules(
		ctx,
		claim.SessionHeader,
		&application,
		&supplier,
		settlementCoin,
	); err != nil {
		logger.Warn(fmt.Sprintf("failed to trigger the token-logic-module settle session accounting for supplier %s", supplierAddr))
		return err
	}

	// Update the application's on-chain record
	k.applicationKeeper.SetApplication(ctx, application)
	logger.Info(fmt.Sprintf("updated on-chain application record with address %q", applicationAddress))

	// Update the application's on-chain record
	k.supplierKeeper.SetSupplier(ctx, supplier)
	logger.Info(fmt.Sprintf("updated on-chain supplier record with address %q", supplierAddr))

	// Update isSuccessful to true for telemetry
	isSuccessful = true
	return nil
}

// relayCountToCoin calculates the amount of uPOKT to mint based on the number of relays, the service-specific ComputeUnitsPerRelay, and the ComputeUnitsPerTokenMultiplier tokenomics param
// TODO_IN_THIS_PR: What if we use root smt.MerkleRoot instead?
func relayCountToCoin(numRelays, computeUnitsPerRelay uint64, computeUnitsToTokensMultiplier uint64) (cosmostypes.Coin, error) {
	upokt := math.NewInt(int64(numRelays * computeUnitsPerRelay * computeUnitsToTokensMultiplier))

	if upokt.IsNegative() {
		return cosmostypes.Coin{}, tokenomicstypes.ErrTokenomicsRootHashInvalid.Wrap("sum * compute_units_to_tokens_multiplier is negative")
	}

	return cosmostypes.NewCoin(volatile.DenomuPOKT, upokt), nil
}

// getComputUnitsPerRelayFromApplication retrieves the ComputeUnitsPerRelay for a given service from the application's service configs
func (k Keeper) getComputUnitsPerRelayFromApplication(application apptypes.Application, serviceID string) (cupr uint64, err error) {
	logger := k.Logger().With("method", "getComputeUnitsPerRelayFromApplication")

	serviceConfigs := application.ServiceConfigs
	if len(serviceConfigs) == 0 {
		logger.Warn(fmt.Sprintf("application with address %q has no service configs", application.Address))
		return 0, tokenomicstypes.ErrTokenomicsApplicationNoServiceConfigs
	}

	for _, sc := range serviceConfigs {
		service := sc.GetService()
		if service.Id == serviceID {
			return service.ComputeUnitsPerRelay, nil
		}
	}

	logger.Warn(fmt.Sprintf("service with ID %q not found in application with address %q", serviceID, application.Address))
	return 0, tokenomicstypes.ErrTokenomicsApplicationNoServiceConfigs
}
