---
sidebar_position: 3
title: DevNet
---

# DevNet <!-- omit in toc -->

:::note

This page is only relevant to you if you are part of the core protocol team at Grove.

Make sure to follow the instructions [here](https://www.notion.so/buildwithgrove/Infrastructure-Setup-79b1431b70374e24b10cd9da556c7645?pvs=4) if you are unsure how
to set up your access to GCP.

:::

## Table of Contents <!-- omit in toc -->

- [GCP Console](#gcp-console)
- [Grafana logs](#grafana-logs)
- [Infrastructure Provisioning](#infrastructure-provisioning)
- [Configuration](#configuration)

## GCP Console

As an example, this [GCP link](<https://console.cloud.google.com/kubernetes/workload/overview?project=protocol-us-central1-d505&pageState=(%22savedViews%22:(%22i%22:%22a39690ef57a74a59b7550d42ac7655bc%22,%22c%22:%5B%5D,%22n%22:%5B%22devnet-issue-498%22%5D))>) links to `devnet-issue-498` created
from [PR #498](https://github.com/pokt-network/poktroll/pull/498).

![GCP console](./gcp_workloads.png)

## Grafana logs

As an example, this [Grafana link](https://grafana.poktroll.com/explore?schemaVersion=1&panes=%7B%22TtK%22:%7B%22datasource%22:%22P8E80F9AEF21F6940%22,%22queries%22:%5B%7B%22refId%22:%22A%22,%22expr%22:%22%7Bcontainer%3D%5C%22poktrolld%5C%22,%20namespace%3D%5C%22devnet-issue-477%5C%22%7D%20%7C%3D%20%60%60%20%7C%20json%22,%22queryType%22:%22range%22,%22datasource%22:%7B%22type%22:%22loki%22,%22uid%22:%22P8E80F9AEF21F6940%22%7D,%22editorMode%22:%22builder%22%7D%5D,%22range%22:%7B%22from%22:%22now-1h%22,%22to%22:%22now%22%7D,%22panelsState%22:%7B%22logs%22:%7B%22logs%22:%7B%22visualisationType%22:%22logs%22%7D%7D%7D%7D%7D&orgId=1) links to the logs for `devnet-issue-477` from [PR #477](https://github.com/pokt-network/poktroll/pull/477)

![Grafana logs explorer](./grafana_explore_logs.png)

## Infrastructure Provisioning

The following is a list of details to know how our DevNet infrastructure is provisioned:

- **Grove & Kubernetes**: The Kubernetes cluster is provisioned using Grove's internal tooling.
- **Main Cluster**: We set up ArgoCD on the cluster and configure it to sync the [main/root application on the cluster](https://github.com/pokt-network/protocol-infra/blob/main/clusters/protocol-us-central1-app.yaml).
- **App of Apps**: ArgoCD provisions all the necessary resources and other ArgoCD Applications included in that Application, following the [ArgoCD App of Apps pattern](https://argo-cd.readthedocs.io/en/stable/operator-manual/cluster-bootstrapping/).
- **PR `devnet` label**: One of the manifests provisioned is an [ArgoCD ApplicationSet](https://github.com/pokt-network/protocol-infra/blob/main/clusters/protocol-us-central1/devnets-github-label.yaml), which monitors our GitHub labels and provisions a network for each GitHub issue tagged with the `devnet` label.
- **PR `devnet-test-e2e` label**: As part of our CI process, when a GitHub issue is labeled `devnet-test-e2e`, we execute a [script](https://github.com/pokt-network/poktroll/blob/main/.github/workflows-helpers/run-e2e-test.sh#L1) that creates a [Kubernetes Job](https://github.com/pokt-network/poktroll/blob/main/.github/workflows-helpers/run-e2e-test-job-template.yaml) to for that `DevNet`.
- **Ephemeral DevNet**: The DevNet deplyed using the labels described above are considered to be ephemeral.
- **Closing PR**s: When the PR is closed or the label is removed, the infrastructure is cleaned up.
- **Persistent DevNet**: We have an option to provision `persistent DevNets` by creating a DevNet yaml file in [this directory](https://github.com/pokt-network/protocol-infra/tree/main/devnets-configs) ([ArgoCD Application that monitors this directory](https://github.com/pokt-network/protocol-infra/blob/main/clusters/protocol-us-central1/devnets-persistent.yaml) for reference).

## Configuration

Each DevNet ArgoCD App (following the App of Apps pattern) provisions a Helm chart called [full-network](https://github.com/pokt-network/protocol-infra/tree/main/charts/full-network).

Each `full-network` includes other ArgoCD applications that deploy Validators and off-chain actors.

Each Helm chart receives a list of configuration files. For example, see the [relayminer configuration](https://github.com/pokt-network/protocol-infra/blob/main/charts/full-network/templates/Application-Relayminer.yaml#L37). All possible values can be found in the `values.yaml` of the Helm chart, such as the [relayminer Helm chart](https://github.com/pokt-network/helm-charts/blob/main/charts/relayminer/values.yaml).
