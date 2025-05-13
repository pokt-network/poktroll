## Summary

< One line summary>

You can use the following as input for an LLM of your choice to autogenerate a summary (ignoring any additional files needed):

```bash
git --no-pager diff main  -- ':!*.pb.go' ':!*.pulsar.go' ':!*.json' ':!*.yaml' ':!*.yml' ':!*.gif' ':!*.lock' | diff2html -s side --format json -i stdin -o stdout | pbcopy
```

### Primary Changes:

- < Change 1 >
- < Change 2 > 

### Secondary Changes:

- < Change 1 >
- < Change 2 > 

## Issue

- Issue_or_PR: #{ISSUE_OR_PR_NUMBER}

## Type of change

Select one or more from the following:

- [ ] New feature, functionality or library
- [ ] Bug fix
- [ ] Code health or cleanup
- [ ] Documentation
- [ ] Other (specify)

## Sanity Checklist

- [ ] I have updated the GitHub Issue Metadata: `assignees`, `reviewers`, `labels`, `project`, `iteration` and `milestone`
- [ ] For docs: `make docusaurus_start`
- [ ] For small changes: `make go_develop_and_test` and `make test_e2e`
- [ ] For major changes: `devnet-test-e2e` label to run E2E tests in CI
- [ ] For migration changes: `make test_e2e_oneshot`
- [ ] 'TODO's, configurations and other docs
