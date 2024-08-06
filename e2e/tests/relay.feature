Feature: Relay Namespace

    # NB: `make acc_initialize_pubkeys` must have been executed before this test is run
    Scenario: App can send a JSON-RPC relay to Supplier
        Given the user has the pocketd binary installed
        And the application "app1" is staked for service "anvil"
        And the supplier "supplier1" is staked for service "anvil"
        And the session for application "app1" and service "anvil" contains the supplier "supplier1"
        When the application "app1" sends the supplier "supplier1" a request for service "anvil" with path "" and data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'
        Then the application "app1" receives a successful relay response signed by "supplier1" for data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}'

    Scenario: App can send a REST relay to Supplier
        Given the user has the pocketd binary installed
        And the application "app1" is staked for service "ollama"
        And the supplier "supplier1" is staked for service "ollama"
        And the session for application "app1" and service "ollama" contains the supplier "supplier1"
        When the application "app1" sends the supplier "supplier1" a request for service "ollama" with path "/api/chat" and data '{"model": "qwen:0.5b", "stream": false, "messages": [{"role": "user", "content":"count from 1 to 10"}]}'
        Then the application "app1" receives a successful relay response signed by "supplier1" for data data '{"model": "qwen:0.5b", "stream": false, "messages": [{"role": "user", "content":"count from 1 to 10"}]}'
        And a "tokenomics" module "ClaimSettled" end block event is broadcast

    # TODO_TEST(@Olshansk):
    # - Successful relay through applicat's sovereign appgate server
    # - Successful relay through gateway app is delegation to
    # - Successful relay through gateway when app is delegating to multiple gateways
    # - Failed relay through gateway app is not delegation to
    # - Succeedful relays when using multiple suppliers for app in some session
    # - Error if app1 is not staked for svc1 but relay is sent
    # - Error if supplier is not staked for svc1 but relay is sent
    # - Error if claiming the session too early
    # - Error if proving the session too early