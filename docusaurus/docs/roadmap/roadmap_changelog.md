---
title: Roadmap Changelog
---

# Roadmap Changelog <!-- omit in toc -->

The purpose of this doc is to keep track of the changes made to the [Shannon roadmap](https://github.com/orgs/pokt-network/projects/144).

- [Relevant links](#relevant-links)
- [11/13/2023](#11132023)
  - [Change #2](#change-2)
  - [After Change #2](#after-change-2)
  - [Before Change #2](#before-change-2)
- [11/01/2023](#11012023)
  - [Change #1](#change-1)
  - [After Change #1](#after-change-1)
  - [Before Change #1](#before-change-1)

## Relevant links

- [Shannon Project](https://github.com/orgs/pokt-network/projects/144?query=is%3Aopen+sort%3Aupdated-desc) - GitHub dashboard
- [Shannon Roadmap](https://github.com/orgs/pokt-network/projects/144/views/4?query=is%3Aopen+sort%3Aupdated-desc) - GitHub Roadmap
- [PoktRoll Repo](https://github.com/pokt-network/poktroll) - Source Code
- [PoktRoll Issues](https://github.com/pokt-network/poktroll/issues) - GitHub Issues
- [PoktRoll Milestones](https://github.com/pokt-network/poktroll/milestones) - GitHub Milestones

## 11/13/2023

### Change #2

A few changes were made, but the important pieces are the focus of each of the next three iterations:

- `Iteration 4 (TECHDEBT #1 - E2E Relay)`

  - `Why`: MVP E2E Relay + C&P lifecycle built top of automated tests to prevent regression
  - Tend to the techdebt (hacks, workarounds, etc) that were added to the codebase needed to enable the E2E Relay
  - Automate the E2E Relay test
  - Add unit tests to various components involved in the E2E Relay
  - Implement the MVP of the Claim & Proof lifecycle

- `Iteration 5 (Load & Discover)`

  - `Why`: De-risk permissionless applications and fill in missing gaps in our understanding of Gateway requirements
  - Integrate Grove's portal in Shannon's LocalNet & DevNet (regression, learning, etc...)
  - Build out the first iteration of the infrastructure & tooling needed for load-testing permissionless apps
  - Introduce logging, telemetry, block explorers, etc to gain visibility into the network

- `Iteration 6 (Deploy TestNet #1)`

  - `Why`: Deploy the first TestNet to fill unknown gaps in our knowledge and provide feedback to the Rollkit team on things we will need in the future
  - Deploy the first iteration of Shannon to Celestia's Mocha TestNet
  - Identify missing gaps in our knowledge of TestNet deployment, missing infrastructure needs, and de-risk timelines
  - Invite a node runner from the community to start running a Supplier in TestNet
  - Continue some on-chain feature development

### After Change #2

![Screenshot 2023-11-13 at 12 42 02 PM](https://github.com/pokt-network/poktroll/assets/1892194/d1bc7be8-47c3-4358-be77-626533c7f98e)

### Before Change #2

![Screenshot 2023-11-13 at 11 51 46 AM](https://github.com/pokt-network/pocket/assets/1892194/68e3348d-5b56-4f6b-9bb6-799e683073c8)

## 11/01/2023

### Change #1

1. We're adding a 1 week `E2E Relay` iteration to focus solely on finishing off `Foundation` & `Integration` related work needed to enable automating end-to-end relays.
2. We've delayed the `Govern` iteration to next year because:
   - It is not a blocker for TestNetT
   - here are still open-ended questions from PNF that need to be addressed first.
3. We've introduced `TECHDEBT` iterations to tackle `TODOs` left throughout the code.
   - The first iteration will be focused on `TODO_BLOCKER` in the source code
   - Details to other iterations will be ironed out closer to the iteration.
4. We have decided to have multiple `Test` iterations, each of which will be focused on testing different components.
   - The first iteration will be focused on load testing relays to de-risk permissionless applications and verify the Claim & Proof lifecycle.
   - Details to each iteration will be ironed out closer to the iteration.

### After Change #1

![Screenshot 2023-11-01 at 2 15 09 PM](https://github.com/pokt-network/poktroll/assets/1892194/e8ef99e6-aecc-433b-8a32-5fb42c05cb86)

### Before Change #1

![Screenshot 2023-11-01 at 11 05 21 AM](https://github.com/pokt-network/poktroll/assets/1892194/0826d4af-d0e1-4edc-a173-362425672c64)
