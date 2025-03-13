

https://www.vultr.com/api/#tag/instances

curl "https://api.vultr.com/v2/plans" \
  -X GET \
  -H "Authorization: Bearer ${VULTR_API_KEY}"

curl "https://api.vultr.com/v2/os" \
  -X GET \
  -H "Authorization: Bearer ${VULTR_API_KEY}"

1. https://my.vultr.com/settings/#settingsapi
2.

# ,{"id":2136,"name":"Debian 12 x64 (bookworm}

# vc2-6c-16gb
# Seattle
# IP Address:
# 45.32.230.119
# Username:
# root
# Password:
# •••••••
# vCPU/s:
# 6 vCPUs
# RAM:
# 16384.00 MB
# Storage:

 curl "https://api.vultr.com/v2/instances" \
  -X POST \
  -H "Authorization: Bearer ${VULTR_API_KEY}" \
  -H "Content-Type: application/json" \
  --data '{
    "region" : "sea",
    "plan" : "vc2-6c-16gb",
    "label" : "olshansky-delete-me",
    "os_id" : 2136,
    "backups" : "disabled",
    "hostname": "olshansky",
    "tags": ["olshansky", "personal", "test", "cli", "full-node"]
  }' >> create.json

{"instance":{"id":"7efe0354-5941-4b30-b259-b5d530a5d1e0","os":"Debian 11 x64 (bullseye)","ram":4096,"disk":0,"main_ip":"0.0.0.0","vcpu_count":2,"region":"sea","plan":"vc2-2c-4gb","date_created":"2025-03-13T01:57:21+00:00","status":"pending","allowed_bandwidth":4,"netmask_v4":"","gateway_v4":"0.0.0.0","power_status":"running","server_status":"none","v6_network":"","v6_main_ip":"","v6_network_size":0,"label":"olshansky-delete-me","internal_ip":"","kvm":"","hostname":"olshansky","tag":"personal","tags":["personal","test","cli","full-node"],"os_id":477,"app_id":0,"image_id":"","firewall_group_id":"","features":[],"user_scheme":"root","pending_charges":0,"default_password":"@5yZp%k[]%?pn+M5"}}%




3. Go to https://my.vultr.com/subs/?id=7efe0354-5941-4b30-b259-b5d530a5d1e0

https://my.vultr.com/subs/?id=7efe0354-5941-4b30-b259-b5d530a5d1e0

4.

https://my.vultr.com/subs/?id=7efe0354-5941-4b30-b259-b5d530a5d1e0


VULTR_INSTANCE_ID=$(cat create.json | jq -r '.instance.id')

curl "https://api.vultr.com/v2/instances/${VULTR_INSTANCE_ID}" \
  -X GET \
  -H "Authorization: Bearer ${VULTR_API_KEY}" > get.json

5.



VULTR_INSTANCE_IP=$(cat get.json | jq -r '.instance.main_ip')
ssh root@$VULTR_INSTANCE_IP
# password is in create.jsonf

6. delete

curl "https://api.vultr.com/v2/instances/${VULTR_INSTANCE_ID-id}" \
  -X DELETE \
  -H "Authorization: Bearer ${VULTR_API_KEY}"