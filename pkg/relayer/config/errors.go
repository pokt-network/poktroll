package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                                = "relayminer_config"
	ErrRelayMinerConfigUnmarshalYAML         = sdkerrors.Register(codespace, 1, "config reader cannot unmarshal yaml content")
	ErrRelayMinerConfigInvalidNodeUrl        = sdkerrors.Register(codespace, 2, "invalid node url in RelayMiner config")
	ErrRelayMinerConfigInvalidSigningKeyName = sdkerrors.Register(codespace, 3, "invalid signing key name in RelayMiner config")
	ErrRelayMinerConfigInvalidSmtStorePath   = sdkerrors.Register(codespace, 4, "invalid smt store path in RelayMiner config")
	ErrRelayMinerConfigEmpty                 = sdkerrors.Register(codespace, 5, "empty RelayMiner config")
	ErrRelayMinerConfigInvalidSupplier       = sdkerrors.Register(codespace, 6, "invalid supplier in RelayMiner config")
	ErrRelayMinerConfigInvalidProxy          = sdkerrors.Register(codespace, 7, "invalid proxy in RelayMiner config")
)
