package integration_test

// TODO_TEST: Test the whole supplier lifecycle using `NewCompleteIntegrationApp`
// 1. Stake a supplier
// 2. Check that is is not included in sessions
// 3. Wait until it is activated
// 4. Check that is is included in sessions
// 5. Unstake the supplier mid session
// 6. Check that it is included in the current session
// 7. Check that it is not included in the next session
// 8. Submit a proof and a claim and ensure it is successful
// 9. Check that it gets rewarded when settling the claim
// 10. Check that it gets removed from the suppliers list after the unbonding period is over
