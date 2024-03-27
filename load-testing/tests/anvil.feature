Feature: Loading anvil

Scenario Outline: Anvil can handle the maximum number of concurrent users
  Given anvil is running
  And load of <users> concurrent users
  When each user requests the ethereum block height
  Then load is handled within <timeout> seconds

    Examples:
      | users | timeout |
      | 10    | 1       |
      | 100   | 1       |
#      | 1000  | 5       |
#      | 10000 | 10      |
