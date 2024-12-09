Feature: Send Namespace

    Scenario: User can send uPOKT
        Given the user has the pocketd binary installed
        And the account "app1" has a balance greater than "1000" uPOKT
        And an account exists for "app2"
        When the user sends "1000" uPOKT from account "app1" to account "app2"
        Then the user should be able to see standard output containing "txhash:"
        And the user should be able to see standard output containing "code: 0"
        And the pocketd binary should exit without error
        And the user should wait for "5" seconds
        And the account balance of "app1" should be "1001" uPOKT "less" than before
        And the account balance of "app2" should be "1000" uPOKT "more" than before
