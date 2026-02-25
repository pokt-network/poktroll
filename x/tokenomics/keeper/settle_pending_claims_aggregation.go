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
	aggregated := make(map[string]*aggregatedMintBurnOp)

	for _, result := range results {
		for _, mint := range result.GetMints() {
			if err := mint.Validate(); err != nil {
				return nil, errUnspecifiedOpReason
			}
			key := fmt.Sprintf("%s|%d", mint.DestinationModule, mint.OpReason)
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
	aggregated := make(map[string]*aggregatedMintBurnOp)

	for _, result := range results {
		for _, burn := range result.GetBurns() {
			if err := burn.Validate(); err != nil {
				return nil, errUnspecifiedOpReason
			}
			key := fmt.Sprintf("%s|%d", burn.DestinationModule, burn.OpReason)
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
	aggregated := make(map[string]*aggregatedModToModTransfer)

	for _, result := range results {
		for _, transfer := range result.GetModToModTransfers() {
			if err := transfer.Validate(); err != nil {
				return nil, errUnspecifiedOpReason
			}
			key := fmt.Sprintf("%s|%s|%d", transfer.SenderModule, transfer.RecipientModule, transfer.OpReason)
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

	keys := make([]string, 0, len(aggregated))
	for k := range aggregated {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sorted := make([]aggregatedModToModTransfer, 0, len(keys))
	for _, k := range keys {
		sorted = append(sorted, *aggregated[k])
	}
	return sorted, nil
}

// aggregateModToAcctTransfers aggregates module-to-account transfers across all settled
// results by unique (SenderModule, RecipientAddress, OpReason) key.
// Returns a deterministically sorted slice.
func aggregateModToAcctTransfers(results tlm.ClaimSettlementResults) ([]aggregatedModToAcctTransfer, error) {
	aggregated := make(map[string]*aggregatedModToAcctTransfer)

	for _, result := range results {
		for _, transfer := range result.GetModToAcctTransfers() {
			if err := transfer.Validate(); err != nil {
				return nil, errUnspecifiedOpReason
			}
			key := fmt.Sprintf("%s|%s|%d", transfer.SenderModule, transfer.RecipientAddress, transfer.OpReason)
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

	keys := make([]string, 0, len(aggregated))
	for k := range aggregated {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sorted := make([]aggregatedModToAcctTransfer, 0, len(keys))
	for _, k := range keys {
		sorted = append(sorted, *aggregated[k])
	}
	return sorted, nil
}

// sortedMintBurnOps is a helper that extracts a deterministically sorted slice
// from a map of aggregated mint/burn operations.
func sortedMintBurnOps(aggregated map[string]*aggregatedMintBurnOp) []aggregatedMintBurnOp {
	keys := make([]string, 0, len(aggregated))
	for k := range aggregated {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	sorted := make([]aggregatedMintBurnOp, 0, len(keys))
	for _, k := range keys {
		sorted = append(sorted, *aggregated[k])
	}
	return sorted
}
