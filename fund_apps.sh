#!/bin/bash

TOTAL_APPS=50000
AMOUNT="100500000upokt"
FAUCET_ACCOUNT="faucet"
MSGS_PER_TX=5000
ADDRESSES_FILE="app_addresses.txt"

echo "Starting funding of $TOTAL_APPS applications (Messages per TX: $MSGS_PER_TX)"

# Function to wait for next block
wait_for_next_block() {
    current_height=$(poktrolld query block --output json | sed '1d' | jq -r .header.height)
    target_height=$((current_height + 1))

    echo "Waiting for block $target_height..."
    while true; do
        new_height=$(poktrolld query block --output json | sed '1d' | jq -r .header.height)
        if [ "$new_height" -ge "$target_height" ]; then
            break
        fi
        sleep 1
    done
}

# Process in batches
for ((i=1; i<=TOTAL_APPS; i+=MSGS_PER_TX)); do
    batch_end=$((i + MSGS_PER_TX - 1))
    [ $batch_end -gt $TOTAL_APPS ] && batch_end=$TOTAL_APPS

    echo "Processing batch $i to $batch_end"

    # Generate messages for this batch
    ACCOUNTS=""
    for j in $(seq $i $batch_end); do
        APP_ADDRESS=$(sed -n "${j}p" $ADDRESSES_FILE)
        if [ $j -eq $i ]; then
            ACCOUNTS="$APP_ADDRESS"
        else
            ACCOUNTS="$ACCOUNTS $APP_ADDRESS"
        fi
    done


    # Create and broadcast multi-msg transaction
    poktrolld tx bank multi-send $FAUCET_ACCOUNT $ACCOUNTS $AMOUNT \
        --from $FAUCET_ACCOUNT \
        --chain-id poktroll \
        --keyring-backend test \
        --gas auto \
        --gas-adjustment 1.5 \
        --gas-prices 0.025upokt \
        -y

    echo "Batch $((i / MSGS_PER_TX + 1)) complete"
    wait_for_next_block
done

echo "All transactions submitted!"