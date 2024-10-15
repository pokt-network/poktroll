---
title: Consensus failure recovery plan
sidebar_position: 6
---

# Consensus Failure Recovery Plan



## Common consensus failure errors



- `wrong Block.Header.AppHash` - the data in block is different between nodes. Can be investigated by comparing the data dir - [more information here](../../develop/developer_guide/chain_halt_troubleshooting.md).

