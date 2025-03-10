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
      | gateway     | 1                | 10             | 1          |
      | supplier    | 1                | 10             | 1          |
    When a load of concurrent relay requests are sent from the applications
    Then the number of failed relay requests is "0"
    # TODO_FOLLOWUP(@red-0ne): Implement the following steps
    # Then "0" over servicing events are observed
    # And "0" slashing events are observed
    # And "0" expired claim events are observed
    # And there are as many reimbursement requests as the number of settled claims
    # And the number of claims submitted and claims settled is the same
    # And the number of proofs submitted and proofs required is the same
    # And the actors onchain balances are as expected
    # TODO_CONSIDERATION: Revisit for additional interesting test cases.