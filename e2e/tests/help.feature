Feature: Root Namespace

    Scenario: User Needs Help
        Given the user has the pocketd binary installed
        When the user runs the command "help"
        Then the user should be able to see standard output containing "Available Commands"
        And the pocketd binary should exit without error
