package proof

import (
	"encoding/binary"

	"github.com/pokt-network/smt"

	testsession "github.com/pokt-network/poktroll/testutil/session"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
)

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

func SmstRootWithSum(sum uint64) smt.MerkleRoot {
	root := make([]byte, 40)
	copy(root[:32], []byte("This is exactly 32 characters!!!"))
	binary.BigEndian.PutUint64(root[32:], sum)
	return smt.MerkleRoot(root)
}
