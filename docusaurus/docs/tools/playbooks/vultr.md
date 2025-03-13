---
title: Vultr Deployment
sidebar_position: 1
---

## Vultr API Quick Start Guide <!-- omit in toc -->

This guide demonstrates common Vultr API operations for managing virtual machine instances.

- [Prerequisites](#prerequisites)
  [OPTIONAL: Exploring Available Resources](#optional-exploring-available-resources)
  - [List Available Plans](#list-available-plans)
  - [List Available Operating Systems](#list-available-operating-systems)
- [Environment Setup](#environment-setup)
  [Creating an Instance](#managing-instances)
- [Managing Instances](#managing-instances)
  - [Get Instance Details](#get-instance-details)
  - [Connect to Your Instance](#connect-to-your-instance)
  - [Delete Instance](#delete-instance)
- [Resources](#resources)

## Prerequisites

```bash
export VULTR_API_KEY="your-api-key-here"
```

Obtain your API key from [my.vultr.com/settings/#settingsapi](https://my.vultr.com/settings/#settingsapi)

## [OPTIONAL] Exploring Available Resources

### List Available Plans

Get the list of available plans:

```bash
curl "https://api.vultr.com/v2/plans" \
  -X GET \
  -H "Authorization: Bearer ${VULTR_API_KEY}" \
  > vultr_plans.json
```

And explore the JSON by running:

```bash
cat vultr_plans.json | jq
```

### List Available Operating Systems

Get the list of available operating systems:

```bash
curl "https://api.vultr.com/v2/os" \
  -X GET \
  -H "Authorization: Bearer ${VULTR_API_KEY}" \
  > vultr_os.json
```

And explore the JSON by running:

```bash
cat vultr_os.json | jq
```

## Managing Instances

### Creating an Instance

The command below creates a new instance with the following parameters:

- `plan vc2-6c-16gb`:
- `os_id 2136`: Debian 12 x64

```bash
curl "https://api.vultr.com/v2/instances" \
  -X POST \
  -H "Authorization: Bearer ${VULTR_API_KEY}" \
  -H "Content-Type: application/json" \
  --data '{
    "region" : "sea",
    "plan" : "vc2-6c-16gb",
    "label" : "YOUR_INSTANCE_NAME",
    "os_id" : 2136,
    "backups" : "disabled",
    "hostname": "YOUR_HOST_NAME",
    "tags": ["personal", "test", "cli", "YOUR_HOST_NAME"]
  }' \
  > vultr_create.json
```

Make sure to replace the following placeholders:

- `YOUR_INSTANCE_NAME`
- `YOUR_HOST_NAME`

### Get Instance Details

```bash
VULTR_INSTANCE_ID=$(cat vultr_create.json | jq -r '.instance.id')

curl "https://api.vultr.com/v2/instances/${VULTR_INSTANCE_ID}" \
  -X GET \
  -H "Authorization: Bearer ${VULTR_API_KEY}" \
  > vultr_get.json
```

### Environment Setup

Once you've created and retrieved your instance details, you can set up your environment variables for easier management.

```bash
export VULTR_INSTANCE_ID=$(cat vultr_create.json | jq -r '.instance.id')
export VULTR_INSTANCE_IP=$(cat vultr_get.json | jq -r '.instance.main_ip')
export VULTR_PASSWORD=$(cat vultr_create.json | jq -r '.instance.default_password')
```

### Connect to Your Instance

Connect to your instance:

```bash
ssh root@$VULTR_INSTANCE_IP
```

Password is in `vultr_create.json` under `instance.default_password`. To copy password to clipboard:

```bash
cat vultr_create.json | jq -r '.instance.default_password' | pbcopy
```

### Delete Instance

```bash
curl "https://api.vultr.com/v2/instances/${VULTR_INSTANCE_ID}" \
  -X DELETE \
  -H "Authorization: Bearer ${VULTR_API_KEY}"
```

## Resources

API Documentation: [www.vultr.com/api/#tag/instances](https://www.vultr.com/api/#tag/instances)
