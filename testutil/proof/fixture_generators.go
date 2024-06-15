package proof

import (
	"encoding/binary"
	"fmt"

	"github.com/pokt-network/smt"

	testsession "github.com/pokt-network/poktroll/testutil/session"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

// BaseClaim returns a base (default, example, etc..) claim with the given app
// address, supplier address, and sum that can be used for testing.
func BaseClaim(appAddr, supplierAddr string, sum uint64) prooftypes.Claim {
	return prooftypes.Claim{
		SupplierAddress: supplierAddr,
		SessionHeader: &sessiontypes.SessionHeader{
			ApplicationAddress: appAddr,
			Service: &sharedtypes.Service{
				Id:   "svc1",
				Name: "svcName1",
			},
			SessionId:               "session_id",
			SessionStartBlockHeight: 1,
			SessionEndBlockHeight:   testsession.GetSessionEndHeightWithDefaultParams(1),
		},
		RootHash: SmstRootWithSum(sum),
	}
}

// SmstRootWithSum returns a SMST root with the given sum and a default
// hard-coded count of 1.
// TODO_POTENTIAL_TECHDEBT: Note that the count is meant to represent the number
// of non-empty leaves in the tree, and may need become a parameter depending on
// how the tests evolve.
func SmstRootWithSum(sum uint64) smt.MerkleRoot {
	root := make([]byte, smt.SmstRootSizeBytes)

	// Insert a string into the root hash
	copy(root[:smt.SmtRootSizeBytes], []byte(fmt.Sprintf("This is exactly %d characters!!!", smt.SmtRootSizeBytes)))

	// Insert the sum into the root hash
	binary.BigEndian.PutUint64(root[smt.SmtRootSizeBytes:], sum)

	// Insert the count into the root hash
	// TODO_TECHDEBT: This is a hard-coded count of 1, but could be a parameter.
	// TODO_TECHDEBT: We are assuming the sum takes up 8 bytes.
	binary.BigEndian.PutUint64(root[smt.SmtRootSizeBytes+8:], 1)

	return smt.MerkleRoot(root)
}
