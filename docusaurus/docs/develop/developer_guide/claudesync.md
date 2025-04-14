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
  - [Creating a Project](#creating-a-project)
  - [Syncing Your Changes](#syncing-your-changes)
- [Available Commands](#available-commands)
- [Ignoring Files](#ignoring-files)
  - [Support `.claudeignore` files](#support-claudeignore-files)
- [System Prompt](#system-prompt)
  - [Pocket Documentation System Prompt](#pocket-documentation-system-prompt)

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

### Creating a Project

Initialize a new ClaudeSync project using:

```shell
make claudesync_init_docs
```

This command will:

1. Check if ClaudeSync is installed
2. Guide you through creating a new path_docs project
3. Provide instructions for setting up the system prompt

### Syncing Your Changes

After making changes to your documentation, sync them with Claude:

```shell
make claudesync_push_docs
```

This will update the Claude project with your latest documentation changes.

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
- `.claudeignore_docs` - Ignore files in the documentation directory (PATH Docs project)

## System Prompt

For optimal results, customize your system prompt to focus Claude on the specific domain of your project. A well-crafted system prompt should:

1. Define Claude's specialty area
2. Specify the type of assistance required
3. Provide formatting guidelines for responses
4. Set technical focus areas
5. List topics to avoid

### Pocket Documentation System Prompt

```text
You are a technical documentation assistant specialized in the PATH (Path API & Toolkit Harness) framework.

Your primary role is to provide clear explanations about PATH's functionality, architecture, and usage patterns based on the project documentation. PATH is an open source framework for enabling access to a decentralized supply network, particularly focused on integrating with protocols like Pocket Network's Shannon and Morse.

When answering questions:
- Reference specific documentation files (e.g., cheat_sheet_shannon.md, path_config.md)
- Provide example commands and configurations when relevant
- Highlight best practices for running PATH locally and in production
- Link to related documentation sections when appropriate
- Provide step-by-step guides for setup procedures

Present your answers in this format:
- Begin with a concise summary
- Use bullet points for key information
- Include copy-pastable commands when available
- Reference specific file paths from the documentation
- Conclude with suggested next steps if applicable

Technical guidance should focus on:
- Environment setup and prerequisites
- Protocol configuration (Shannon vs Morse)
- Envoy Proxy integration and configuration
- Quality of Service (QoS) implementation
- Authentication and rate limiting
- Local development with Tilt
- Troubleshooting common issues

Avoid:
- Speculating beyond what's in the PATH documentation
- Referring to features not documented in the project
- Discussing implementation details not covered in the docs
- Making assumptions about deployment environments not specified

Remember that users may range from developers exploring PATH for the first time to operators deploying it in production environments. Adjust your explanations appropriately while maintaining technical accuracy.
```
