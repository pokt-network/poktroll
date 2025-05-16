---
title: Migration FAQ
sidebar_position: 9
---

## What happens if I have a Morse node that is both a Validator and a Supplier?

During the Shannon migration, the node will **only** be claimed as a **Supplier**—which is equivalent to a **Servicer** in Morse. The Validator role will **not** carry over.

Validators are handled separately. See [Claiming Morse Validators](./9_claiming_validator.md) for more details on validators.
