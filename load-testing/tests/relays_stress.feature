Feature: Loading gateway server with relays

  #Scenario Outline:
  Scenario: Incrementing the number of relays and actors
    Given localnet is running
    And the following initial actors are staked:
      | actor       | count |
      | gateway     | 1     |
      | application | 1     |
      | supplier    | 1     |
    And more actors are staked as follows:
      | actor       | actor inc rate | blocks per inc | max actors |
      | gateway     | 1              | 4              | 4          |
      | application | 1              | 4              | 3          |
      | supplier    | 1              | 4              | 2          |
    When a load of concurrent relay requests are sent per second as follows:
      | start relay rate | relay inc rate | blocks per inc | max relays rate |
      | 5                | 5              | 4              | 20              |

#    Examples:
#      |  |  |
#      |  |  |
