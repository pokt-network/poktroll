package main

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"os"
	"strconv"

	comettypes "github.com/cometbft/cometbft/types"
	sdkclient "github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	txtypes "github.com/cosmos/cosmos-sdk/types/tx"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
)

const defaultRPCEndpoint = "https://shannon-grove-rpc.mainnet.poktroll.com"

// Initialize cumulative bytes counters
var originalCumulativeBytes, modifiedCumulativeBytes, failedMsgSubmitProofCount uint64
var totalTxsWithMsgSubmitProof, totalMsgSubmitProofCount uint64

func main() {
	fmt.Println("=== Proof Payload Hash Analysis ===")

	// Parse command line arguments
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run proof_payload_hash_analysis.go <block_height> [rpc_endpoint]")
		fmt.Println("Example: go run proof_payload_hash_analysis.go 171558")
		fmt.Printf("Example: go run proof_payload_hash_analysis.go 171558 %s\n", defaultRPCEndpoint)
		os.Exit(1)
	}

	blockHeight, err := strconv.ParseInt(os.Args[1], 10, 64)
	if err != nil {
		log.Fatalf("Invalid block height: %v", err)
	}

	rpcEndpoint := defaultRPCEndpoint
	if len(os.Args) > 2 {
		rpcEndpoint = os.Args[2]
	}

	fmt.Printf("Fetching block height %d from %s\n\n", blockHeight, rpcEndpoint)

	// Fetch block results from RPC endpoint
	cometClient, err := sdkclient.NewClientFromNode(rpcEndpoint)
	block, err := cometClient.Block(context.Background(), &blockHeight)
	if err != nil {
		log.Fatalf("Error fetching block: %v", err)
	}

	if err := processBlockTxs(block.Block); err != nil {
		log.Fatalf("Error processing block txs: %v", err)
	}
}

// processBlockResults processes block results and analyzes MsgSubmitProof transactions
func processBlockTxs(block *comettypes.Block) error {
	// Set up codec for protobuf unmarshaling
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	prooftypes.RegisterInterfaces(interfaceRegistry)
	servicetypes.RegisterInterfaces(interfaceRegistry)
	sdk.RegisterInterfaces(interfaceRegistry)

	// Register crypto types
	interfaceRegistry.RegisterImplementations(
		(*cryptotypes.PubKey)(nil),
		&ed25519.PubKey{},
		&secp256k1.PubKey{},
	)

	cdc := codec.NewProtoCodec(interfaceRegistry)

	fmt.Printf("Processing block height: %d\n", block.Height)

	// Process each transaction result
	for i, tx := range block.Txs {
		fmt.Printf("=== Processing Transaction %d ===\n", i+1)

		// Step 2: Convert transaction data into bytes (already have them)
		originalCumulativeBytes += uint64(len(tx))

		// Steps 4-9: Process all MsgSubmitProof messages in the transaction
		processedTxBytes, err := processTransactionMsgSubmitProofs(tx, cdc)
		if err != nil {
			fmt.Printf("Error processing transaction: %v\n", err)
			// Use original bytes as fallback
			modifiedCumulativeBytes += uint64(len(tx))
		} else {
			modifiedCumulativeBytes += uint64(len(processedTxBytes))
			fmt.Printf("Processed transaction: %d bytes -> %d bytes (saved: %d bytes)\n",
				len(tx), len(processedTxBytes), len(tx)-len(processedTxBytes))
		}

		fmt.Printf("\n")
	}

	// Step 10: Compare the two cumulative bytes
	fmt.Printf("=== Final Results ===\n")
	fmt.Printf("Total transactions with MsgSubmitProof: %d\n", totalTxsWithMsgSubmitProof)
	fmt.Printf("Total MsgSubmitProof messages: %d\n", totalMsgSubmitProofCount)
	fmt.Printf("MsgSubmitProof messages failed processing: %d\n", failedMsgSubmitProofCount)
	fmt.Printf("Original cumulative bytes: %d\n", originalCumulativeBytes)
	fmt.Printf("Modified cumulative bytes: %d\n", modifiedCumulativeBytes)
	fmt.Printf("Total bytes saved: %d\n", originalCumulativeBytes-modifiedCumulativeBytes)

	if originalCumulativeBytes > 0 {
		compressionRatio := float64(originalCumulativeBytes-modifiedCumulativeBytes) / float64(originalCumulativeBytes) * 100
		fmt.Printf("Compression ratio: %.2f%%\n", compressionRatio)
	}

	return nil
}

