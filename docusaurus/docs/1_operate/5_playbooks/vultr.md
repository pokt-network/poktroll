---
title: Vultr Deployment Playbook
sidebar_position: 1
---

## Vultr Deployment Playbook <!-- omit in toc -->

This guide demonstrates common Vultr API operations for managing virtual machine instances via the [Vultr API](https://www.vultr.com/api).

- [Prerequisites](#prerequisites)
  - [Whitelist your IP](#whitelist-your-ip)
  - [API Key](#api-key)
- [Managing Instances](#managing-instances)
  - [Creating an Instance](#creating-an-instance)
  - [Get Instance Details](#get-instance-details)
  - [Environment Setup](#environment-setup)
  - [Connect to Your Instance](#connect-to-your-instance)
  - [Delete Instance](#delete-instance)
- [\[OPTIONAL\] Exploring Available Resources](#optional-exploring-available-resources)
  - [List Available Plans](#list-available-plans)
  - [List Available Operating Systems](#list-available-operating-systems)
- [Additional Resources](#additional-resources)

## Prerequisites

### Whitelist your IP

You must whitelist your IP address with Vultr.

1. Go to the [Vultr Settings API dashboard](https://my.vultr.com/settings/#settingsapi)
2. Retrieve your `32` bit `IPV4` address by running this on your host machine:

   ```bash
   curl ifconfig.me
   ```

3. Update the `Access Control` list with `{IPv4_OUTPUT_ABOVE}/32` and click `Add`.

<details>
  <summary>Screenshot Example</summary>

![Image](https://github.com/user-attachments/assets/d7b93a18-7423-43f8-adfb-bdb3bf8239ac)

</details>

### API Key

Obtain your API key from [my.vultr.com/settings/#settingsapi](https://my.vultr.com/settings/#settingsapi)

```bash
export VULTR_API_KEY="your-api-key-here"
```

:::important IP Whitelist

:::

## Managing Instances

### Creating an Instance

:::warning Update command

Make sure to replace the following placeholders:

- `YOUR_INSTANCE_NAME`
- `YOUR_HOST_NAME`
- Optionally, list of tags

:::

The command below creates a new instance with the following parameters:

- **plan** `vc2-6c-16gb`: 6 vCPUs w/ 16GB RAM and 320GB SSD
- **os_id** `2136`: Debian 12 x64
- **region** `sea`: Seattle, WA, USA

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

### Get Instance Details

```bash
VULTR_INSTANCE_ID=$(cat vultr_create.json | jq -r '.instance.id')

curl "https://api.vultr.com/v2/instances/${VULTR_INSTANCE_ID}" \
  -X GET \
  -H "Authorization: Bearer ${VULTR_API_KEY}" \
  > vultr_get.json

echo "###\nVisit your instance at https://my.vultr.com/subs/?id=${VULTR_INSTANCE_ID} \n###\n"
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

## Additional Resources

- Vultr API Documentation: [vultr.com/api/](https://www.vultr.com/api)
- Vultr CLI GitHub Repository: [github.com/vultr/vultr-cli](https://github.com/vultr/vultr-cli)
