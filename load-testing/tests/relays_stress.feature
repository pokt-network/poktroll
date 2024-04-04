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
      | gateway     | 1              | 4              | 3          |
      | application | 2              | 4              | 5          |
      | supplier    | 1              | 4              | 3          |
    When a load of "1" concurrent relay requests are sent per application per second

#    Examples:
#      |  |  |
#      |  |  |
