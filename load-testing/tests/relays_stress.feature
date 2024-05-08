Feature: Loading gateway server with relays

  #Scenario Outline:
  Scenario: Incrementing the number of relays and actors
    Given localnet is running
    And a rate of "4" relay requests per second is sent per application
    And the following initial actors are staked:
      | actor       | count |
      | gateway     | 1     |
      | application | 4     |
      | supplier    | 1     |
    And more actors are staked as follows:
      | actor       | actor inc amount | blocks per inc | max actors |
      | gateway     | 1                | 40              | 3          |
      | application | 4                | 40              | 12         |
      | supplier    | 1                | 40              | 3          |
    When a load of concurrent relay requests are sent from the applications
    Then the correct pairs count of claim and proof messages should be committed on-chain
