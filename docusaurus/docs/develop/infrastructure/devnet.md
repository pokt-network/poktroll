---
sidebar_position: 3
title: DevNet
---

# DevNet <!-- omit in toc -->

:::note

This page is only relevant to you if you are part of the core protocol team at Grove.

:::

- [](#)

## GCP Console

TODO_IN_THIS_PR: Need to link and add screenshot of the infra

https://console.cloud.google.com/kubernetes/workload/overview?project=protocol-us-central1-d505&pageState=(%22savedViews%22:(%22i%22:%22a39690ef57a74a59b7550d42ac7655bc%22,%22c%22:%5B%5D,%22n%22:%5B%22devnet-issue-152%22%5D))

## Grafana logs

TODO_IN_THIS_PR: Need to link and add screenshot of the infra.

https://grafana.poktroll.com/explore?schemaVersion=1&panes=%7B%22TtK%22%3A%7B%22datasource%22%3A%22P8E80F9AEF21F6940%22%2C%22queries%22%3A%5B%7B%22refId%22%3A%22A%22%2C%22expr%22%3A%22%7Bcontainer%3D%5C%22poktrolld%5C%22%2C+namespace%3D%5C%22devnet-issue-477%5C%22%7D+%7C%3D+%60%60+%7C+json%22%2C%22queryType%22%3A%22range%22%2C%22datasource%22%3A%7B%22type%22%3A%22loki%22%2C%22uid%22%3A%22P8E80F9AEF21F6940%22%7D%2C%22editorMode%22%3A%22builder%22%7D%5D%2C%22range%22%3A%7B%22from%22%3A%221713896821855%22%2C%22to%22%3A%221713897121855%22%7D%2C%22panelsState%22%3A%7B%22logs%22%3A%7B%22id%22%3A%22A_1713897061114756750_173e5f8d%22%2C%22visualisationType%22%3A%22logs%22%7D%7D%7D%7D&orgId=1

## Rough Notes

TODO_IN_THIS_PR: Exaplin this part.

- GitHub CI -> k8s job -> pod where test runs
- workloads created independent of CI
- When do workloads close? When PR closes
- Workloads are persistent as long as PR lives
- `devnet` -> creates workloads
  When we push a new commit? WHat happens?
- Update sha
- Push new images
- Workloads update
