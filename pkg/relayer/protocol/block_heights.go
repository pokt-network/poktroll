package protocol

import (
	"encoding/binary"
	"log"
	"math/rand"

	"github.com/pokt-network/poktroll/pkg/client"
)

// GetEarliestCreateClaimHeight returns the earliest block height at which a claim
// for a session with the given createClaimWindowStartHeight can be created.
//
// TODO_TEST(@bryanchriswhite): Add test coverage and more logs
func GetEarliestCreateClaimHeight(createClaimWindowStartBlock client.Block) int64 {
	createClaimWindowStartBlockHash := createClaimWindowStartBlock.Hash()
	log.Printf("DEBUG: using createClaimWindowStartBlock %d's hash %x as randomness", createClaimWindowStartBlock.Height(), createClaimWindowStartBlockHash)
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
func GetEarliestSubmitProofHeight(submitProofWindowStartBlock client.Block) int64 {
	earliestSubmitProofBlockHash := submitProofWindowStartBlock.Hash()
	log.Printf("DEBUG: using submitProofWindowStartBlock %d's hash %x as randomness", submitProofWindowStartBlock.Height(), earliestSubmitProofBlockHash)
	rngSeed, _ := binary.Varint(earliestSubmitProofBlockHash)
	randomNumber := rand.NewSource(rngSeed).Int63()

	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// randSubmitProofHeightOffset := randomNumber % (claimproofparams.GovSubmitProofIntervalBlocks - claimproofparams.GovSubmitProofWindowBlocks - 1)
	_ = randomNumber
	randSubmitProofHeightOffset := int64(0)

	return submitProofWindowStartBlock.Height() + randSubmitProofHeightOffset
}
