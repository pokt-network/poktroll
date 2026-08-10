package keeper

import (
	"fmt"
	"sort"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

// errUnspecifiedOpReason is returned when a settlement operation has an UNSPECIFIED reason,
// indicating a programming bug in the TLM that produced it.
var errUnspecifiedOpReason = tokenomicstypes.ErrTokenomicsSettlementInternal.Wrap(
	"settlement operation has UNSPECIFIED reason; this indicates a TLM bug",
)

// Aggregation map keys. Struct keys avoid the per-operation fmt.Sprintf heap allocation
// that previously ran in the innermost aggregation loop (~181K string allocations per
// mainnet settlement block); the equivalent "%s|%d"-style string is now built only at sort
// time, once per unique key (a few dozen), preserving the exact prior ordering.
type mintBurnKey struct {
	DestinationModule string
	OpReason          tokenomicstypes.SettlementOpReason
}

type modToModKey struct {
	SenderModule    string
	RecipientModule string
	OpReason        tokenomicstypes.SettlementOpReason
}

type modToAcctKey struct {
	SenderModule     string
	RecipientAddress string
	OpReason         tokenomicstypes.SettlementOpReason
}

// aggregatedMintBurnOp holds the aggregated mint or burn operation for a unique key.
type aggregatedMintBurnOp struct {
	DestinationModule string
	OpReason          tokenomicstypes.SettlementOpReason
	Coin              cosmostypes.Coin
	NumClaims         uint32
}

// aggregatedModToModTransfer holds the aggregated module-to-module transfer for a unique key.
type aggregatedModToModTransfer struct {
	SenderModule    string
	RecipientModule string
	OpReason        tokenomicstypes.SettlementOpReason
	Coin            cosmostypes.Coin
	NumClaims       uint32
}

// aggregatedModToAcctTransfer holds the aggregated module-to-account transfer for a unique key.
type aggregatedModToAcctTransfer struct {
	SenderModule     string
	RecipientAddress string
	OpReason         tokenomicstypes.SettlementOpReason
	Coin             cosmostypes.Coin
	NumClaims        uint32
}

// aggregateMints aggregates mint operations across all settled results by unique
// (DestinationModule, OpReason) key. Returns a deterministically sorted slice.
func aggregateMints(results tlm.ClaimSettlementResults) ([]aggregatedMintBurnOp, error) {
	aggregated := make(map[mintBurnKey]*aggregatedMintBurnOp)

	for _, result := range results {
		for _, mint := range result.GetMints() {
			if err := mint.Validate(); err != nil {
				return nil, errUnspecifiedOpReason
			}
			key := mintBurnKey{DestinationModule: mint.DestinationModule, OpReason: mint.OpReason}
			if existing, ok := aggregated[key]; ok {
				existing.Coin = existing.Coin.Add(mint.Coin)
				existing.NumClaims++
			} else {
				aggregated[key] = &aggregatedMintBurnOp{
					DestinationModule: mint.DestinationModule,
					OpReason:          mint.OpReason,
					Coin:              mint.Coin,
					NumClaims:         1,
				}
			}
		}
	}

	return sortedMintBurnOps(aggregated), nil
}

// aggregateBurns aggregates burn operations across all settled results by unique
// (DestinationModule, OpReason) key. Returns a deterministically sorted slice.
func aggregateBurns(results tlm.ClaimSettlementResults) ([]aggregatedMintBurnOp, error) {
	aggregated := make(map[mintBurnKey]*aggregatedMintBurnOp)

	for _, result := range results {
		for _, burn := range result.GetBurns() {
			if err := burn.Validate(); err != nil {
				return nil, errUnspecifiedOpReason
			}
			key := mintBurnKey{DestinationModule: burn.DestinationModule, OpReason: burn.OpReason}
			if existing, ok := aggregated[key]; ok {
				existing.Coin = existing.Coin.Add(burn.Coin)
				existing.NumClaims++
			} else {
				aggregated[key] = &aggregatedMintBurnOp{
					DestinationModule: burn.DestinationModule,
					OpReason:          burn.OpReason,
					Coin:              burn.Coin,
					NumClaims:         1,
				}
			}
		}
	}

	return sortedMintBurnOps(aggregated), nil
}

// aggregateModToModTransfers aggregates module-to-module transfers across all settled
// results by unique (SenderModule, RecipientModule, OpReason) key.
// Returns a deterministically sorted slice.
func aggregateModToModTransfers(results tlm.ClaimSettlementResults) ([]aggregatedModToModTransfer, error) {
	aggregated := make(map[modToModKey]*aggregatedModToModTransfer)

	for _, result := range results {
		for _, transfer := range result.GetModToModTransfers() {
			if err := transfer.Validate(); err != nil {
				return nil, errUnspecifiedOpReason
			}
			key := modToModKey{SenderModule: transfer.SenderModule, RecipientModule: transfer.RecipientModule, OpReason: transfer.OpReason}
			if existing, ok := aggregated[key]; ok {
				existing.Coin = existing.Coin.Add(transfer.Coin)
				existing.NumClaims++
			} else {
				aggregated[key] = &aggregatedModToModTransfer{
					SenderModule:    transfer.SenderModule,
					RecipientModule: transfer.RecipientModule,
					OpReason:        transfer.OpReason,
					Coin:            transfer.Coin,
					NumClaims:       1,
				}
			}
		}
	}

	sorted := make([]aggregatedModToModTransfer, 0, len(aggregated))
	for _, op := range aggregated {
		sorted = append(sorted, *op)
	}
	// Preserve the exact prior ordering (sort.Strings over "sender|recipient|reason").
	sort.Slice(sorted, func(i, j int) bool {
		return modToModOrderKey(sorted[i]) < modToModOrderKey(sorted[j])
	})
	return sorted, nil
}

// modToModOrderKey reproduces the legacy string sort key so aggregated transfers execute
// (and emit events) in the same order as before struct keys were introduced.
func modToModOrderKey(op aggregatedModToModTransfer) string {
	return fmt.Sprintf("%s|%s|%d", op.SenderModule, op.RecipientModule, op.OpReason)
}

// aggregateModToAcctTransfers aggregates module-to-account transfers across all settled
// results by unique (SenderModule, RecipientAddress, OpReason) key.
// Returns a deterministically sorted slice.
func aggregateModToAcctTransfers(results tlm.ClaimSettlementResults) ([]aggregatedModToAcctTransfer, error) {
	aggregated := make(map[modToAcctKey]*aggregatedModToAcctTransfer)

	for _, result := range results {
		for _, transfer := range result.GetModToAcctTransfers() {
			if err := transfer.Validate(); err != nil {
				return nil, errUnspecifiedOpReason
			}
			key := modToAcctKey{SenderModule: transfer.SenderModule, RecipientAddress: transfer.RecipientAddress, OpReason: transfer.OpReason}
			if existing, ok := aggregated[key]; ok {
				existing.Coin = existing.Coin.Add(transfer.Coin)
				existing.NumClaims++
			} else {
				aggregated[key] = &aggregatedModToAcctTransfer{
					SenderModule:     transfer.SenderModule,
					RecipientAddress: transfer.RecipientAddress,
					OpReason:         transfer.OpReason,
					Coin:             transfer.Coin,
					NumClaims:        1,
				}
			}
		}
	}

	sorted := make([]aggregatedModToAcctTransfer, 0, len(aggregated))
	for _, op := range aggregated {
		sorted = append(sorted, *op)
	}
	// Preserve the exact prior ordering (sort.Strings over "sender|recipient|reason").
	sort.Slice(sorted, func(i, j int) bool {
		return modToAcctOrderKey(sorted[i]) < modToAcctOrderKey(sorted[j])
	})
	return sorted, nil
}

// modToAcctOrderKey reproduces the legacy string sort key so aggregated transfers execute
// (and emit events) in the same order as before struct keys were introduced.
func modToAcctOrderKey(op aggregatedModToAcctTransfer) string {
	return fmt.Sprintf("%s|%s|%d", op.SenderModule, op.RecipientAddress, op.OpReason)
}

// sortedMintBurnOps extracts a deterministically sorted slice from a map of aggregated
// mint/burn operations, preserving the exact prior ordering (sort.Strings over
// "module|reason").
func sortedMintBurnOps(aggregated map[mintBurnKey]*aggregatedMintBurnOp) []aggregatedMintBurnOp {
	sorted := make([]aggregatedMintBurnOp, 0, len(aggregated))
	for _, op := range aggregated {
		sorted = append(sorted, *op)
	}
	sort.Slice(sorted, func(i, j int) bool {
		return mintBurnOrderKey(sorted[i]) < mintBurnOrderKey(sorted[j])
	})
	return sorted
}

// mintBurnOrderKey reproduces the legacy string sort key so aggregated mint/burn ops
// execute (and emit events) in the same order as before struct keys were introduced.
func mintBurnOrderKey(op aggregatedMintBurnOp) string {
	return fmt.Sprintf("%s|%d", op.DestinationModule, op.OpReason)
}
