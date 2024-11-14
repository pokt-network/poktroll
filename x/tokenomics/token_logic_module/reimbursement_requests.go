package token_logic_module

import (
	"context"
	"fmt"

	cosmoslog "cosmossdk.io/log"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"

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
	result := tlmCtx.Result
	service := tlmCtx.Service
	sessionHeader := tlmCtx.SessionHeader
	application := tlmCtx.Application
	supplier := tlmCtx.Supplier
	actualSettlementCoin := tlmCtx.SettlementCoin

	logger = logger.With("method", "TokenLogicModuleGlobalMintReimbursementRequest")

	// Do not process the reimbursement request if there is no global inflation.
	if GlobalInflationPerClaim == 0 {
		logger.Warn("global inflation is set to zero. Skipping Global Mint Reimbursement Request TLM.")
		return nil
	}

	// Determine how much new uPOKT to mint based on global inflation
	newMintCoin, _ := CalculateGlobalPerClaimMintInflationFromSettlementAmount(actualSettlementCoin)
	if newMintCoin.Amount.Int64() == 0 {
		return tokenomicstypes.ErrTokenomicsMintAmountZero
	}

	newAppStake, err := application.Stake.SafeSub(newMintCoin)
	// This should THEORETICALLY NEVER fall below zero.
	// `ensureClaimAmountLimits` should have already checked and adjusted the settlement
	// amount so that the application stake covers the global inflation.
	// TODO_POST_MAINNET: Consider removing this since it should never happen just to simplify the code
	if err != nil {
		return err
	}
	application.Stake = &newAppStake
	logger.Info(fmt.Sprintf("updated application %q stake to %s", application.Address, newAppStake))

	// Send the global per claim mint inflation uPOKT from the tokenomics module
	// account to PNF/DAO.
	daoRewardAddress := tlmCtx.Params.Tokenomics.GetDaoRewardAddress()
	daoAccountAddr, err := cosmostypes.AccAddressFromBech32(daoRewardAddress)
	if err != nil {
		return tokenomicstypes.ErrTokenomicsApplicationReimbursementRequestFailed.Wrapf(
			"getting PNF/DAO address: %v",
			err,
		)
	}

	// Send the global per claim mint inflation uPOKT from the application module
	// account to the tokenomics module account as an intermediary step.
	result.AppendModToModTransfer(ModToModTransfer{
		OriginTLM:       TLMGlobalMintReimbursementRequest,
		OriginReason:    TLMGlobalMintReimbursementRequest_ReimbursementEscrowModuleTransfer,
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
	result.AppendModToAcctTransfer(ModToAcctTransfer{
		OriginTLM:        TLMGlobalMintReimbursementRequest,
		OriginReason:     TLMGlobalMintReimbursementRequest_DaoReimbursementEscrow,
		SenderModule:     tokenomicstypes.ModuleName,
		RecipientAddress: daoAccountAddr,
		Coin:             newMintCoin,
	})
	logger.Info(fmt.Sprintf(
		"operation queued: send (%s) from the tokenomics module account to the PNF/DAO account (%s)",
		newMintCoin, daoAccountAddr.String(),
	))

	// Prepare and emit the event for the application that'll required reimbursement.
	// Recall that it is being overcharged to compoensate for global inflation while
	// preventing self-dealing attacks.
	reimbursementRequestEvent := &tokenomicstypes.EventApplicationReimbursementRequest{
		ApplicationAddr:      application.Address,
		SupplierOperatorAddr: supplier.OperatorAddress,
		SupplierOwnerAddr:    supplier.OwnerAddress,
		ServiceId:            service.Id,
		SessionId:            sessionHeader.SessionId,
		Amount:               &newMintCoin,
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
