package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/pokt-network/smt"

	"github.com/pokt-network/poktroll/pkg/crypto/protocol"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

const TypeMsgCreateClaim = "create_claim"

var _ sdk.Msg = (*MsgCreateClaim)(nil)

func NewMsgCreateClaim(
	supplierOperatorAddr string,
	sessionHeader *sessiontypes.SessionHeader,
	rootHash []byte,
) *MsgCreateClaim {
	return &MsgCreateClaim{
		SupplierOperatorAddress: supplierOperatorAddr,
		SessionHeader:           sessionHeader,
		RootHash:                rootHash,
	}
}

// ValidateBasic performs basic stateless validation of a MsgCreateClaim.
func (msg *MsgCreateClaim) ValidateBasic() error {
	// Validate the supplier operator address
	if _, err := sdk.AccAddressFromBech32(msg.GetSupplierOperatorAddress()); err != nil {
		return ErrProofInvalidAddress.Wrapf("%s", msg.GetSupplierOperatorAddress())
	}

	// Retrieve & validate the session header
	sessionHeader := msg.SessionHeader
	if sessionHeader == nil {
		return ErrProofInvalidSessionHeader.Wrapf("session header is nil")
	}
	if err := sessionHeader.ValidateBasic(); err != nil {
		return ErrProofInvalidSessionHeader.Wrapf("invalid session header: %v", err)
	}

	// Validate the root hash
	if len(msg.RootHash) != protocol.TrieRootSize {
		return ErrProofInvalidClaimRootHash.Wrapf("expecting root hash to be %d bytes, got %d bytes", protocol.TrieRootSumSize, len(msg.RootHash))
	}

	// Get the Merkle root from the root hash to validate the claim's relays and compute units.
	merkleRoot := smt.MerkleSumRoot(msg.RootHash)

	count, err := merkleRoot.Count()
	if err != nil {
		return ErrProofInvalidClaimRootHash.Wrapf("error getting Merkle root %v count due to: %v", msg.RootHash, err)
	}

	if count == 0 {
		return ErrProofInvalidClaimRootHash.Wrapf("has zero count in Merkle root %v", msg.RootHash)
	}

	sum, err := merkleRoot.Sum()
	if err != nil {
		return ErrProofInvalidClaimRootHash.Wrapf("error getting Merkle root %v sum due to: %v", msg.RootHash, err)
	}

	if sum == 0 {
		return ErrProofInvalidClaimRootHash.Wrapf("has zero sum in Merkle root %v", msg.RootHash)
	}

	return nil
}
