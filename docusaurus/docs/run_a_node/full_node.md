**Full Node - Docker Compose**

- [What is a Full Node](#what-is-a-full-node)
- [Docker Compose Example Repository](#docker-compose-example-repository)

### What is a Full Node

In a blockchain network, a Full Node retains a complete copy of the ledger, verifying all transactions and blocks according to the network's rules. While it does not play a role in block creation or consensus, it is crucial for ensuring data integrity, enhancing network security, and fostering decentralization. Full Nodes facilitate this by distributing transactions and blocks to other nodes.

Within the Pocket Network ecosystem, the role of Full Nodes is pivotal for Node Runners. These nodes are essential for off-chain entities like RelayMiners and AppGates, which rely on interaction with the Pocket Network blockchain for optimal functionality.

This guide outlines the setup process for a Full Node using Docker Compose, offering a simplified and efficient method for launching a Full Node.

### Docker Compose Example Repository

To help you understand how a Full Node can be operated with Docker Compose, we have prepared an example in the [poktroll-docker-compose-example](https://github.com/pokt-network/poktroll-docker-compose-example) GitHub repository.

Please refer to the "Deploying a Full Node" section of the guide in that repository for detailed instructions on setting up your Full Node using the provided Docker Compose example.