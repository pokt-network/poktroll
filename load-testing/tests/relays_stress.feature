Feature: Loading gateway server with relays

  Scenario: Incrementing the number of relays and actors
    Given localnet is running
    And a rate of "1" relay requests per second is sent per application
    And the following initial actors are staked:
      | actor       | count |
      | gateway     | 1     |
      | application | 1     |
      | supplier    | 1     |
    And more actors are staked as follows:
      | actor       | actor inc amount | blocks per inc | max actors |
      | gateway     | 1                | 4              | 3          |
      | application | 2                | 4              | 6          |
      | supplier    | 1                | 4              | 3          |
    When a load of concurrent relay requests are sent from the applications
#    Then "12" pairs of claim and proof messages should be committed on-chain
