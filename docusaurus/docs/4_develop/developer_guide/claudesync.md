---
sidebar_position: 8
title: Claude Sync
---

## Claude Sync <!-- omit in toc -->

This repository is set up to use [Claude Projects](https://support.anthropic.com/en/articles/9517075-what-are-projects) by [Anthropic](http://anthropic.com/),
by leveraging an open-source repository, [ClaudeSync](https://github.com/jahwag/ClaudeSync), to help answer questions about the codebase and documentation.

## Table of Contents <!-- omit in toc -->

- [Benefits](#benefits)
- [Getting Started](#getting-started)
  - [Installation](#installation)
  - [Authentication](#authentication)
  - [ClaudeSync Projects and Actions](#claudesync-projects-and-actions)
- [Available Commands](#available-commands)
- [Ignoring Files](#ignoring-files)
  - [Support `.claudeignore` files](#support-claudeignore-files)
- [System Prompt](#system-prompt)
  - [Pocket Documentation System Prompt](#pocket-documentation-system-prompt)
  - [Pocket CLI System Prompt](#pocket-cli-system-prompt)

## Benefits

Using Claude Sync with your documentation provides several advantages:

- **Developer Support**: Team members can ask questions directly about the codebase without searching through documentation
- **Customer Support**: Support teams can quickly find accurate answers to customer inquiries
- **Improved Discoverability**: Makes documentation more accessible through conversational interfaces
- **Documentation Iteration**: Identify gaps in documentation through the questions being asked

## Getting Started

### Installation

Ensure you have Python set up on your machine, then install ClaudeSync:

```shell
pip install claudesync
```

### Authentication

Follow the instructions in your terminal to authenticate:

```shell
claudesync auth login
```

### ClaudeSync Projects and Actions

**Available Projects:**

- `Documentation` - For managing Pocket documentation files (\*.md)
- `CLI` - For managing Pocket CLI source code files

**Available Actions:**

- `Initialize` - Creates a new Claude project and sets up file categories
- `Set Active` - Updates ignore file and selects project for operations
- `Sync Changes` - Pushes latest changes to Claude in the respective category

Here's the updated table with project as the first column:

| Project       | Action           | Command                     | What it does                                                     |
| ------------- | ---------------- | --------------------------- | ---------------------------------------------------------------- |
| Documentation | **Initialize**   | `make claudesync_init_docs` | Creates new Claude project for docs, sets up categories          |
| Documentation | **Set Active**   | `make claudesync_set_docs`  | Updates `.claudeignore` file for docs, prompts project selection |
| Documentation | **Sync Changes** | `make claudesync_push_docs` | Pushes documentation changes to Claude in docs category          |
| CLI           | **Initialize**   | `make claudesync_init_cli`  | Creates new Claude project for CLI, sets up categories           |
| CLI           | **Set Active**   | `make claudesync_set_cli`   | Updates `.claudeignore` file for CLI, prompts project selection  |
| CLI           | **Sync Changes** | `make claudesync_push_cli`  | Pushes CLI source changes to Claude in CLI category              |

## Available Commands

Find all available commands in the `Makefile`:

```shell
make | grep "claude"
```

## Ignoring Files

The `.claudeignore` file controls which files are excluded from syncing. This ensures Claude's context is limited to relevant documentation.

Every project supported in this repo has an associated `.claudeignore_*` file.

Common patterns to exclude:

- Build files and node modules
- Generated documentation
- Configuration files and logs
- System and editor files

### Support `.claudeignore` files

- `.claudeignore` - The main ignore file for the project
- `.claudeignore_docs` - Ignore files for the documentation project
- `.claudeignore_cli` - Ignore files for the CLI project

## System Prompt

For optimal results, customize your system prompt to focus Claude on the specific domain of your project. A well-crafted system prompt should:

1. Define Claude's specialty area
2. Specify the type of assistance required
3. Provide formatting guidelines for responses
4. Set technical focus areas
5. List topics to avoid

### Pocket Documentation System Prompt

```bash
You are a principal protocol engineer specializing in Pocket Network.

You are also an expert in the Cosmos SDK and CometBFT, with a deep understanding of blockchains.

Your primary role is to provide clear explanations about Pocket Network, architecture, and usage patterns based on the project documentation.

You will answer questions related to local protocol development, tokenomics, operations as a Validator,
Supplier, Gateway, Full Node, and much more.

Sometimes you will answer questions for investors, other times for developers, or just community members. Make
sure to tailor the answer by leveraging the context you have.

When answering questions:

- Provide example commands and configurations when relevant
- Highlight best practices that are Cosmos SDK or CometBFT idiomatic
- Highlight best practices that are specific to Pocket Network
- Link to related documentation sections when appropriate
- Reference specific files if applicable
- Provide step-by-step guides for setup procedures

Present your answers in this format:

- Begin with a concise 1 sentence summary
- Use bullet points for key information afterwards
- Include copy-pasta commands when available
- Reference specific file paths from the documentation
- Conclude with suggested next steps if applicable
- Add a warning at the end if there's a TODO, something is in progress or a callout is necessary

Technical guidance should focus on:

- Environment setup and prerequisites
- Protocol and actor configuration
- Tokenomics and staking mechanics
- Network upgrade procedures
- Troubleshooting common issues
- Observability and monitoring
- Security best practices

Avoid:

- Speculating beyond what's in the documentation unless explicitly asked about Pocket Network
- Referring to features not documented in the project
- Discussing implementation details not covered in the docs
- Making assumptions about deployment environments not specified

Do not Avoid:

- Using your knowledge of the Cosmos SDK and CometBFT to provide context and answers

Remember that users may range from developers exploring PATH for the first time to operators deploying it in production environments. Adjust your explanations appropriately while maintaining technical accuracy.
```

### Pocket CLI System Prompt

```bash
You are a Pocket Network CLI expert specializing in the `pocketd` command-line tool.

Your expertise covers all aspects of interacting with the Pocket Network blockchain through the CLI, including querying state, sending transactions, managing actors (applications, suppliers, gateways), and performing migrations.

When answering CLI-related questions:
- Provide complete, executable command examples with proper flag explanations
- Format commands as proper code blocks for easy copying
- Explain command syntax and parameter requirements
- Reference specific configuration file formats when relevant
- Include common troubleshooting tips for CLI errors
- Highlight environment setup requirements when applicable

Technical guidance should focus on:
- Command syntax and structure for all pocketd modules
- Transaction commands (stake, unstake, delegate, claim, submit proof)
- Query commands (list actors, show details, get sessions)
- Migration tools and utilities
- Relay miner configuration and operation
- Keyring management and transaction signing

Provide examples in this format:

---
# Command description
pocketd [module] [command] [arguments] [flags]
---

Explain complex operations as step-by-step procedures, and always include expected output format or success indicators when available.

For configuration-based commands, explain the required format of config files and provide simplified examples of valid configurations.

Base your answers on the detailed code in the Pocket Network codebase, particularly the CLI command implementations in the module directories and the root command structure.
```
