package types_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	testproof "github.com/pokt-network/poktroll/testutil/proof"
	"github.com/pokt-network/poktroll/testutil/sample"
	"github.com/pokt-network/poktroll/x/proof/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
)

func TestProofMsgs_RejectNonCanonicalAddresses(t *testing.T) {
	operator, app := sample.AccAddressBech32(), sample.AccAddressBech32()
	upperOperator, upperApp := strings.ToUpper(operator), strings.ToUpper(app)
	rootHash := testproof.RandSmstRootWithSumAndCount(t, 1, 1)
	proof := []byte{1, 2, 3, 4}

	header := func(appAddr string) *sessiontypes.SessionHeader {
		return &sessiontypes.SessionHeader{
			ApplicationAddress:      appAddr,
			ServiceId:               "svcId",
			SessionStartBlockHeight: 100,
			SessionEndBlockHeight:   101,
			SessionId:               "valid_session_id",
		}
	}

	for desc, msg := range map[string]interface{ ValidateBasic() error }{
		"create claim operator":    types.NewMsgCreateClaim(upperOperator, header(app), rootHash),
		"create claim application": types.NewMsgCreateClaim(operator, header(upperApp), rootHash),
		"submit proof operator":    types.NewMsgSubmitProof(upperOperator, header(app), proof),
		"submit proof application": types.NewMsgSubmitProof(operator, header(upperApp), proof),
	} {
		t.Run(desc, func(t *testing.T) {
			err := msg.ValidateBasic()
			require.Error(t, err)
			require.Contains(t, err.Error(), "canonical")
		})
	}
	require.NoError(t, types.NewMsgCreateClaim(operator, header(app), rootHash).ValidateBasic())
	require.NoError(t, types.NewMsgSubmitProof(operator, header(app), proof).ValidateBasic())
}
