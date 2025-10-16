<<<<<<< Updated upstream
Feature: RelayMiner Relay Command

    # This test validates the `pocketd relayminer relay` CLI command works correctly.
    # It ensures backwards compatibility after the signer initialization fix.
    Scenario: App can send a relay using relayminer relay command
        Given the user has the pocketd binary installed
        And the application "app1" is staked for service "anvil"
        And the supplier "supplier1" is staked for service "anvil"
        And the session for application "app1" and service "anvil" contains the supplier "supplier1"
        When the user runs relayminer relay for app "app1" to supplier "supplier1" with payload '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
        Then the user should be able to see standard output containing "result"
        And the user should be able to see standard output containing "jsonrpc"
        And the pocketd binary should exit without error
=======
# TODO_INVESTIGATE: This test is disabled due to CGO changes - https://github.com/pokt-network/poktroll/discussions/1822
# This test validates the `pocketd relayminer relay` CLI command which depends on JSON stdout output.
# The JSON stdout output was removed as part of the CGO disabling changes.
# To re-enable: Uncomment the scenario below and restore the JSON stdout printing in pkg/relayer/cmd/cmd_relay.go

# Feature: RelayMiner Relay Command

    # This test validates the `pocketd relayminer relay` CLI command works correctly.
    # It ensures backwards compatibility after the signer initialization fix.
    # Scenario: App can send a relay using relayminer relay command
    #     Given the user has the pocketd binary installed
    #     And the application "app1" is staked for service "anvil"
    #     And the supplier "supplier1" is staked for service "anvil"
    #     And the session for application "app1" and service "anvil" contains the supplier "supplier1"
    #     When the user runs relayminer relay for app "app1" to supplier "supplier1" with payload '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
    #     Then the user should be able to see standard output containing "result"
    #     And the user should be able to see standard output containing "jsonrpc"
    #     And the pocketd binary should exit without error
>>>>>>> Stashed changes
