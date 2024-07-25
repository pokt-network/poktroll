Feature: Service Namespace

    Scenario: A source owner adds a new service
        # Given the user has the pocketd binary installed
        # And an account exists for "supplier1"
        # And the "supplier" account for "supplier1" is staked
        # And an account exists for "app1"
        # And the "application" account for "app1" is staked
        # And the service "anvil" registered for application "app1" has a compute units per relay of "1"
        # When the supplier "supplier1" has serviced a session with "10" relays for service "anvil" for application "app1"
        # And the user should wait for the "proof" module "CreateClaim" Message to be submitted
        # And the user should wait for the "proof" module "SubmitProof" Message to be submitted
        # And the user should wait for the "tokenomics" module "ClaimSettled" end block event to be broadcast
        # Then the account balance of "supplier1" should be "420" uPOKT "more" than before
        # And the "application" stake of "app1" should be "420" uPOKT "less" than before

    Scenario: A source owner successfully updates a service they own

    Scenario: A source fails to update a service they do not own