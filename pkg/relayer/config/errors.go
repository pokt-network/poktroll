package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                                        = "relayminer_config"
	ErrRelayMinerConfigUnmarshalYAML                 = sdkerrors.Register(codespace, 1, "config reader cannot unmarshal yaml content")
	ErrRelayMinerConfigInvalidQueryNodeGRPCUrl       = sdkerrors.Register(codespace, 2, "invalid query node grpc url in RelayMiner config")
	ErrRelayMinerConfigInvalidNetworkNodeGRPCUrl     = sdkerrors.Register(codespace, 3, "invalid network node grpc url in RelayMiner config")
	ErrRelayMinerConfigInvalidPocketNodeWebsocketUrl = sdkerrors.Register(codespace, 4, "invalid pocket node websocket url in RelayMiner config")
	ErrRelayMinerConfigInvalidServiceEndpoint        = sdkerrors.Register(codespace, 5, "invalid service endpoint in RelayMiner config")
	ErrRelayMinerConfigInvalidSigningKeyName         = sdkerrors.Register(codespace, 6, "invalid signing key name in RelayMiner config")
	ErrRelayMinerConfigInvalidSmtStorePath           = sdkerrors.Register(codespace, 7, "invalid smt store path in RelayMiner config")
)
