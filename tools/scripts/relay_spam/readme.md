# Relay Spam

The script is designed to send requests to path instances in a way that maximizes the number of sessions - it spreads the load across all applications in the config file.

## Prerequisites

- LocalNet is running with PATH (3 relayminers, 3 path gateways).
- Accounts are initizalied and suppliers/gateways are staked. (`make acc_initialize_pubkeys && sleep 3 && make supplier2_stake supplier3_stake gateway2_stake gateway3_stake`)
- `anvil.localhost` is a part of `/etc/hosts` 

## Steps

Import accounts into the keyring:

```
./relay_spam.rb import_accounts
```

Fund accounts:  
```
./relay_spam.rb fund_accounts
```

Stake applications:
```
./relay_spam.rb stake_applications
```

Run the relay spam:
```
./relay_spam.rb run
```

```
./relay_spam.rb run --help
Usage: relay_spam.rb [options] COMMAND
    -c, --config FILE                Config file (default: config.yml)
    -n, --num-requests NUM           Number of requests (default: 1000)
    -p, --concurrency NUM            Concurrent requests (default: 50)
    -r, --rate-limit NUM             Rate limit in requests per second (optional)
```

Possible options:

- `-r` or `--rate-limit` - rate limit in requests per second (optional)
- `-c` or `--concurrency` - number of concurrent requests (optional)
- `-n` or `--num-requests` - number of requests to send (optional)


# Helpful commands
Add more accounts to the config.yml file (if you need more than already populated)
```
./relay_spam.rb populate_config_accounts -a 50
```