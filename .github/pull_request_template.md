<!-- README: DELETE THIS COMMENT BLOCK after:
    1. Add a descriptive title `[<Tag>] <DESCRIPTION>`
    2. Update _Assignee(s)_
    3. Add _Label(s)_
    4. Set _Project(s)_
    5. Specify _Epic_ and _Iteration_ under _Project_
    6. Set _Milestone_
-->

## Summary

<!-- README: DELETE THIS COMMENT BLOCK after
      - Providing a quick summary of the changes yourself
-->

## Issue

<!-- README: DELETE THIS COMMENT BLOCK after:
     - Explain the reasoning for the PR in 1-2 sentences. Adding a screenshot is fair game.
     - If applicable: specify the ticket number below if there is a relevant issue; _keep the `-` so the full issue is referenced._
-->

- #{ISSUE_NUMBER}

## Type of change

Select one or more:

- [ ] New feature, functionality or library
- [ ] Bug fix
- [ ] Code health or cleanup
- [ ] Documentation
- [ ] Other (specify)

## Testing

**Documentation changes** (only if making doc changes)
- [ ] `make docusaurus_start`; only needed if you make doc changes

**Local Testing** (only if making code changes)
- [ ] **Unit Tests**: `make go_develop_and_test`
- [ ] **LocalNet E2E Tests**: `make test_e2e`
- See [quickstart guide](https://dev.poktroll.com/developer_guide/quickstart) for instructions

**PR Testing** (only if making code changes)
- [ ] **DevNet E2E Tests**: Add the `devnet-test-e2e` label to the PR.
    - **THIS IS VERY EXPENSIVE**, so only do it after all the reviews are complete.
    - Optionally run `make trigger_ci` if you want to re-trigger tests without any code changes
    - If tests fail, try re-running failed tests only using the GitHub UI as shown [here](https://github.com/pokt-network/poktroll/assets/1892194/607984e9-0615-4569-9452-4c730190c1d2)


## Sanity Checklist

- [ ] I have tested my changes using the available tooling
- [ ] I have commented my code
- [ ] I have performed a self-review of my own code; both comments & source code
- [ ] I create and reference any new tickets, if applicable
- [ ] I have left TODOs throughout the codebase, if applicable
