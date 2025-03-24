#############
### TODOS ###
#############
## Inspired by @goldinguy_ in this post: https://goldin.io/blog/stop-using-todo ###
#
## How do I use TODOs?
#
# 1. TODO_<KEYWORD>: <Description of the work todo and why>
# 	e.g. TODO_HACK: This is a hack, we need to fix it later
#
# 2. If there's a specific issue and/or person, include it in paranthesiss
#
#   e.g. TODO(@Olshansk): Automatically link to the Github user https://github.com/olshansk
#   e.g. TODO_INVESTIGATE(#420): Automatically link this to github issue https://github.com/pokt-network/poktroll/issues/420
#   e.g. TODO_DISCUSS(@Olshansk, #420): Specific individual should tend to the action item in the specific ticket
#   e.g. TODO_CLEANUP: This is not tied to an issue, or a person, and can be done by the core team or external contributors.
#
# 3. Feel free to add additional keywords to the list above.
#
## TODO LIST
# TODO                        - General Purpose catch-all. Try to keep the usage of this to a minimum.
# TODO_COMMUNITY              - A TODO that may be a candidate for outsourcing to the community.
# TODO_DECIDE                 - A TODO indicating we need to make a decision and document it using an ADR in the future; https://github.com/pokt-network/pocket-network-protocol/tree/main/ADRs
# TODO_TECHDEBT               - Code that works but isn’t ideal; needs a fix to improve maintainability and avoid accumulating technical debt.
# TODO_QOL                    - AFTER MAINNET. Low-priority improvements that enhance the code quality or developer experience.
# TODO_IMPROVE                - A nice-to-have but not a priority; it’s okay if we never get to this.
# TODO_OPTIMIZE               - Performance improvements that can be pursued if performance demands increase.
# TODO_DISCUSS                - Marks code that requires a larger discussion with the team to clarify next steps or make decisions.
# TODO_INCOMPLETE             - Notes unfinished work or partial implementation that should be completed later.
# TODO_INVESTIGATE            - Requires more investigation to clarify issues or validate behavior; avoid premature optimizations without research.
# TODO_CLEANUP                - Lower-priority cleanup or refactoring tasks; it’s acceptable if this is never prioritized.
# TODO_HACK                   - Code is functional but particularly bad; prioritization needed to fix due to the hacky implementation.
# TODO_REFACTOR               - Indicates a need for a significant rewrite or architectural change across the codebase.
# TODO_CONSIDERATION          - Marks optional considerations or ideas related to the code that could be explored later.
# TODO_CONSOLIDATE            - Similar implementations or data structures are likely scattered; consolidate to reduce duplication.
# TODO_TEST / TODO_ADDTEST    - Signals that additional test coverage is needed in this code section.
# TODO_FLAKY                  - Known flaky test; provides an explanation if there’s an understanding of the root cause.
# TODO_DEPRECATE              - Marks code slated for eventual removal or replacement.
# TODO_RESEARCH               - Requires substantial research or exploration before proceeding with next steps or optimization.
# TODO_DOCUMENT               - Involves creating or updating documentation, such as READMEs, inline comments, or other resources.
# TODO_BUG                    - Known bug exists; this should be prioritized based on severity.
# TODO_NB                     - Important note that may not require immediate action but should be referenced later.
# TODO_IN_THIS_???            - Indicates ongoing non-critical changes before final review. THIS SHOULD NEVER BE COMMITTED TO MASTER and has workflows to prevent it.
# TODO_UPNEXT(@???)			  - Indicates this should be done shortly after an existing PR. THIS MUST HAVE A USER ASSIGNED TO IT and has workflows to prevent it.

# Define shared variable for the exclude parameters
EXCLUDE_GREP = --exclude-dir={.git,vendor,./docusaurus,.vscode,.idea} --exclude={Makefile,reviewdog.yml,*.pb.go,*.pulsar.go}

.PHONY: todo_list
todo_list: ## List all the TODOs in the project (excludes vendor and prototype directories)
	grep -r $(EXCLUDE_GREP) TODO . | grep -v 'TODO()'

.PHONY: todo_count
todo_count: ## Print a count of all the TODOs in the project
	grep -r $(EXCLUDE_GREP) TODO . | grep -v 'TODO()' | wc -l