package suites

import (
	"encoding/hex"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"

	"github.com/pokt-network/poktroll/app/volatile"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

var (
	ValidAddServiceFeeCoin             = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000001)
	ValidProofMissingPenaltyCoin       = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 500)
	ValidProofSubmissionFeeCoin        = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 5000000)
	ValidProofRequirementThresholdCoin = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)
	ValidRelayDifficultyTargetHash, _  = hex.DecodeString("00000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffff")

	SharedModuleParamConfig = ModuleParamConfig{
		ParamsMsgs: ModuleParamsMessages{
			MsgUpdateParams:         sharedtypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: sharedtypes.MsgUpdateParamsResponse{},
			MsgUpdateParam:          sharedtypes.MsgUpdateParam{},
			MsgUpdateParamResponse:  sharedtypes.MsgUpdateParamResponse{},
			QueryParamsRequest:      sharedtypes.QueryParamsRequest{},
			QueryParamsResponse:     sharedtypes.QueryParamsResponse{},
		},
		ParamTypes: map[ParamType]any{
			ParamTypeUint64: sharedtypes.MsgUpdateParam_AsInt64{},
			ParamTypeInt64:  sharedtypes.MsgUpdateParam_AsInt64{},
			ParamTypeString: sharedtypes.MsgUpdateParam_AsString{},
			ParamTypeBytes:  sharedtypes.MsgUpdateParam_AsBytes{},
		},
		ValidParams: sharedtypes.Params{
			NumBlocksPerSession:                12,
			GracePeriodEndOffsetBlocks:         0,
			ClaimWindowOpenOffsetBlocks:        2,
			ClaimWindowCloseOffsetBlocks:       3,
			ProofWindowOpenOffsetBlocks:        1,
			ProofWindowCloseOffsetBlocks:       3,
			SupplierUnbondingPeriodSessions:    9,
			ApplicationUnbondingPeriodSessions: 9,
			ComputeUnitsToTokensMultiplier:     420,
		},
		DefaultParams:    sharedtypes.DefaultParams(),
		NewParamClientFn: sharedtypes.NewQueryClient,
	}

	SessionModuleParamConfig = ModuleParamConfig{
		ParamsMsgs: ModuleParamsMessages{
			MsgUpdateParams:         sessiontypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: sessiontypes.MsgUpdateParamsResponse{},
			QueryParamsRequest:      sessiontypes.QueryParamsRequest{},
			QueryParamsResponse:     sessiontypes.QueryParamsResponse{},
		},
		ValidParams:      sessiontypes.Params{},
		DefaultParams:    sessiontypes.DefaultParams(),
		NewParamClientFn: sessiontypes.NewQueryClient,
	}

	ServiceModuleParamConfig = ModuleParamConfig{
		ParamsMsgs: ModuleParamsMessages{
			MsgUpdateParams:         servicetypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: servicetypes.MsgUpdateParamsResponse{},
			MsgUpdateParam:          servicetypes.MsgUpdateParam{},
			MsgUpdateParamResponse:  servicetypes.MsgUpdateParamResponse{},
			QueryParamsRequest:      servicetypes.QueryParamsRequest{},
			QueryParamsResponse:     servicetypes.QueryParamsResponse{},
		},
		ValidParams: servicetypes.Params{
			AddServiceFee: &ValidAddServiceFeeCoin,
		},
		ParamTypes: map[ParamType]any{
			ParamTypeCoin: servicetypes.MsgUpdateParam_AsCoin{},
		},
		DefaultParams:    servicetypes.DefaultParams(),
		NewParamClientFn: servicetypes.NewQueryClient,
	}

	ApplicationModuleParamConfig = ModuleParamConfig{
		ParamsMsgs: ModuleParamsMessages{
			MsgUpdateParams:         apptypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: apptypes.MsgUpdateParamsResponse{},
			QueryParamsRequest:      apptypes.QueryParamsRequest{},
			QueryParamsResponse:     apptypes.QueryParamsResponse{},
		},
		ValidParams: apptypes.Params{
			MaxDelegatedGateways: 999,
		},
		DefaultParams:    apptypes.DefaultParams(),
		NewParamClientFn: apptypes.NewQueryClient,
	}

	GatewayModuleParamConfig = ModuleParamConfig{
		ParamsMsgs: ModuleParamsMessages{
			MsgUpdateParams:         gatewaytypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: gatewaytypes.MsgUpdateParamsResponse{},
			QueryParamsRequest:      gatewaytypes.QueryParamsRequest{},
			QueryParamsResponse:     gatewaytypes.QueryParamsResponse{},
		},
		ValidParams:      gatewaytypes.Params{},
		DefaultParams:    gatewaytypes.DefaultParams(),
		NewParamClientFn: gatewaytypes.NewQueryClient,
	}

	SupplierModuleParamConfig = ModuleParamConfig{
		ParamsMsgs: ModuleParamsMessages{
			MsgUpdateParams:         suppliertypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: suppliertypes.MsgUpdateParamsResponse{},
			QueryParamsRequest:      suppliertypes.QueryParamsRequest{},
			QueryParamsResponse:     suppliertypes.QueryParamsResponse{},
		},
		ValidParams:      suppliertypes.Params{},
		DefaultParams:    suppliertypes.DefaultParams(),
		NewParamClientFn: suppliertypes.NewQueryClient,
	}

	ProofModuleParamConfig = ModuleParamConfig{
		ParamsMsgs: ModuleParamsMessages{
			MsgUpdateParams:         prooftypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: prooftypes.MsgUpdateParamsResponse{},
			MsgUpdateParam:          prooftypes.MsgUpdateParam{},
			MsgUpdateParamResponse:  prooftypes.MsgUpdateParamResponse{},
			QueryParamsRequest:      prooftypes.QueryParamsRequest{},
			QueryParamsResponse:     prooftypes.QueryParamsResponse{},
		},
		ValidParams: prooftypes.Params{
			RelayDifficultyTargetHash: ValidRelayDifficultyTargetHash,
			ProofRequestProbability:   0.1,
			ProofRequirementThreshold: &ValidProofRequirementThresholdCoin,
			ProofMissingPenalty:       &ValidProofMissingPenaltyCoin,
			ProofSubmissionFee:        &ValidProofSubmissionFeeCoin,
		},
		ParamTypes: map[ParamType]any{
			ParamTypeUint64:  prooftypes.MsgUpdateParam_AsInt64{},
			ParamTypeInt64:   prooftypes.MsgUpdateParam_AsInt64{},
			ParamTypeString:  prooftypes.MsgUpdateParam_AsString{},
			ParamTypeBytes:   prooftypes.MsgUpdateParam_AsBytes{},
			ParamTypeFloat32: prooftypes.MsgUpdateParam_AsFloat{},
			ParamTypeCoin:    prooftypes.MsgUpdateParam_AsCoin{},
		},
		DefaultParams:    prooftypes.DefaultParams(),
		NewParamClientFn: prooftypes.NewQueryClient,
	}

	TokenomicsModuleParamConfig = ModuleParamConfig{
		ParamsMsgs: ModuleParamsMessages{
			MsgUpdateParams:         tokenomicstypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: tokenomicstypes.MsgUpdateParamsResponse{},
			QueryParamsRequest:      tokenomicstypes.QueryParamsRequest{},
			QueryParamsResponse:     tokenomicstypes.QueryParamsResponse{},
		},
		ValidParams:      tokenomicstypes.Params{},
		DefaultParams:    tokenomicstypes.DefaultParams(),
		NewParamClientFn: tokenomicstypes.NewQueryClient,
	}
)
