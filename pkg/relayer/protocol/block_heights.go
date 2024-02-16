package protocol

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"

	"github.com/pokt-network/poktroll/pkg/client"
	"github.com/pokt-network/poktroll/pkg/polylog"
)

// GetEarliestCreateClaimHeight returns the earliest block height at which a claim
// for a session with the given createClaimWindowStartHeight can be created.
//
// TODO_TEST(@bryanchriswhite): Add test coverage and more logs
func GetEarliestCreateClaimHeight(ctx context.Context, createClaimWindowStartBlock client.Block) int64 {
	logger := polylog.Ctx(ctx)

	createClaimWindowStartBlockHash := createClaimWindowStartBlock.Hash()
	logger.Debug().
		Int64(
			"create_claim_window_start_block",
			createClaimWindowStartBlock.Height(),
		).
		Str(
			"create_claim_window_start_block_hash",
			// TODO_TECHDEBT: add polylog.Event#Hex() type method.
			fmt.Sprintf("%x", createClaimWindowStartBlockHash),
		)
	rngSeed, _ := binary.Varint(createClaimWindowStartBlockHash)
	randomNumber := rand.NewSource(rngSeed).Int63()

	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// randCreateClaimHeightOffset := randomNumber % (claimproofparams.GovCreateClaimIntervalBlocks - claimproofparams.GovCreateClaimWindowBlocks - 1)
	_ = randomNumber
	randCreateClaimHeightOffset := int64(0)

	return createClaimWindowStartBlock.Height() + randCreateClaimHeightOffset
}

// GetEarliestSubmitProofHeight returns the earliest block height at which a proof
// for a session with the given submitProofWindowStartHeight can be submitted.
//
// TODO_TEST(@bryanchriswhite): Add test coverage and more logs
func GetEarliestSubmitProofHeight(ctx context.Context, submitProofWindowStartBlock client.Block) int64 {
	logger := polylog.Ctx(ctx)

	earliestSubmitProofBlockHash := submitProofWindowStartBlock.Hash()
	logger.Debug().
		Int64(
			"submit_proof_window_start_block",
			submitProofWindowStartBlock.Height(),
		).
		Str(
			"submit_proof_window_start_block_hash",
			fmt.Sprintf("%x", earliestSubmitProofBlockHash),
		)
	rngSeed, _ := binary.Varint(earliestSubmitProofBlockHash)
	randomNumber := rand.NewSource(rngSeed).Int63()

	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// randSubmitProofHeightOffset := randomNumber % (claimproofparams.GovSubmitProofIntervalBlocks - claimproofparams.GovSubmitProofWindowBlocks - 1)
	_ = randomNumber
	randSubmitProofHeightOffset := int64(0)

	return submitProofWindowStartBlock.Height() + randSubmitProofHeightOffset
}
