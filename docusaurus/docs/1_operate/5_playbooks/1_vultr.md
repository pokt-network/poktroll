---
title: Vultr Deployment Playbook
sidebar_position: 1
---

This guide demonstrates common Vultr API operations for managing virtual machine instances via the [Vultr API](https://www.vultr.com/api).

## Table of Contents <!-- omit in toc -->

- [Prerequisites](#prerequisites)
  - [Whitelist your IP](#whitelist-your-ip)
  - [API Key](#api-key)
- [Managing Instances](#managing-instances)
  - [Create the Vultr Instance](#create-the-vultr-instance)
  - [Retrieve the Vultr Instance Configuration](#retrieve-the-vultr-instance-configuration)
  - [Environment Setup](#environment-setup)
  - [Connect to Your Instance](#connect-to-your-instance)
  - [Delete Instance](#delete-instance)
- [\[Optional\] Prepare your instance for Pocket](#optional-prepare-your-instance-for-pocket)
  - [Install `pocketd`](#install-pocketd)
  - [Import or create an account](#import-or-create-an-account)
  - [Run a full node](#run-a-full-node)
- [Additional Resources](#additional-resources)
  - [Explore Available Plans](#explore-available-plans)
  - [Explore Available Operating Systems](#explore-available-operating-systems)
  - [Additional Links](#additional-links)

## Prerequisites

### Whitelist your IP

You must whitelist your IP address with Vultr.

1. Go to the [Vultr Settings API dashboard](https://my.vultr.com/settings/#settingsapi)
2. Retrieve your `32` bit `IPV4` address by running this on your host machine:

   ```bash
   curl ifconfig.me
   ```

3. Update the `Access Control` list with `{IPv4_OUTPUT_ABOVE}/32` and click `Add`.

:::important IP Whitelist

<details>
  <summary>Screenshot Example</summary>

![Image](https://github.com/user-attachments/assets/d7b93a18-7423-43f8-adfb-bdb3bf8239ac)

</details>

:::

### API Key

Obtain your API key from [my.vultr.com/settings/#settingsapi](https://my.vultr.com/settings/#settingsapi)

```bash
export VULTR_API_KEY="your-api-key-here"
```

## Managing Instances

### Create the Vultr Instance

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
    "label" : "${REPLACE_ME_WITH_SOME_INSTANCE_NAME}",
    "os_id" : 2136,
    "backups" : "disabled",
    "hostname": "${REPLACE_ME_WITH_SOME_HOST_NAME}",
    "tags": ["personal", "test", "cli", "${REPLACE_ME_WITH_SOME_LABEL}"]
  }' \
  > vultr_create.json
```

**⚠️ Update all the params starting with `REPLACE_ME_` above ⚠️**

### Retrieve the Vultr Instance Configuration

Check the instance status at [my.vultr.com/subs/?id=VULTR_INSTANCE_ID](https://my.vultr.com/subs/?id=VULTR_INSTANCE_ID).

```bash
export VULTR_INSTANCE_ID=$(cat vultr_create.json | jq -r '.instance.id')

echo "##############\nVisit your instance at https://my.vultr.com/subs/?id=${VULTR_INSTANCE_ID} \n##############"
```

And get the instance details:

```bash
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

Using the password is in `vultr_create.json` under `instance.default_password`.

To copy password to clipboard:

```bash
cat vultr_create.json | jq -r '.instance.default_password' | pbcopy
```

### Delete Instance

```bash
curl "https://api.vultr.com/v2/instances/${VULTR_INSTANCE_ID}" \
  -X DELETE \
  -H "Authorization: Bearer ${VULTR_API_KEY}"
```

## [Optional] Prepare your instance for Pocket

### Install `pocketd`

```bash
curl -sSL https://raw.githubusercontent.com/pokt-network/poktroll/main/tools/scripts/pocketd-install.sh | bash -s -- --tag v0.1.12-dev1 --upgrade
```

### Import or create an account

Export a key from your local machine:

```bash
pkd keys export {key-name} --unsafe --unarmored-hex
```

And import it into your instance:

```bash
pocket keys import {key-name} {hex-priv-key}
```

Or create a new one:

```bash
pocket keys add {key-name}
```

### Run a full node

See the instructions in the [full node cheatsheet](../1_cheat_sheets/2_full_node_cheatsheet.md).

## Additional Resources

### Explore Available Plans

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

### Explore Available Operating Systems

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

### Additional Links

- Vultr API Documentation: [vultr.com/api/](https://www.vultr.com/api)
- Vultr CLI GitHub Repository: [github.com/vultr/vultr-cli](https://github.com/vultr/vultr-cli)
