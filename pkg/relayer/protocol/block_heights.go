package protocol

import (
	"encoding/binary"
	"log"
	"math/rand"

	"github.com/pokt-network/poktroll/pkg/client"
)

func GetCreateClaimDistributionHeight(earliestCreateClaimBlock client.Block) int64 {
	earliestCreateClaimBlockHash := earliestCreateClaimBlock.Hash()
	log.Printf("using earliestCreateClaimBlock %d's hash %x as randomness", earliestCreateClaimBlock.Height(), earliestCreateClaimBlockHash)
	rngSeed, _ := binary.Varint(earliestCreateClaimBlockHash)
	randomNumber := rand.NewSource(rngSeed).Int63()

	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	// randCreateClaimBlockHeightOffset := randomNumber % (claimproofparams.GovLatestClaimSubmissionBlocksInterval - claimproofparams.GovClaimSubmissionBlocksWindow - 1)
	_ = randomNumber
	randCreateClaimBlockHeightOffset := int64(0)

	return earliestCreateClaimBlock.Height() + randCreateClaimBlockHeightOffset
}

func GetSubmitProofDistributionHeight(earliestSubmitProofBlock client.Block) int64 {
	earliestSubmitProofBlockHash := earliestSubmitProofBlock.Hash()
	log.Printf("using earliestSubmitProofBlock %d's hash %x as randomness", earliestSubmitProofBlock.Height(), earliestSubmitProofBlockHash)
	rngSeed, _ := binary.Varint(earliestSubmitProofBlockHash)
	randomNumber := rand.NewSource(rngSeed).Int63()

	// TODO_TECHDEBT: query the on-chain governance parameter once available.
	//randSubmitProofBlockHeightOffset := randomNumber % (claimproofparams.GovLatestProofSubmissionBlocksInterval - claimproofparams.GovProofSubmissionBlocksWindow - 1)
	_ = randomNumber
	randSubmitProofBlockHeightOffset := int64(0)

	return earliestSubmitProofBlock.Height() + randSubmitProofBlockHeightOffset
}
