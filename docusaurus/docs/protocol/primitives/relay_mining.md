---
title: Relay Mining
sidebar_position: 3
---

# Relay Mining <!-- omit in toc -->

:::warning

TODO_DOCUMENT(@Olshansk): This is just a placeholder. Use the [relay mining presentation](https://docs.google.com/presentation/d/1xlCGzS_oHXJOzvcu-jHZUfmhD3qeVCzc6SUSJijTuJ4/edit#slide=id.p) and
the [relay mining paper](https://arxiv.org/abs/2305.10672) as a reference for writing this.

:::

- [Introduction](#introduction)

## Introduction

tl;dr Modulate on-chain difficulty up (similar to Bitcoin) so we can accommodate
surges in relays and have no upper limit on the number of relays per session.

Relay Mining is the only solution in Web3 to incentivizing read-only requests
and solving for the problem of high volume: `how can we scale to billions or trillions
of relays per session`.

This complements the design of [Probabilistic Proofs](./probabilistic_proofs.md)
to solve for all scenarios.
