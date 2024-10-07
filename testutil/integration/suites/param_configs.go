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

// ParamType is a type alias for a module parameter type. It is the string that
// is returned when calling reflect.Value#Type()#Name() on a module parameter.
type ParamType = string

const (
	ParamTypeInt64   ParamType = "int64"
	ParamTypeUint64  ParamType = "uint64"
	ParamTypeFloat32 ParamType = "float32"
	ParamTypeString  ParamType = "string"
	ParamTypeBytes   ParamType = "uint8"
	ParamTypeCoin    ParamType = "Coin"
)

// ModuleParamConfig holds type information about a module's parameters update
// message(s) along with default and valid non-default values and a query constructor
// function for the module. It is used by ParamsSuite to construct and send
// parameter update messages and assert on their results.
type ModuleParamConfig struct {
	ParamsMsgs ModuleParamsMessages
	// ParamTypes is a map of parameter types to their respective MsgUpdateParam_As*
	// types which satisfy the oneof for the MsgUpdateParam#AsType field. Each AsType
	// type which the module supports should be included in this map.
	ParamTypes map[ParamType]any
	// ValidParams is a set of parameters which are expected to be valid when used
	// together AND when used individually, where the reamining parameters are set
	// to their default values.
	ValidParams      any
	DefaultParams    any
	NewParamClientFn any
}

// ModuleParamsMessages holds a reference to each of the params-related message
// types for a given module. The values are only used for their type information
// which is obtained via reflection. The values are not used for their actual
// message contents and MAY be the zero value.
// If MsgUpdateParam is omitted (i.e. nil), ParamsSuite will assume that
// this module does not support individual parameter updates (i.e. MsgUpdateParam).
// In this case, MsgUpdateParamResponse SHOULD also be omitted.
type ModuleParamsMessages struct {
	MsgUpdateParams         any
	MsgUpdateParamsResponse any
	MsgUpdateParam          any
	MsgUpdateParamResponse  any
	QueryParamsRequest      any
	QueryParamsResponse     any
}

var (
	ValidAddServiceFeeCoin             = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 1000000001)
	ValidProofMissingPenaltyCoin       = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 500)
	ValidProofSubmissionFeeCoin        = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 5000000)
	ValidProofRequirementThresholdCoin = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)
	ValidRelayDifficultyTargetHash, _  = hex.DecodeString("00000000ffffffffffffffffffffffffffffffffffffffffffffffffffffffff")
	ValidActorMinStake                 = cosmostypes.NewInt64Coin(volatile.DenomuPOKT, 100)

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
			// TODO_IMPROVE: Add a Uint64 asType instead of using int64 for uint64 params.
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
			MsgUpdateParam:          apptypes.MsgUpdateParam{},
			MsgUpdateParamResponse:  apptypes.MsgUpdateParamResponse{},
			QueryParamsRequest:      apptypes.QueryParamsRequest{},
			QueryParamsResponse:     apptypes.QueryParamsResponse{},
		},
		ValidParams: apptypes.Params{
			MaxDelegatedGateways: 999,
			MinStake:             &ValidActorMinStake,
		},
		ParamTypes: map[ParamType]any{
			// TODO_IMPROVE: Add a Uint64 asType instead of using int64 for uint64 params.
			ParamTypeUint64: apptypes.MsgUpdateParam_AsInt64{},
			ParamTypeCoin:   apptypes.MsgUpdateParam_AsCoin{},
		},
		DefaultParams:    apptypes.DefaultParams(),
		NewParamClientFn: apptypes.NewQueryClient,
	}

	GatewayModuleParamConfig = ModuleParamConfig{
		ParamsMsgs: ModuleParamsMessages{
			MsgUpdateParams:         gatewaytypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: gatewaytypes.MsgUpdateParamsResponse{},
			MsgUpdateParam:          gatewaytypes.MsgUpdateParam{},
			MsgUpdateParamResponse:  gatewaytypes.MsgUpdateParamResponse{},
			QueryParamsRequest:      gatewaytypes.QueryParamsRequest{},
			QueryParamsResponse:     gatewaytypes.QueryParamsResponse{},
		},
		ValidParams: gatewaytypes.Params{
			MinStake: &ValidActorMinStake,
		},
		ParamTypes: map[ParamType]any{
			ParamTypeCoin: gatewaytypes.MsgUpdateParam_AsCoin{},
		},
		DefaultParams:    gatewaytypes.DefaultParams(),
		NewParamClientFn: gatewaytypes.NewQueryClient,
	}

	SupplierModuleParamConfig = ModuleParamConfig{
		ParamsMsgs: ModuleParamsMessages{
			MsgUpdateParams:         suppliertypes.MsgUpdateParams{},
			MsgUpdateParamsResponse: suppliertypes.MsgUpdateParamsResponse{},
			MsgUpdateParam:          suppliertypes.MsgUpdateParam{},
			MsgUpdateParamResponse:  suppliertypes.MsgUpdateParamResponse{},
			QueryParamsRequest:      suppliertypes.QueryParamsRequest{},
			QueryParamsResponse:     suppliertypes.QueryParamsResponse{},
		},
		ValidParams: suppliertypes.Params{
			MinStake: &ValidActorMinStake,
		},
		ParamTypes: map[ParamType]any{
			ParamTypeCoin: suppliertypes.MsgUpdateParam_AsCoin{},
		},
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
