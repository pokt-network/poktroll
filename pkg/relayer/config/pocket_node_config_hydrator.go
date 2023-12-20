package config

import "net/url"

// HydratePocketNodeUrls populates the pocket node fields of the RelayMinerConfig
// that are relevant to the "pocket_node" section in the config file.
func (relayMinerConfig *RelayMinerConfig) HydratePocketNodeUrls(
	yamlPocketNodeConfig *YAMLRelayMinerPocketNodeConfig,
) error {
	relayMinerConfig.PocketNode = &RelayMinerPocketNodeConfig{}

	if len(yamlPocketNodeConfig.TxNodeGRPCUrl) == 0 {
		return ErrRelayMinerConfigInvalidNodeUrl.Wrap("tx node grpc url is required")
	}

	// Check if the pocket node grpc url is a valid URL
	txNodeGRPCUrl, err := url.Parse(yamlPocketNodeConfig.TxNodeGRPCUrl)
	if err != nil {
		return ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
			"invalid tx node grpc url %s",
			err.Error(),
		)
	}
	relayMinerConfig.PocketNode.TxNodeGRPCUrl = txNodeGRPCUrl

	// If the query node grpc url is empty, use the tx node grpc url
	if len(yamlPocketNodeConfig.QueryNodeGRPCUrl) == 0 {
		relayMinerConfig.PocketNode.QueryNodeGRPCUrl = relayMinerConfig.PocketNode.TxNodeGRPCUrl
	} else {
		// If the query node grpc url is not empty, make sure it is a valid URL
		queryNodeGRPCUrl, err := url.Parse(yamlPocketNodeConfig.QueryNodeGRPCUrl)
		if err != nil {
			return ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
				"invalid query node grpc url %s",
				err.Error(),
			)
		}
		relayMinerConfig.PocketNode.QueryNodeGRPCUrl = queryNodeGRPCUrl
	}

	if len(yamlPocketNodeConfig.QueryNodeRPCUrl) == 0 {
		return ErrRelayMinerConfigInvalidNodeUrl.Wrap("query node rpc url is required")
	}

	// Check if the query node rpc url is a valid URL
	queryNodeRPCUrl, err := url.Parse(yamlPocketNodeConfig.QueryNodeRPCUrl)
	if err != nil {
		return ErrRelayMinerConfigInvalidNodeUrl.Wrapf(
			"invalid query node rpc url %s",
			err.Error(),
		)
	}
	relayMinerConfig.PocketNode.QueryNodeRPCUrl = queryNodeRPCUrl

	return nil
}
