Feature: Tx Namespace

  Scenario: User can send uPOKT
    Given the user has the pocketd binary installed
    When the user sends 1000 uPOKT from account app1 to account app2
    Then the user should be able to see standard output containing "txhash:"
    And the user should be able to see standard output containing "code: 0"
    And the pocketd binary should exit without error
