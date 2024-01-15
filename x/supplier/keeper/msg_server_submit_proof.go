package keeper

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/x/supplier/types"
)

func (k msgServer) SubmitProof(
	goCtx context.Context,
	msg *types.MsgSubmitProof,
) (*types.MsgSubmitProofResponse, error) {
	// TODO_BLOCKER: Prevent Proof upserts after the tokenomics module has processes the respective session.
	// TODO_BLOCKER: Validate the signature on the Proof message corresponds to the supplier before Upserting.

	ctx := sdk.UnwrapSDKContext(goCtx)

	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	/*
		INCOMPLETE: Handling the message

		## Validation

		### Session validation
		1. [ ] claimed session ID == retrieved session ID
		2. [ ] this supplier is in the session's suppliers list
		3. [ ] proof signer addr == session application addr

		### Msg distribution validation (depends on session validation)
		1. [ ] pseudo-randomize earliest block offset
		2. [ ] governance-based earliest block offset

		### Proof validation
		1. [ ] session validation
		2. [ ] msg distribution validation
		3. [ ] claim with matching session ID exists
		4. [ ] proof path matches last committed block hash at claim height - 1
		5. [ ] proof validates with claimed root hash

		## Persistence
		1. [ ] submit proof message
			- supplier address
			- session header
			- proof

		## Accounting
		1. [ ] extract work done from root hash
		2. [ ] calculate reward/burn token with governance-based multiplier
		3. [ ] reward supplier
		4. [ ] burn application tokens
	*/

	_ = ctx

	return &types.MsgSubmitProofResponse{}, nil
}
