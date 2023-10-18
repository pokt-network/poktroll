# Poktroll LocalNet <!-- omit in toc -->

<!-- TODO(@olshansk, @okdas): Add a video showing how to use & run LocalNet. -->

## Background <!-- omit in toc -->

Poktroll comes with a LocalNet that can be used for development and testing on a local machine. As a rollup, it requires an underlying Data Availability layer, which is provisioned by the locally running celestia node.

## Table of Contents <!-- omit in toc -->

- [Run Poktroll locally](#run-poktroll-locally)
  - [Report issues](#report-issues)
  - [TL;DR](#tldr)
- [Develop on the LocalNet](#develop-on-the-localnet)
  - [Scaling network actors](#scaling-network-actors)
  - [Modify Kubernetes workloads](#modify-kubernetes-workloads)

## Run Poktroll locally

### Report issues

If you encounter a problem using this guide, please create a new [GitHub Issue](https://github.com/pokt-network/pocket/issues/new/choose).

### TL;DR

1. Install dependencies:
   1. [Ignite](https://docs.ignite.com/welcome/install)
   2. [Docker](https://docs.docker.com/engine/install/)
   3. [Kind](https://kind.sigs.k8s.io/#installation-and-usage)
   4. [Helm](https://helm.sh/docs/intro/install/#through-package-managers)
   5. [Tilt](https://docs.tilt.dev/install.html) (note: we recommend using Kind cluster with Tilt)
2. Run `make localnet_up` to start the network
3. When prompted, click `space` to see the web UI with logs and current status of the network. Alternatively, you can go directly to [localhost:10350](http://localhost:10350)

## Develop on the LocalNet

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

If you need to modify Kubernetes resources, follow these steps:

1. Clone the [helm-charts](https://github.com/pokt-network/helm-charts) repository.
2. In `localnet_config.yaml`, set `helm_chart_local_repo.enabled` to `true` and `path` to the **relative** path of the cloned repository.

The following is an example that has not been tested yet:

```bash
cd ~/src/pocket
git clone git@github.com:pokt-network/helm-charts.git

cd ~/src/pocket/poktroll
sed -i.bak "s/helm_chart_local_repo\.enabled: false.*/helm_chart_local_repo.enabled: true/" localnet_config.yaml
sed -i.bak "s#path: .*#path: ../helm-charts#" localnet_config.yaml
```
