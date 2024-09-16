#############
### TODOS ###
#############

# How do I use TODOs?
# 1. <KEYWORD>: <Description of follow up work>;
# 	e.g. TODO_HACK: This is a hack, we need to fix it later
# 2. If there's a specific issue, or specific person, add that in paranthesiss
#   e.g. TODO(@Olshansk): Automatically link to the Github user https://github.com/olshansk
#   e.g. TODO_INVESTIGATE(#420): Automatically link this to github issue https://github.com/pokt-network/poktroll/issues/420
#   e.g. TODO_DISCUSS(@Olshansk, #420): Specific individual should tend to the action item in the specific ticket
#   e.g. TODO_CLEANUP(core): This is not tied to an issue, or a person, but should only be done by the core team.
#   e.g. TODO_CLEANUP: This is not tied to an issue, or a person, and can be done by the core team or external contributors.
# 3. Feel free to add additional keywords to the list above.

# Inspired by @goldinguy_ in this post: https://goldin.io/blog/stop-using-todo ###
# TODO                        - General Purpose catch-all.
# TODO_COMMUNITY              - A TODO that may be a candidate for outsourcing to the community.
# TODO_DECIDE                 - A TODO indicating we need to make a decision and document it using an ADR in the future; https://github.com/pokt-network/pocket-network-protocol/tree/main/ADRs
# TODO_TECHDEBT               - Not a great implementation, but we need to fix it later.
# TODO_BLOCKER                - BEFORE MAINNET. Similar to TECHDEBT, but of higher priority, urgency & risk prior to the next release
# TODO_QOL                    - AFTER MAINNET. Similar to TECHDEBT, but of lower priority. Doesn't deserve a GitHub Issue but will improve everyone's life.
# TODO_IMPROVE                - A nice to have, but not a priority. It's okay if we never get to this.
# TODO_OPTIMIZE               - An opportunity for performance improvement if/when it's necessary
# TODO_DISCUSS                - Probably requires a lengthy offline discussion to understand next steps.
# TODO_INCOMPLETE             - A change which was out of scope of a specific PR but needed to be documented.
# TODO_INVESTIGATE            - TBD what was going on, but needed to continue moving and not get distracted.
# TODO_CLEANUP                - Like TECHDEBT, but not as bad.  It's okay if we never get to this.
# TODO_HACK                   - Like TECHDEBT, but much worse. This needs to be prioritized
# TODO_REFACTOR               - Similar to TECHDEBT, but will require a substantial rewrite and change across the codebase
# TODO_CONSIDERATION          - A comment that involves extra work but was thoughts / considered as part of some implementation
# TODO_CONSOLIDATE            - We likely have similar implementations/types of the same thing, and we should consolidate them.
# TODO_ADDTEST / TODO_TEST    - Add more tests for a specific code section
# TODO_FLAKY                  - Signals that the test is flaky and we are aware of it. Provide an explanation if you know why.
# TODO_DEPRECATE              - Code that should be removed in the future
# TODO_RESEARCH               - A non-trivial action item that requires deep research and investigation being next steps can be taken
# TODO_DOCUMENT		          - A comment that involves the creation of a README or other documentation
# TODO_BUG                    - There is a known existing bug in this code
# TODO_NB                     - An important note to reference later
# TODO_DISCUSS_IN_THIS_COMMIT - SHOULD NEVER BE COMMITTED TO MASTER. It is a way for the reviewer of a PR to start / reply to a discussion.
# TODO_IN_THIS_COMMIT         - SHOULD NEVER BE COMMITTED TO MASTER. It is a way to start the review process while non-critical changes are still in progress


# Define shared variable for the exclude parameters
EXCLUDE_GREP = --exclude-dir={.git,vendor,./docusaurus,.vscode,.idea} --exclude={Makefile,reviewdog.yml,*.pb.go,*.pulsar.go}

.PHONY: todo_list
todo_list: ## List all the TODOs in the project (excludes vendor and prototype directories)
	grep -r $(EXCLUDE_GREP) TODO . | grep -v 'TODO()'

.PHONY: todo_count
todo_count: ## Print a count of all the TODOs in the project
	grep -r $(EXCLUDE_GREP) TODO . | grep -v 'TODO()' | wc -l

.PHONY: todo_this_commit
todo_this_commit: ## List all the TODOs needed to be done in this commit
	grep -r $(EXCLUDE_GREP) TODO_IN_THIS .| grep -v 'TODO()'
