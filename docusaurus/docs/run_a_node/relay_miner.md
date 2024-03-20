**RelayMiner - Docker Compose**

- [What is a RelayMiner](#what-is-a-relayminer)
- [Docker Compose Example Repository](#docker-compose-example-repository)

### What is a RelayMiner

A RelayMiner is a specialized node designed for individuals to offer services through the Pocket Network. For more information on this role, please refer to the [RelayMiner documentation](../actors/relay_miner.md). Unlike the Pocket Morse Mainnet client, which supports service provision on the current Pocket Network Mainnet and maintains a copy of the blockchain data, the RelayMiner operates without storing blockchain data. Instead, it relies on connections to Full Nodes for interacting with the blockchain. It is crucial to deploy a [Full Node](full_node.md) prior to setting up a RelayMiner, as this ensures the necessary infrastructure for blockchain communication is in place.

### Docker Compose Example Repository

To help you get started with deploying a RelayMiner, we have prepared an example using Docker Compose. This example is available in the [poktroll-docker-compose-example](https://github.com/pokt-network/poktroll-docker-compose-example?tab=readme-ov-file#deploying-a-relay-miner) GitHub repository.

Please refer to the "Deploying a RelayMiner" section within the repository for detailed instructions and guidance on setting up your RelayMiner using the provided Docker Compose example.
