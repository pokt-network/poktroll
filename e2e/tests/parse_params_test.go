//go:build e2e

package e2e

import (
	"fmt"
	"strconv"
	"strings"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/pocket"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	gatewaytypes "github.com/pokt-network/poktroll/x/gateway/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	suppliertypes "github.com/pokt-network/poktroll/x/supplier/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	paramNameColIdx = iota
	paramValueColIdx
	paramTypeColIdx
)

// parseParamsTable parses a gocuke.DataTable into a paramsAnyMap.
func (s *suite) parseParamsTable(table gocuke.DataTable) paramsAnyMap {
	s.Helper()

	paramsMap := make(paramsAnyMap)

	// Track complex parameter fields for aggregation
	complexParams := make(map[string]map[string]float64)

	// NB: skip the header row.
	for rowIdx := 1; rowIdx < table.NumRows(); rowIdx++ {
		param := s.parseParam(table, rowIdx)

		// Check if this is a dotted parameter (e.g., "mint_equals_burn_claim_distribution.dao")
		if strings.Contains(param.name, ".") {
			parts := strings.Split(param.name, ".")
			if len(parts) == 2 {
				complexParamName := parts[0]
				fieldName := parts[1]

				if complexParams[complexParamName] == nil {
					complexParams[complexParamName] = make(map[string]float64)
				}

				// Store the field value
				complexParams[complexParamName][fieldName] = param.value.(float64)
			}
		} else {
			paramsMap[param.name] = param
		}
	}
	// Convert complex parameters to their proper types
	for complexParamName, fields := range complexParams {
		switch complexParamName {
		case "mint_equals_burn_claim_distribution":
			distribution := tokenomicstypes.MintEqualsBurnClaimDistribution{
				Dao:         fields["dao"],
				Proposer:    fields["proposer"],
				Supplier:    fields["supplier"],
				SourceOwner: fields["source_owner"],
				Application: fields["application"],
			}
			paramsMap[tokenomicstypes.ParamMintEqualsBurnClaimDistribution] = paramAny{
				name:    tokenomicstypes.ParamMintEqualsBurnClaimDistribution,
				typeStr: "MintEqualsBurnClaimDistribution",
				value:   distribution,
			}
		case "mint_allocation_percentages":
			allocation := tokenomicstypes.MintAllocationPercentages{
				Dao:         fields["dao"],
				Proposer:    fields["proposer"],
				Supplier:    fields["supplier"],
				SourceOwner: fields["source_owner"],
				Application: fields["application"],
			}
			paramsMap[tokenomicstypes.ParamMintAllocationPercentages] = paramAny{
				name:    tokenomicstypes.ParamMintAllocationPercentages,
				typeStr: "MintAllocationPercentages",
				value:   allocation,
			}
		}
	}

	return paramsMap
}

// parseParam parses a row of a gocuke.DataTable into a paramName and a paramAny.
func (s *suite) parseParam(table gocuke.DataTable, rowIdx int) paramAny {
	s.Helper()

	var paramValue any
	paramName := table.Cell(rowIdx, paramNameColIdx).String()
	paramType := table.Cell(rowIdx, paramTypeColIdx).String()

	switch paramType {
	case "string":
		paramValue = table.Cell(rowIdx, paramValueColIdx).String()
	case "int64":
		paramValue = table.Cell(rowIdx, paramValueColIdx).Int64()
	case "uint64":
		paramValue = uint64(table.Cell(rowIdx, paramValueColIdx).Int64())
	case "bytes":
		paramValue = []byte(table.Cell(rowIdx, paramValueColIdx).String())
	case "float":
		floatValue, err := strconv.ParseFloat(table.Cell(rowIdx, paramValueColIdx).String(), 64)
		require.NoError(s, err)

		paramValue = floatValue
	case "coin":
		coinAmount := table.Cell(rowIdx, paramValueColIdx).Int64()
		coinValue := cosmostypes.NewCoin(pocket.DenomuPOKT, math.NewInt(coinAmount))
		paramValue = &coinValue
	default:
		s.Fatalf("ERROR: unexpected param type %q", paramType)
	}

	return paramAny{
		name:    paramName,
		typeStr: paramType,
		value:   paramValue,
	}
}

