# Poktroll LocalNet <!-- omit in toc -->

Poktroll comes with a LocalNet that can be used to test the changes on local machine. Poktroll, being a rollup, requires a Data Availability Layer, so we also provision a Celestia node to act as a Data Availability Layer.

- [Run Poktroll locally](#run-poktroll-locally)
  - [Report issues](#report-issues)
  - [TLDR](#tldr)
- [Develop on the LocalNet](#develop-on-the-localnet)
  - [Scale actors of the network](#scale-actors-of-the-network)
  - [Modify Kubernetes workloads](#modify-kubernetes-workloads)

## Run Poktroll locally

### Report issues

If you encounter a problem using this guide, please create an issue.

### TLDR

1. Install dependencies:
   1. [Ignite](https://docs.ignite.com/welcome/install)
   2. [Docker](https://docs.docker.com/engine/install/)
   3. [Kind](https://kind.sigs.k8s.io/#installation-and-usage)
   4. [Helm](https://helm.sh/docs/intro/install/#through-package-managers)
   5. [Tilt](https://docs.tilt.dev/install.html) (note: we recommend using Kind cluster with Tilt)
2. `make localnet_up` to start the network
3. When prompted, click `space` to see the web UI with Logs and current status of the network

## Develop on the LocalNet

After the LocalNet is started, a new file `localnet_config.yaml` gets generated in the root of the repository. This file contains the configuration of the network. It looks like this:

```yaml
helm_chart_local_repo:
  enabled: false
  path: ../helm-charts
relayers:
  count: 1
```

### Scale actors of the network

To scale the number of actors, edit the `localnet_config.yaml` file and change the `count` of the relayers.

### Modify Kubernetes workloads

If you need to modify Kubernetes resources, you need to follow the steps below:

1. Clone the [helm-charts](https://github.com/pokt-network/helm-charts) repository
2. In `localnet_config.yaml`, set `helm_chart_local_repo.enabled` to `true` and `path` to the **relative** path of the cloned repository.