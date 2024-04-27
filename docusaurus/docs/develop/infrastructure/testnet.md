---
sidebar_position: 4
title: TestNet
---

# TestNet <!-- omit in toc -->

## Infrastructure provisioning

- K8s cluster is provisioned by Grove internal tooling;
- We set up ArgoCD on the cluster and configure it to sync the [main/root application on the cluster](https://github.com/pokt-network/protocol-infra/blob/main/clusters/protocol-us-central1-app.yaml).
- ArgoCD, using this ArgoCD Application, provisions all the resources and other ArgoCD Applications that are included with that ArgoCD Application. This approach follows [ArgoCD App of Apps pattern](https://argo-cd.readthedocs.io/en/stable/operator-manual/cluster-bootstrapping/).
- As a part of that ArgoCD Application we have resources such as StatefulSets and COnfigMaps that describe configuration and infrastructure to run validators and seed nodes. Examples:
  - https://github.com/pokt-network/protocol-infra/blob/main/clusters/protocol-us-central1/testnet-validated.yaml
  - https://github.com/pokt-network/protocol-infra/blob/main/clusters/protocol-us-central1/testnet-validated-seed.yaml
  <!-- btw I'm going to change that as I don't really like that set up. We need to add an abstraction here. -->