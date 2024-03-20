package keeper

import (
	"context"

	"cosmossdk.io/core/appmodule"
)

// SettlePendingClaims settles all pending claims.
func (k Keeper) SettlePendingClaims(ctx context.Context, env appmodule.Environment) error {
	// endTime := env.HeaderService.GetHeaderInfo(ctx).Time.Add(-k.config.MaxExecutionPeriod)
	// proposals, err := k.proposalsByVPEnd(ctx, endTime)
	// if err != nil {
	// 	return nil
	// }
	// for _, proposal := range proposals {
	// 	proposal := proposal

	// 	err := k.pruneProposal(ctx, proposal.Id)
	// 	if err != nil {
	// 		return err
	// 	}
	// 	// Emit event for proposal finalized with its result
	// 	if err := k.environment.EventService.EventManager(ctx).Emit(
	// 		&group.EventProposalPruned{
	// 			ProposalId:  proposal.Id,
	// 			Status:      proposal.Status,
	// 			TallyResult: &proposal.FinalTallyResult,
	// 		},
	// 	); err != nil {
	// 		return err
	// 	}
	// }

	return nil

	// logger := am.tokenomicsKeeper.Logger().With("EndBlock", "TokenomicsModuleEndBlock")

	// ctx := sdk.UnwrapSDKContext(goCtx)
	// blockHeight := ctx.BlockHeight()

	// // TODO_BLOCKER(@Olshansk): Optimize this by indexing claims appropriately
	// // and only retrieving the claims that need to be settled rather than all
	// // of them and iterating through them one by one.
	// claims := am.proofKeeper.GetAllClaims(goCtx)
	// numClaimsSettled := 0
	// for _, claim := range claims {
	// 	// TODO_IN_THIS_PR: Discuss with @red-0ne if we need to account for
	// 	// the grace period here
	// 	if claim.SessionHeader.SessionEndBlockHeight == blockHeight {
	// 		if err := am.tokenomicsKeeper.SettleSessionAccounting(ctx, &claim); err != nil {
	// 			logger.Error("error settling session accounting", "error", err, "claim", claim)
	// 			return err
	// 		}
	// 		numClaimsSettled++
	// 		logger.Info(fmt.Sprintf("settled claim %s at block height %d", claim.SessionHeader.SessionId, blockHeight))
	// 	}
	// }

	// logger.Info(fmt.Sprintf("settled %d claims at block height %d", numClaimsSettled, blockHeight))
}
