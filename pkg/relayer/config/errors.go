package config

import sdkerrors "cosmossdk.io/errors"

var (
	codespace                                = "relayminer_config"
	ErrRelayMinerConfigUnmarshalYAML         = sdkerrors.Register(codespace, 2100, "config reader cannot unmarshal yaml content")
	ErrRelayMinerConfigInvalidNodeUrl        = sdkerrors.Register(codespace, 2101, "invalid node url in RelayMiner config")
	ErrRelayMinerConfigInvalidSigningKeyName = sdkerrors.Register(codespace, 2102, "invalid signing key name in RelayMiner config")
	ErrRelayMinerConfigInvalidSmtStorePath   = sdkerrors.Register(codespace, 2103, "invalid smt store path in RelayMiner config")
	ErrRelayMinerConfigEmpty                 = sdkerrors.Register(codespace, 2104, "empty RelayMiner config")
	ErrRelayMinerConfigInvalidSupplier       = sdkerrors.Register(codespace, 2105, "invalid supplier in RelayMiner config")
	ErrRelayMinerConfigInvalidServer         = sdkerrors.Register(codespace, 2106, "invalid server in RelayMiner config")
	ErrRelayerMinerWrongForwardToken         = sdkerrors.Register(codespace, 2107, "wrong or empty forward.token in configuration file. (you can use 'make relayminer_forward_token_gen' command to generate a token)")
)
