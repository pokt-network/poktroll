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
      | actor       | increment | blocks | max |
      | gateway     | 1         | 3      | 5   |
      | application | 1         | 3      | 5   |
      | supplier    | 1         | 3      | 5   |
    When a load of concurrent relay requests are sent per second as follows:
      | initial relays per sec | inc relays per sec | blocks per inc | max relays per sec |
      | 1                      | 5                  | 2              | 20                 |

#    Examples:
#      |  |  |
#      |  |  |
