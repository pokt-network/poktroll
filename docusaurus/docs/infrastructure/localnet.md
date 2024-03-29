---
sidebar_position: 1
title: LocalNet
---

# LocalNet <!-- omit in toc -->

<!--
  TODO_IMPROVE(@olshansk, @okdas):
  - Add a video showing how to use & run LocalNet
  - Add a component diagram outlining the infrastructure
  -  -->

## Background <!-- omit in toc -->

This document walks you through launching a LocalNet that brings up a k8s cluster
with a Data Availability network, a validator, Pocket actors and everything else
needed to send an end-to-end relay.

- [Run Poktroll locally](#run-poktroll-locally)
  - [Report issues](#report-issues)
  - [TL;DR](#tldr)
- [Developing with LocalNet](#developing-with-localnet)
  - [localnet_config.yaml](#localnet_configyaml)
  - [Scaling network actors](#scaling-network-actors)
  - [Modify Kubernetes workloads](#modify-kubernetes-workloads)
- [Troubleshooting](#troubleshooting)
  - [Clean Slate (Nuclear Option)](#clean-slate-nuclear-option)

## Run Poktroll locally

### Report issues

If you encounter any problems, please create a new [GitHub Issue here](https://github.com/pokt-network/pocket/issues/new/choose).

### TL;DR

1. Install dependencies:
   1. [Ignite](https://docs.ignite.com/welcome/install)
   2. [Docker](https://docs.docker.com/engine/install/)
   3. [Kind](https://kind.sigs.k8s.io/#installation-and-usage)
   4. [Helm](https://helm.sh/docs/intro/install/#through-package-managers)
   5. [Tilt](https://docs.tilt.dev/install.html) (note: we recommend using Kind cluster with Tilt)
2. Run `make localnet_up` to start the network
3. When prompted, click `space` to see the web UI with logs and current status of the network. Alternatively, you can go directly to [localhost:10350](http://localhost:10350)

## Developing with LocalNet

### localnet_config.yaml

Once LocalNet is started, a new file `localnet_config.yaml` is generated in the root directory of the repository. This file contains the configuration of the network. It looks like this:

```yaml
helm_chart_local_repo:
  enabled: false
  path: ../helm-charts
relayers:
  count: 1
```

### Scaling network actors

To scale the number of actors, edit the `localnet_config.yaml` file and change the `count` of the relayers.

For example:

```diff
helm_chart_local_repo:
  enabled: false
  path: ../helm-charts
relayers:
-   count: 1
+   count: 2
```

_NOTE: You may need to up to 1 minute for the new actors to be registered and deployed locally._

### Modify Kubernetes workloads

The local cluster is dependant on the latest available helm charts available at
[github.com/pokt-network/helm-charts](https://github.com/pokt-network/helm-charts.git).

If you need to modify Kubernetes resources in your local setup, follow these steps.

Clone the helm charts locally:

TODO_TECHDEBT(@okdas): Embed `helm-charts` in this repo so we are always using a
local version.

```bash
cd .. && git clone git@github.com:pokt-network/helm-charts.git
cd -
```

Update the path of the helm charts to the **relative** path of the cloned repository:

```bash
sed -i'' -e '/helm_chart_local_repo:/,/path:/ s|\(path: \).*|\1../helm-charts|' localnet_config.yaml
```

Set the `helm_chart_local_repo.enabled` flag in `localnet_config.yaml` to `true`:

```bash
sed -i'' -e '/helm_chart_local_repo:/,+1 s/\(enabled: \)true/\1false/; /helm_chart_local_repo:/,+1 s/\(enabled: \)false/\1true/' localnet_config.yaml
```

You can always set it back to `false` with:

```bash
sed -i'' -e '/helm_chart_local_repo:/,+1 s/\(enabled: \)false/\1true/; /helm_chart_local_repo:/,+1 s/\(enabled: \)true/\1false/' localnet_config.yaml
```

## Troubleshooting

### Clean Slate (Nuclear Option)

If you're encountering weird issues and just need to start over, follow these steps:

```bash
make localnet_down
kind delete cluster
make docker_wipe
make go_develop_and_test
kind create cluster
make localnet_up
```
