Feature: Loading anvil

  Scenario Outline: Anvil can handle the maximum number of concurrent users
    Given anvil is running
    And load of <num_requests> concurrent requests for the "eth_blockNumber" JSON-RPC method
    Then load is handled within <timeout> seconds

    Examples:
      | num_requests | timeout |
      | 10           | 1       |
      | 100          | 1       |
      | 1000         | 5       |
      | 10000        | 10      |
