Feature: Loading gateway server with relays

  Scenario: Incrementing the number of relays and actors
    Given localnet is running
    And a rate of "1" relay requests per second is sent per application
    And the following initial actors are staked:
      | actor       | count |
      | application | 4     |
      | gateway     | 1     |
      | supplier    | 1     |
    And more actors are staked as follows:
      | actor       | actor inc amount | blocks per inc | max actors |
      | application | 4                | 10             | 12         |
      | gateway     | 1                | 10             | 3          |
      | supplier    | 1                | 10             | 1          |
    When a load of concurrent relay requests are sent from the applications
    Then the correct pairs count of claim and proof messages should be committed on-chain