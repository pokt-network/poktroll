package config

import "net/url"

// HydratePocketNodeUrls populates the pocket node fields of the RelayMinerConfig
// that are relevant to the "pocket_node" section in the config file.
func (relayMinerConfig *RelayMinerConfig) HydratePocketNodeUrls(
	yamlPocketNodeConfig *YAMLRelayMinerPocketNodeConfig,
) error {
	relayMinerConfig.PocketNode = &RelayMinerPocketNodeConfig{}

	if len(yamlPocketNodeConfig.TxNodeRPCUrl) == 0 {
		return ErrRelayMinerConfigInvalidNodeUrl.Wrap("tx node rpc url is required")
	}

	// Check if the pocket node rpc url is a valid URL
	txNodeRPCUrl, err := url.Parse(yamlPocketNodeConfig.TxNodeRPCUrl)
	if err != nil {
		return ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
			"invalid tx node rpc url %s",
			err.Error(),
		)
	}
	relayMinerConfig.PocketNode.TxNodeRPCUrl = txNodeRPCUrl

	// If the query node rpc url is empty, use the tx node rpc url
	if len(yamlPocketNodeConfig.QueryNodeRPCUrl) == 0 {
		relayMinerConfig.PocketNode.QueryNodeRPCUrl = relayMinerConfig.PocketNode.TxNodeRPCUrl
	} else {
		// If the query node rpc url is not empty, make sure it is a valid URL
		queryNodeRPCUrl, parseErr := url.Parse(yamlPocketNodeConfig.QueryNodeRPCUrl)
		if parseErr != nil {
			return ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
				"invalid query node rpc url %s",
				parseErr.Error(),
			)
		}
		relayMinerConfig.PocketNode.QueryNodeRPCUrl = queryNodeRPCUrl
	}

	if len(yamlPocketNodeConfig.QueryNodeGRPCUrl) == 0 {
		return ErrRelayMinerConfigInvalidNodeUrl.Wrap("query node grpc url is required")
	}

	// Check if the query node grpc url is a valid URL
	queryNodeGRPCUrl, err := url.Parse(yamlPocketNodeConfig.QueryNodeGRPCUrl)
	if err != nil {
		return ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
			"invalid query node grpc url %s",
			err.Error(),
		)
	}
	relayMinerConfig.PocketNode.QueryNodeGRPCUrl = queryNodeGRPCUrl

	return nil
}