// paramsMapToMsgUpdateParams converts a paramsAnyMap into a MsgUpdateParams, which
// it returns as a proto.Message/cosmostypes.Msg interface type.
func (s *suite) paramsMapToMsgUpdateParams(moduleName string, paramsMap paramsAnyMap) (msgUpdateParams cosmostypes.Msg) {
	s.Helper()

	switch moduleName {
	case tokenomicstypes.ModuleName:
		msgUpdateParams = s.newTokenomicsMsgUpdateParams(paramsMap)
	case prooftypes.ModuleName:
		msgUpdateParams = s.newProofMsgUpdateParams(paramsMap)
	case sharedtypes.ModuleName:
		msgUpdateParams = s.newSharedMsgUpdateParams(paramsMap)
	case apptypes.ModuleName:
		msgUpdateParams = s.newAppMsgUpdateParams(paramsMap)
	case servicetypes.ModuleName:
		msgUpdateParams = s.newServiceMsgUpdateParams(paramsMap)
	case suppliertypes.ModuleName:
		msgUpdateParams = s.newSupplierMsgUpdateParams(paramsMap)
	default:
		err := fmt.Errorf("ERROR: unexpected module name %q", moduleName)
		s.Fatal(err)
		panic(err)
	}

	return msgUpdateParams
}

