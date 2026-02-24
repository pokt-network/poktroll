package keeper

import (
	"fmt"
	"sort"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	tlm "github.com/pokt-network/poktroll/x/tokenomics/token_logic_module"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
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
func aggregateMints(results tlm.ClaimSettlementResults) []aggregatedMintBurnOp {
	aggregated := make(map[string]*aggregatedMintBurnOp)

	for _, result := range results {
		for _, mint := range result.GetMints() {
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

	return sortedMintBurnOps(aggregated)
}

// aggregateBurns aggregates burn operations across all settled results by unique
// (DestinationModule, OpReason) key. Returns a deterministically sorted slice.
func aggregateBurns(results tlm.ClaimSettlementResults) []aggregatedMintBurnOp {
	aggregated := make(map[string]*aggregatedMintBurnOp)

	for _, result := range results {
		for _, burn := range result.GetBurns() {
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

	return sortedMintBurnOps(aggregated)
}

// aggregateModToModTransfers aggregates module-to-module transfers across all settled
// results by unique (SenderModule, RecipientModule, OpReason) key.
// Returns a deterministically sorted slice.
func aggregateModToModTransfers(results tlm.ClaimSettlementResults) []aggregatedModToModTransfer {
	aggregated := make(map[string]*aggregatedModToModTransfer)

	for _, result := range results {
		for _, transfer := range result.GetModToModTransfers() {
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
	return sorted
}

// aggregateModToAcctTransfers aggregates module-to-account transfers across all settled
// results by unique (SenderModule, RecipientAddress, OpReason) key.
// Returns a deterministically sorted slice.
func aggregateModToAcctTransfers(results tlm.ClaimSettlementResults) []aggregatedModToAcctTransfer {
	aggregated := make(map[string]*aggregatedModToAcctTransfer)

	for _, result := range results {
		for _, transfer := range result.GetModToAcctTransfers() {
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
	return sorted
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
