---
sidebar_position: 4
title: TestNet
---

# TestNet <!-- omit in toc -->

:::note

This page is only relevant to you if you are part of the core protocol team at Grove.

:::

## Table of Contents <!-- omit in toc -->

- [Infrastructure provisioning](#infrastructure-provisioning)
- [Version upgrade](#version-upgrade)
- [Regenesis procedure](#regenesis-procedure)

## Infrastructure provisioning

- **Grove & Kubernetes**: The Kubernetes cluster is provisioned using Grove's internal tooling.
- **Main Cluster**: We set up ArgoCD on the cluster and configure it to sync the [main/root application on the cluster](https://github.com/pokt-network/protocol-infra/blob/main/clusters/protocol-us-central1-app.yaml).
- **App of Apps**: ArgoCD provisions all the necessary resources and other ArgoCD Applications included in that Application, following the [ArgoCD App of Apps pattern](https://argo-cd.readthedocs.io/en/stable/operator-manual/cluster-bootstrapping/).
- As a part of that ArgoCD Application we have resources such as StatefulSets and ConfigMaps that describe configuration and infrastructure to run validators and seed nodes. Examples:
  - [testnet-validated.yaml](https://github.com/pokt-network/protocol-infra/blob/main/clusters/protocol-us-central1/testnet-validated.yaml)
  - [testnet-validated-seed.yaml](https://github.com/pokt-network/protocol-infra/blob/main/clusters/protocol-us-central1/testnet-validated-seed.yaml)
  <!-- TODO_DOCUMENT(@okdas): improve the setup because this requires an abstraction. -->

## Version upgrade

The notion doc on how to upgrade Grove's validator can be found [here](https://www.notion.so/How-to-upgrade-validator-seed-node-ee85c4de651047f29151c0c51cd8f14a?pvs=4)

## Regenesis procedure

- [Genesis generation notion doc](https://www.notion.so/Generating-a-new-genesis-json-file-b6a41c010a114713b6b0cdc2ebb6e264?pvs=4)
- [Step-by-step guide to do a full re-genesis](https://www.notion.so/How-to-re-genesis-a-Shannon-TestNet-a6230dd8869149c3a4c21613e3cfad15?pvs=4)
