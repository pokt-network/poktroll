package token_logic_module

import (
	"context"
	"fmt"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/pkg/encoding"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

var _ TokenLogicModule = (*tlmGlobalMintReimbursementRequest)(nil)

type tlmGlobalMintReimbursementRequest struct{}

// NewGlobalMintReimbursementRequestTLM returns a new GlobalMintReimbursementRequest TLM.
func NewGlobalMintReimbursementRequestTLM() TokenLogicModule {
	return &tlmGlobalMintReimbursementRequest{}
}

func (tlm tlmGlobalMintReimbursementRequest) GetId() TokenLogicModuleId {
	return TLMGlobalMintReimbursementRequest
}

// Process processes the business logic for the GlobalMintReimbursementRequest TLM.
func (tlm tlmGlobalMintReimbursementRequest) Process(
	ctx context.Context,
	logger cosmoslog.Logger,
	tlmCtx TLMContext,
) error {
	blockHeight := cosmostypes.UnwrapSDKContext(ctx).BlockHeight()
	result := tlmCtx.Result
	service := tlmCtx.Service
	sessionHeader := tlmCtx.SessionHeader
	application := tlmCtx.Application
	supplier := tlmCtx.Supplier
	actualSettlementCoin := tlmCtx.SettlementCoin

	logger = logger.With(
		"tlm", "TokenLogicModuleGlobalMintReimbursementRequest",
		"method", "Process",
		"height", blockHeight,
		"session_id", sessionHeader.GetSessionId(),
		"service_id", service.Id,
		"application", application.Address,
		"supplier_operator", supplier.OperatorAddress,
		"actual_settlement_coin", actualSettlementCoin,
	)

	globalInflationPerClaim := tlmCtx.TokenomicsParams.GetGlobalInflationPerClaim()
	globalInflationPerClaimRat, err := encoding.Float64ToRat(globalInflationPerClaim)
	if err != nil {
		logger.Error(fmt.Sprintf("error processing TLM due to: %v", err))
		return err
	}

	// Do not process the reimbursement request if there is no global inflation.
	if globalInflationPerClaim == 0 {
		logger.Warn("global inflation is set to zero. Skipping Global Mint Reimbursement Request TLM.")
		return nil
	}

	// Determine how much new uPOKT to mint based on global inflation
	newMintCoin := CalculateGlobalPerClaimMintInflationFromSettlementAmount(actualSettlementCoin, globalInflationPerClaimRat)
	if newMintCoin.Amount.Int64() == 0 {
		return tokenomicstypes.ErrTokenomicsCoinIsZero
	}

	newAppStake, err := application.Stake.SafeSub(newMintCoin)
	// This should THEORETICALLY NEVER happen because `ensureClaimAmountLimits` should have handled it.
	if err != nil {
		logger.Error(fmt.Sprintf("SHOULD NEVER HAPPEN: application stake should never fall below zero. Trying to subtract %s from %s causing error: %v", newMintCoin, application.Stake, err))
		return err
	}
	application.Stake = &newAppStake
	logger.Info(fmt.Sprintf("updated application %q stake to %s", application.Address, newAppStake))

	// Send the global per claim mint inflation uPOKT from the tokenomics module
	// account to PNF/DAO.
	daoRewardAddress := tlmCtx.TokenomicsParams.GetDaoRewardAddress()

	// Send the global per claim mint inflation uPOKT from the application module
	// account to the tokenomics module account as an intermediary step.
	result.AppendModToModTransfer(tokenomicstypes.ModToModTransfer{
		OpReason:        tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_REIMBURSEMENT_REQUEST_ESCROW_MODULE_TRANSFER,
		SenderModule:    apptypes.ModuleName,
		RecipientModule: tokenomicstypes.ModuleName,
		Coin:            newMintCoin,
	})
	logger.Info(fmt.Sprintf(
		"operation queued: send (%s) from the application module account to the tokenomics module account",
		newMintCoin,
	))

	// Send the global per claim mint inflation uPOKT from the tokenomics module
	// for second order economic effects.
	// See: https://discord.com/channels/824324475256438814/997192534168182905/1299372745632649408
	result.AppendModToAcctTransfer(tokenomicstypes.ModToAcctTransfer{
		OpReason:         tokenomicstypes.SettlementOpReason_TLM_GLOBAL_MINT_REIMBURSEMENT_REQUEST_ESCROW_DAO_TRANSFER,
		SenderModule:     tokenomicstypes.ModuleName,
		RecipientAddress: daoRewardAddress,
		Coin:             newMintCoin,
	})
	logger.Info(fmt.Sprintf(
		"operation queued: send (%s) from the tokenomics module account to the PNF/DAO account (%s)",
		newMintCoin, daoRewardAddress,
	))

	// Prepare and emit the event for the application that'll required reimbursement.
	// Recall that it is being overcharged to compensate for global inflation while
	// preventing self-dealing attacks.
	reimbursementRequestEvent := &tokenomicstypes.EventApplicationReimbursementRequest{
		ApplicationAddr:      application.Address,
		SupplierOperatorAddr: supplier.OperatorAddress,
		SupplierOwnerAddr:    supplier.OwnerAddress,
		ServiceId:            service.Id,
		SessionId:            sessionHeader.SessionId,
		Amount:               newMintCoin.String(),
	}

	eventManger := cosmostypes.UnwrapSDKContext(ctx).EventManager()
	if err = eventManger.EmitTypedEvent(reimbursementRequestEvent); err != nil {
		err = tokenomicstypes.ErrTokenomicsEmittingEventFailed.Wrapf(
			"(%+v): %s",
			reimbursementRequestEvent, err,
		)

		logger.Error(err.Error())
		return err
	}

	return nil
}
