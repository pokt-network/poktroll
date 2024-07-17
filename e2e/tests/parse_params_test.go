//go:build e2e

package e2e

import (
	"fmt"
	"strconv"

	"cosmossdk.io/math"
	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

	"github.com/pokt-network/poktroll/app/volatile"
	"github.com/pokt-network/poktroll/proto/types/application"
	"github.com/pokt-network/poktroll/proto/types/proof"
	"github.com/pokt-network/poktroll/proto/types/service"
	"github.com/pokt-network/poktroll/proto/types/shared"
	"github.com/pokt-network/poktroll/proto/types/tokenomics"
	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sharedtypes "github.com/pokt-network/poktroll/x/shared/types"
	tokenomicstypes "github.com/pokt-network/poktroll/x/tokenomics/types"
)

const (
	paramNameColIdx = iota
	paramValueColIdx
	paramTypeColIdx
)

// parseParamsTable parses a gocuke.DataTable into a paramsMap.
func (s *suite) parseParamsTable(table gocuke.DataTable) paramsMap {
	s.Helper()

	paramsMap := make(paramsMap)

	// NB: skip the header row.
	for rowIdx := 1; rowIdx < table.NumRows(); rowIdx++ {
		param := s.parseParam(table, rowIdx)
		paramsMap[param.name] = param
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
	case "bytes":
		paramValue = []byte(table.Cell(rowIdx, paramValueColIdx).String())
	case "float":
		floatValue, err := strconv.ParseFloat(table.Cell(rowIdx, paramValueColIdx).String(), 32)
		require.NoError(s, err)

		paramValue = float32(floatValue)
	case "coin":
		coinAmount := table.Cell(rowIdx, paramValueColIdx).Int64()
		coinValue := cosmostypes.NewCoin(volatile.DenomuPOKT, math.NewInt(coinAmount))
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

// paramsMapToMsgUpdateParams converts a paramsMap into a MsgUpdateParams, which
// it returns as a proto.Message/cosmostypes.Msg interface type.
func (s *suite) paramsMapToMsgUpdateParams(moduleName string, paramsMap paramsMap) (msgUpdateParams cosmostypes.Msg) {
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
	// NB: gateway & supplier modules currently have no parameters
	default:
		err := fmt.Errorf("ERROR: unexpected module name %q", moduleName)
		s.Fatal(err)
		panic(err)
	}

	return msgUpdateParams
}

func (s *suite) newTokenomicsMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &tokenomics.MsgUpdateParams{
		Authority: authority,
		Params:    tokenomics.Params{},
	}

	for paramName, paramValue := range params {
		switch paramName {
		case tokenomics.ParamComputeUnitsToTokensMultiplier:
			msgUpdateParams.Params.ComputeUnitsToTokensMultiplier = uint64(paramValue.value.(int64))
		default:
			s.Fatalf("ERROR: unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newProofMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &proof.MsgUpdateParams{
		Authority: authority,
		Params:    proof.Params{},
	}

	for paramName, paramValue := range params {
		switch paramName {
		case proof.ParamMinRelayDifficultyBits:
			msgUpdateParams.Params.MinRelayDifficultyBits = uint64(paramValue.value.(int64))
		case proof.ParamProofRequestProbability:
			msgUpdateParams.Params.ProofRequestProbability = paramValue.value.(float32)
		case proof.ParamProofRequirementThreshold:
			msgUpdateParams.Params.ProofRequirementThreshold = uint64(paramValue.value.(int64))
		case proof.ParamProofMissingPenalty:
			msgUpdateParams.Params.ProofMissingPenalty = paramValue.value.(*cosmostypes.Coin)
		default:
			s.Fatalf("ERROR: unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newSharedMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &shared.MsgUpdateParams{
		Authority: authority,
		Params:    shared.Params{},
	}

	for paramName, paramValue := range params {
		switch paramName {
		case shared.ParamNumBlocksPerSession:
			msgUpdateParams.Params.NumBlocksPerSession = uint64(paramValue.value.(int64))
		case shared.ParamGracePeriodEndOffsetBlocks:
			msgUpdateParams.Params.GracePeriodEndOffsetBlocks = uint64(paramValue.value.(int64))
		case shared.ParamClaimWindowOpenOffsetBlocks:
			msgUpdateParams.Params.ClaimWindowOpenOffsetBlocks = uint64(paramValue.value.(int64))
		case shared.ParamClaimWindowCloseOffsetBlocks:
			msgUpdateParams.Params.ClaimWindowCloseOffsetBlocks = uint64(paramValue.value.(int64))
		case shared.ParamProofWindowOpenOffsetBlocks:
			msgUpdateParams.Params.ProofWindowOpenOffsetBlocks = uint64(paramValue.value.(int64))
		case shared.ParamProofWindowCloseOffsetBlocks:
			msgUpdateParams.Params.ProofWindowCloseOffsetBlocks = uint64(paramValue.value.(int64))
		default:
			s.Fatalf("ERROR: unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newAppMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &application.MsgUpdateParams{
		Authority: authority,
		Params:    application.Params{},
	}

	for paramName, paramValue := range params {
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case application.ParamMaxDelegatedGateways:
			msgUpdateParams.Params.MaxDelegatedGateways = uint64(paramValue.value.(int64))
		default:
			s.Fatalf("ERROR: unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newServiceMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &service.MsgUpdateParams{
		Authority: authority,
		Params:    service.Params{},
	}

	for paramName, paramValue := range params {
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case service.ParamAddServiceFee:
			msgUpdateParams.Params.AddServiceFee = uint64(paramValue.value.(int64))
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
		msg = proto.Message(&tokenomics.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &tokenomics.MsgUpdateParam_AsString{
				AsString: param.value.(string),
			},
		})
	case "int64":
		msg = proto.Message(&tokenomics.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &tokenomics.MsgUpdateParam_AsInt64{
				AsInt64: param.value.(int64),
			},
		})
	case "bytes":
		msg = proto.Message(&tokenomics.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &tokenomics.MsgUpdateParam_AsBytes{
				AsBytes: param.value.([]byte),
			},
		})
	default:
		s.Fatal("unexpected param type %q for %s module", param.typeStr, tokenomicstypes.ModuleName)
	}

	return msg
}

func (s *suite) newProofMsgUpdateParam(authority string, param paramAny) (msg proto.Message) {
	switch param.typeStr {
	case "string":
		msg = proto.Message(&proof.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &proof.MsgUpdateParam_AsString{
				AsString: param.value.(string),
			},
		})
	case "int64":
		msg = proto.Message(&proof.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &proof.MsgUpdateParam_AsInt64{
				AsInt64: param.value.(int64),
			},
		})
	case "bytes":
		msg = proto.Message(&proof.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &proof.MsgUpdateParam_AsBytes{
				AsBytes: param.value.([]byte),
			},
		})
	case "float":
		msg = proto.Message(&proof.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &proof.MsgUpdateParam_AsFloat{
				AsFloat: param.value.(float32),
			},
		})
	case "coin":
		msg = proto.Message(&proof.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &proof.MsgUpdateParam_AsCoin{
				AsCoin: param.value.(*cosmostypes.Coin),
			},
		})
	default:
		s.Fatal("unexpected param type %q for %s module", param.typeStr, prooftypes.ModuleName)
	}

	return msg
}

func (s *suite) newSharedMsgUpdateParam(authority string, param paramAny) (msg proto.Message) {
	switch param.typeStr {
	case "string":
		msg = proto.Message(&shared.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &shared.MsgUpdateParam_AsString{
				AsString: param.value.(string),
			},
		})
	case "int64":
		msg = proto.Message(&shared.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &shared.MsgUpdateParam_AsInt64{
				AsInt64: param.value.(int64),
			},
		})
	case "bytes":
		msg = proto.Message(&shared.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &shared.MsgUpdateParam_AsBytes{
				AsBytes: param.value.([]byte),
			},
		})
	default:
		s.Fatal("unexpected param type %q for %s module", param.typeStr, sharedtypes.ModuleName)
	}

	return msg
}
