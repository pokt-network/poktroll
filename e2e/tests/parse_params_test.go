//go:build e2e

package e2e

import (
	"fmt"

	cosmostypes "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/cosmos/gogoproto/proto"
	"github.com/regen-network/gocuke"

	apptypes "github.com/pokt-network/poktroll/x/application/types"
	prooftypes "github.com/pokt-network/poktroll/x/proof/types"
	servicetypes "github.com/pokt-network/poktroll/x/service/types"
	sessiontypes "github.com/pokt-network/poktroll/x/session/types"
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
	case sessiontypes.ModuleName:
		msgUpdateParams = s.newSessionMsgUpdateParams(paramsMap)
	case apptypes.ModuleName:
		msgUpdateParams = s.newAppMsgUpdateParams(paramsMap)
	case servicetypes.ModuleName:
		msgUpdateParams = s.newServiceMsgUpdateParams(paramsMap)
	// NB: gateway & supplier modules currently have no parameters
	default:
		err := fmt.Errorf("unexpected module name %q", moduleName)
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
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case prooftypes.ParamMinRelayDifficultyBits:
			msgUpdateParams.Params.MinRelayDifficultyBits = uint64(paramValue.value.(int64))
		case prooftypes.ParamRelayDifficultyBits:
			s.Fatalf("RelayDifficultyBits is an on-chain parameter and cannot be updated through governance proposals")
		default:
			s.Fatalf("unexpected %q type param name %q", paramValue.typeStr, paramName)
		}
	}
	return proto.Message(msgUpdateParams)
}

func (s *suite) newSessionMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &sessiontypes.MsgUpdateParams{
		Authority: authority,
		Params:    sessiontypes.Params{},
	}

	for paramName, paramValue := range params {
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case sessiontypes.ParamNumBlocksPerSession:
			msgUpdateParams.Params.NumBlocksPerSession = uint64(paramValue.value.(int64))
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

func (s *suite) newSessionMsgUpdateParams(params paramsMap) cosmostypes.Msg {
	authority := authtypes.NewModuleAddress(s.granterName).String()

	msgUpdateParams := &sessiontypes.MsgUpdateParams{
		Authority: authority,
		Params:    sessiontypes.Params{},
	}

	for paramName, paramValue := range params {
		s.Logf("paramName: %s, value: %v", paramName, paramValue.value)
		switch paramName {
		case sessiontypes.ParamNumBlocksPerSession:
			msgUpdateParams.Params.NumBlocksPerSession = uint64(paramValue.value.(int64))
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

	// TODO_IMPROVE: can this be simplified?
	switch moduleName {
	case tokenomicstypes.ModuleName:
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
		}
	case prooftypes.ModuleName:
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
		}
	default:
		err := fmt.Errorf("unexpected module name %q", moduleName)
		s.Fatal(err)
		panic(err)
	}

	return msg
}