func (s *suite) newTokenomicsMsgUpdateParams(params paramsAnyMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &tokenomicstypes.MsgUpdateParams{
		Authority: authority,
		Params:    tokenomicstypes.DefaultParams(),
	}

	for paramName, paramValue := range params {
		switch paramName {
		case tokenomicstypes.ParamDaoRewardAddress:
			msgUpdateParams.Params.DaoRewardAddress = paramValue.value.(string)
		case tokenomicstypes.ParamMintAllocationPercentages:
			msgUpdateParams.Params.MintAllocationPercentages = paramValue.value.(tokenomicstypes.MintAllocationPercentages)
		case tokenomicstypes.ParamGlobalInflationPerClaim:
			msgUpdateParams.Params.GlobalInflationPerClaim = paramValue.value.(float64)
		case tokenomicstypes.ParamMintEqualsBurnClaimDistribution:
			msgUpdateParams.Params.MintEqualsBurnClaimDistribution = paramValue.value.(tokenomicstypes.MintEqualsBurnClaimDistribution)
		default:
			s.Fatalf("ERROR: unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newProofMsgUpdateParams(params paramsAnyMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &prooftypes.MsgUpdateParams{
		Authority: authority,
		Params:    prooftypes.DefaultParams(),
	}

	for paramName, paramValue := range params {
		switch paramName {
		case prooftypes.ParamProofRequestProbability:
			msgUpdateParams.Params.ProofRequestProbability = paramValue.value.(float64)
		case prooftypes.ParamProofRequirementThreshold:
			msgUpdateParams.Params.ProofRequirementThreshold = paramValue.value.(*cosmostypes.Coin)
		case prooftypes.ParamProofMissingPenalty:
			msgUpdateParams.Params.ProofMissingPenalty = paramValue.value.(*cosmostypes.Coin)
		case prooftypes.ParamProofSubmissionFee:
			msgUpdateParams.Params.ProofSubmissionFee = paramValue.value.(*cosmostypes.Coin)
		default:
			s.Fatalf("ERROR: unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newSharedMsgUpdateParams(params paramsAnyMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &sharedtypes.MsgUpdateParams{
		Authority: authority,
		Params:    sharedtypes.DefaultParams(),
	}

	for paramName, paramValue := range params {
		switch paramName {
		case sharedtypes.ParamNumBlocksPerSession:
			msgUpdateParams.Params.NumBlocksPerSession = uint64(paramValue.value.(int64))
		case sharedtypes.ParamGracePeriodEndOffsetBlocks:
			msgUpdateParams.Params.GracePeriodEndOffsetBlocks = uint64(paramValue.value.(int64))
		case sharedtypes.ParamClaimWindowOpenOffsetBlocks:
			msgUpdateParams.Params.ClaimWindowOpenOffsetBlocks = uint64(paramValue.value.(int64))
		case sharedtypes.ParamClaimWindowCloseOffsetBlocks:
			msgUpdateParams.Params.ClaimWindowCloseOffsetBlocks = uint64(paramValue.value.(int64))
		case sharedtypes.ParamProofWindowOpenOffsetBlocks:
			msgUpdateParams.Params.ProofWindowOpenOffsetBlocks = uint64(paramValue.value.(int64))
		case sharedtypes.ParamProofWindowCloseOffsetBlocks:
			msgUpdateParams.Params.ProofWindowCloseOffsetBlocks = uint64(paramValue.value.(int64))
		case sharedtypes.ParamSupplierUnbondingPeriodSessions:
			msgUpdateParams.Params.SupplierUnbondingPeriodSessions = uint64(paramValue.value.(int64))
		case sharedtypes.ParamApplicationUnbondingPeriodSessions:
			msgUpdateParams.Params.ApplicationUnbondingPeriodSessions = uint64(paramValue.value.(int64))
		case sharedtypes.ParamGatewayUnbondingPeriodSessions:
			msgUpdateParams.Params.GatewayUnbondingPeriodSessions = uint64(paramValue.value.(int64))
		case sharedtypes.ParamComputeUnitsToTokensMultiplier:
			msgUpdateParams.Params.ComputeUnitsToTokensMultiplier = uint64(paramValue.value.(int64))
		case sharedtypes.ParamComputeUnitCostGranularity:
			msgUpdateParams.Params.ComputeUnitCostGranularity = uint64(paramValue.value.(int64))
		default:
			s.Fatalf("ERROR: unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newAppMsgUpdateParams(params paramsAnyMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &apptypes.MsgUpdateParams{
		Authority: authority,
		Params:    apptypes.DefaultParams(),
	}

	for paramName, paramValue := range params {
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case apptypes.ParamMaxDelegatedGateways:
			msgUpdateParams.Params.MaxDelegatedGateways = paramValue.value.(uint64)
		case apptypes.ParamMinStake:
			msgUpdateParams.Params.MinStake = paramValue.value.(*cosmostypes.Coin)
		default:
			s.Fatalf("ERROR: unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newServiceMsgUpdateParams(params paramsAnyMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &servicetypes.MsgUpdateParams{
		Authority: authority,
		Params:    servicetypes.DefaultParams(),
	}

	for paramName, paramValue := range params {
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case servicetypes.ParamAddServiceFee:
			msgUpdateParams.Params.AddServiceFee = paramValue.value.(*cosmostypes.Coin)
		default:
			s.Fatalf("ERROR: unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newSupplierMsgUpdateParams(params map[string]paramAny) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &suppliertypes.MsgUpdateParams{
		Authority: authority,
		Params:    suppliertypes.DefaultParams(),
	}

	for paramName, paramValue := range params {
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case suppliertypes.ParamMinStake:
			msgUpdateParams.Params.MinStake = paramValue.value.(*cosmostypes.Coin)
		case suppliertypes.ParamStakingFee:
			msgUpdateParams.Params.StakingFee = paramValue.value.(*cosmostypes.Coin)
		default:
			s.Fatalf("ERROR: unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

// newMsgUpdateParam returns a MsgUpdateParam for the given module name, param name,
// and param type/value.
func (s *suite) newMsgUpdateParam(
	moduleName string,
	param paramAny,
) (msg cosmostypes.Msg) {
	s.Helper()

	authority := authtypes.NewModuleAddress(s.granterName).String()

	switch moduleName {
	case tokenomicstypes.ModuleName:
		msg = s.newTokenomicsMsgUpdateParam(authority, param)
	case prooftypes.ModuleName:
		msg = s.newProofMsgUpdateParam(authority, param)
	case sharedtypes.ModuleName:
		msg = s.newSharedMsgUpdateParam(authority, param)
	case servicetypes.ModuleName:
		msg = s.newServiceMsgUpdateParam(authority, param)
	case suppliertypes.ModuleName:
		msg = s.newSupplierMsgUpdateParam(authority, param)
	default:
		err := fmt.Errorf("ERROR: unexpected module name %q", moduleName)
		s.Fatal(err)
		panic(err)
	}

	if msg == nil {
		err := fmt.Errorf("ERROR: unexpected param type %q for %q module", param.typeStr, moduleName)
		s.Fatal(err)
		panic(err)
	}

	return msg
}

func (s *suite) newTokenomicsMsgUpdateParam(authority string, param paramAny) (msg proto.Message) {
	switch param.typeStr {
	case "string":
		msg = proto.Message(&tokenomicstypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &tokenomicstypes.MsgUpdateParam_AsString{
				AsString: param.value.(string),
			},
		})
	default:
		s.Fatalf("unexpected param type %q for %s module", param.typeStr, tokenomicstypes.ModuleName)
	}

	return msg
}

func (s *suite) newProofMsgUpdateParam(authority string, param paramAny) (msg proto.Message) {
	switch param.typeStr {
	case "bytes":
		msg = proto.Message(&prooftypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &prooftypes.MsgUpdateParam_AsBytes{
				AsBytes: param.value.([]byte),
			},
		})
	case "float":
		msg = proto.Message(&prooftypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &prooftypes.MsgUpdateParam_AsFloat{
				AsFloat: param.value.(float64),
			},
		})
	case "coin":
		msg = proto.Message(&prooftypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &prooftypes.MsgUpdateParam_AsCoin{
				AsCoin: param.value.(*cosmostypes.Coin),
			},
		})
	default:
		s.Fatalf("unexpected param type %q for %s module", param.typeStr, prooftypes.ModuleName)
	}

	return msg
}

func (s *suite) newSharedMsgUpdateParam(authority string, param paramAny) (msg proto.Message) {
	switch param.typeStr {
	case "string":
		msg = proto.Message(&sharedtypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &sharedtypes.MsgUpdateParam_AsString{
				AsString: param.value.(string),
			},
		})
	case "uint64":
		msg = proto.Message(&sharedtypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &sharedtypes.MsgUpdateParam_AsUint64{
				AsUint64: param.value.(uint64),
			},
		})
	case "bytes":
		msg = proto.Message(&sharedtypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &sharedtypes.MsgUpdateParam_AsBytes{
				AsBytes: param.value.([]byte),
			},
		})
	default:
		s.Fatalf("unexpected param type %q for %s module", param.typeStr, sharedtypes.ModuleName)
	}

	return msg
}

func (s *suite) newServiceMsgUpdateParam(authority string, param paramAny) (msg proto.Message) {
	switch param.typeStr {
	case "coin":
		msg = proto.Message(&servicetypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &servicetypes.MsgUpdateParam_AsCoin{
				AsCoin: param.value.(*cosmostypes.Coin),
			},
		})
	default:
		s.Fatalf("unexpected param type %q for %s module", param.typeStr, tokenomicstypes.ModuleName)
	}

	return msg
}

func (s *suite) newGatewayMsgUpdateParam(authority string, param paramAny) (msg proto.Message) {
	switch param.typeStr {
	case "coin":
		msg = proto.Message(&gatewaytypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &gatewaytypes.MsgUpdateParam_AsCoin{
				AsCoin: param.value.(*cosmostypes.Coin),
			},
		})
	default:
		s.Fatalf("unexpected param type %q for %s module", param.typeStr, tokenomicstypes.ModuleName)
	}

	return msg
}

func (s *suite) newSupplierMsgUpdateParam(authority string, param paramAny) (msg proto.Message) {
	switch param.typeStr {
	case "coin":
		msg = proto.Message(&suppliertypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &suppliertypes.MsgUpdateParam_AsCoin{
				AsCoin: param.value.(*cosmostypes.Coin),
			},
		})
	default:
		s.Fatalf("unexpected param type %q for %s module", param.typeStr, suppliertypes.ModuleName)
	}

	return msg
}
