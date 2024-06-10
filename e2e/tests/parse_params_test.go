//go:build e2e

package e2e

import (
	"fmt"
	"strconv"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/regen-network/gocuke"
	"github.com/stretchr/testify/require"

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
	default:
		s.Fatalf("unexpected param type %q", paramType)
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

	msgUpdateParams := &tokenomicstypes.MsgUpdateParams{
		Authority: authority,
		Params:    tokenomicstypes.Params{},
	}

	for paramName, paramValue := range params {
		switch paramName {
		case tokenomicstypes.ParamComputeUnitsToTokensMultiplier:
			msgUpdateParams.Params.ComputeUnitsToTokensMultiplier = uint64(paramValue.value.(int64))
		default:
			s.Fatalf("unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newProofMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &prooftypes.MsgUpdateParams{
		Authority: authority,
		Params:    prooftypes.Params{},
	}

	for paramName, paramValue := range params {
		switch paramName {
		case prooftypes.ParamMinRelayDifficultyBits:
			msgUpdateParams.Params.MinRelayDifficultyBits = uint64(paramValue.value.(int64))
		case prooftypes.ParamProofRequestProbability:
			msgUpdateParams.Params.ProofRequestProbability = paramValue.value.(float32)
		default:
			s.Fatalf("unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newSharedMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &sharedtypes.MsgUpdateParams{
		Authority: authority,
		Params:    sharedtypes.Params{},
	}

	for paramName, paramValue := range params {
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case sharedtypes.ParamNumBlocksPerSession:
			msgUpdateParams.Params.NumBlocksPerSession = uint64(paramValue.value.(int64))
		case sharedtypes.ParamClaimWindowOpenOffsetBlocks:
			msgUpdateParams.Params.ClaimWindowOpenOffsetBlocks = uint64(paramValue.value.(int64))
		case sharedtypes.ParamClaimWindowCloseOffsetBlocks:
			msgUpdateParams.Params.ClaimWindowCloseOffsetBlocks = uint64(paramValue.value.(int64))
		case sharedtypes.ParamProofWindowOpenOffsetBlocks:
			msgUpdateParams.Params.ProofWindowOpenOffsetBlocks = uint64(paramValue.value.(int64))
		case sharedtypes.ParamProofWindowCloseOffsetBlocks:
			msgUpdateParams.Params.ProofWindowCloseOffsetBlocks = uint64(paramValue.value.(int64))
		default:
			s.Fatalf("unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newAppMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &apptypes.MsgUpdateParams{
		Authority: authority,
		Params:    apptypes.Params{},
	}

	for paramName, paramValue := range params {
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case apptypes.ParamMaxDelegatedGateways:
			msgUpdateParams.Params.MaxDelegatedGateways = uint64(paramValue.value.(int64))
		default:
			s.Fatalf("unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newServiceMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &servicetypes.MsgUpdateParams{
		Authority: authority,
		Params:    servicetypes.Params{},
	}

	for paramName, paramValue := range params {
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case servicetypes.ParamAddServiceFee:
			msgUpdateParams.Params.AddServiceFee = uint64(paramValue.value.(int64))
		default:
			s.Fatalf("unexpected %q type param name %q", paramValue.typeStr, paramName)
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
		msg = proto.Message(&tokenomicstypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &tokenomicstypes.MsgUpdateParam_AsString{
				AsString: param.value.(string),
			},
		})
	case "int64":
		msg = proto.Message(&tokenomicstypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &tokenomicstypes.MsgUpdateParam_AsInt64{
				AsInt64: param.value.(int64),
			},
		})
	case "bytes":
		msg = proto.Message(&tokenomicstypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &tokenomicstypes.MsgUpdateParam_AsBytes{
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
		msg = proto.Message(&prooftypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &prooftypes.MsgUpdateParam_AsString{
				AsString: param.value.(string),
			},
		})
	case "int64":
		msg = proto.Message(&prooftypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &prooftypes.MsgUpdateParam_AsInt64{
				AsInt64: param.value.(int64),
			},
		})
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
				AsFloat: param.value.(float32),
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
		msg = proto.Message(&sharedtypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &sharedtypes.MsgUpdateParam_AsString{
				AsString: param.value.(string),
			},
		})
	case "int64":
		msg = proto.Message(&sharedtypes.MsgUpdateParam{
			Authority: authority,
			Name:      param.name,
			AsType: &sharedtypes.MsgUpdateParam_AsInt64{
				AsInt64: param.value.(int64),
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
		s.Fatal("unexpected param type %q for %s module", param.typeStr, sharedtypes.ModuleName)
	}

	return msg
}