// processTransactionMsgSubmitProofs processes all MsgSubmitProof messages in a transaction
func processTransactionMsgSubmitProofs(txBytes []byte, cdc codec.Codec) ([]byte, error) {
	// Unmarshal the transaction to extract all messages
	var tx txtypes.Tx
	if err := cdc.Unmarshal(txBytes, &tx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal transaction: %w", err)
	}

	// Track if any messages were modified
	var modified bool
	var msgSubmitProofCount int

	// Process each message in the transaction
	for i, anyMsg := range tx.Body.Messages {
		// Check if this is a MsgSubmitProof
		if anyMsg.TypeUrl == "/pocket.proof.MsgSubmitProof" {
			msgSubmitProofCount++
			fmt.Printf("Processing MsgSubmitProof message %d/%d\n", i+1, len(tx.Body.Messages))

			// Unmarshal the MsgSubmitProof
			var msg prooftypes.MsgSubmitProof
			if err := cdc.Unmarshal(anyMsg.Value, &msg); err != nil {
				fmt.Printf("Failed to unmarshal MsgSubmitProof: %v\n", err)
				failedMsgSubmitProofCount++
				continue
			}

			// Process the proof to replace payload with hash
			modifiedProofBz, err := processProofPayloadHash(&msg, cdc)
			if err != nil {
				fmt.Printf("Failed to process proof: %v\n", err)
				failedMsgSubmitProofCount++
				continue
			}

			if len(modifiedProofBz) != len(msg.Proof) {
				// Update the message with the modified proof
				msg.Proof = modifiedProofBz

				// Re-marshal the modified message
				modifiedMsgBz, err := cdc.Marshal(&msg)
				if err != nil {
					fmt.Printf("Failed to marshal modified MsgSubmitProof: %v\n", err)
					failedMsgSubmitProofCount++
					continue
				}

				// Update the Any message with the modified bytes
				anyMsg.Value = modifiedMsgBz
				modified = true
			}
		}
	}

	// Update global counters
	if msgSubmitProofCount > 0 {
		totalTxsWithMsgSubmitProof++
		totalMsgSubmitProofCount += uint64(msgSubmitProofCount)
	}

	// Marshal the modified transaction if any changes were made
	if modified {
		modifiedTxBytes, err := cdc.Marshal(&tx)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal modified transaction: %w", err)
		}

		return modifiedTxBytes, nil
	}

	// No changes made, return original bytes
	return txBytes, nil
}

// processProofPayloadHash processes a single MsgSubmitProof to replace RelayResponse payload with hash
func processProofPayloadHash(msg *prooftypes.MsgSubmitProof, cdc codec.Codec) ([]byte, error) {
	// Step 5: Extract the RelayResponse from the MsgSubmitProof.Proof
	sparseCompactProof := &smt.SparseCompactMerkleClosestProof{}
	if err := sparseCompactProof.Unmarshal(msg.Proof); err != nil {
		return nil, fmt.Errorf("failed to unmarshal sparse compact merkle closest proof: %w", err)
	}

	// Decompact the proof to access the relay data
	sparseMerkleProof, err := smt.DecompactClosestProof(sparseCompactProof, protocol.NewSMTSpec())
	if err != nil {
		return nil, fmt.Errorf("failed to decompact sparse merkle proof: %w", err)
	}

	// Get the relay from the proof's value hash
	relayBz := sparseMerkleProof.GetValueHash(protocol.NewSMTSpec())
	var relay servicetypes.Relay
	if err := cdc.Unmarshal(relayBz, &relay); err != nil {
		return nil, fmt.Errorf("failed to unmarshal relay: %w", err)
	}

	// Replace the RelayResponse.Payload with its SHA256 hash
	var payloadModified bool
	if relay.Res != nil && len(relay.Res.Payload) > 0 {
		originalPayloadSize := len(relay.Res.Payload)

		// Calculate SHA256 hash of the payload
		hasher := sha256.New()
		hasher.Write(relay.Res.Payload)
		relay.Res.Payload = hasher.Sum(nil)

		payloadModified = true
		bytesSaved := originalPayloadSize - len(relay.Res.Payload)
		fmt.Printf("    Replaced payload with hash: %d bytes -> %d bytes (saved: %d bytes)\n",
			originalPayloadSize, len(relay.Res.Payload), bytesSaved)
	}

	if !payloadModified {
		// No changes made
		return msg.Proof, nil
	}

	// Re-marshal the modified relay
	modifiedRelayBz, err := cdc.Marshal(&relay)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modified relay: %w", err)
	}

	// Rebuild the SparseCompactMerkleClosestProof with the modified relay
	// Use the same path as the original proof, but replace the value with the modified relay

	// Create a new sparse merkle proof with the modified relay
	modifiedSparseMerkleProof := &smt.SparseMerkleClosestProof{
		Path:             sparseMerkleProof.Path,
		FlippedBits:      sparseMerkleProof.FlippedBits,
		Depth:            sparseMerkleProof.Depth,
		ClosestPath:      sparseMerkleProof.ClosestPath,
		ClosestValueHash: modifiedRelayBz,
		ClosestProof:     sparseMerkleProof.ClosestProof,
	}

	// Compact the modified proof
	modifiedSparseCompactProof, err := smt.CompactClosestProof(modifiedSparseMerkleProof, protocol.NewSMTSpec())
	if err != nil {
		return nil, fmt.Errorf("failed to compact modified sparse merkle proof: %w", err)
	}

	// Marshal the compacted proof
	modifiedProofBytes, err := modifiedSparseCompactProof.Marshal()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal modified compact proof: %w", err)
	}

	return modifiedProofBytes, nil
}
