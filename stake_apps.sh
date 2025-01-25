#!/bin/bash

TOTAL_APPS=50000
PARALLEL_JOBS=8
CONFIG_DIR="localnet/poktrolld/config"
SEGMENT_SIZE=$((TOTAL_APPS / PARALLEL_JOBS))

# Function to process a segment of apps
process_segment() {
    local start=$1
    local end=$2
    local job_id=$3
    local config_file="${CONFIG_DIR}/application_stake_config.yaml"

    echo "Job $job_id staking apps $start to $end"
    for i in $(seq $start $end); do
        local app_name="app-$i"
        poktrolld tx application stake-application -y \
            --config "$config_file" \
            --keyring-backend test \
            --from "$app_name" > /dev/null 2>&1;
    done
}

export -f process_segment
export CONFIG_DIR TEMP_DIR

# Launch parallel jobs
for job_id in $(seq 0 $((PARALLEL_JOBS - 1))); do
    start=$((job_id * SEGMENT_SIZE + 1))
    end=$((start + SEGMENT_SIZE - 1))
    # Adjust last segment to include remainder
    if [ $job_id -eq $((PARALLEL_JOBS - 1)) ]; then
        end=$TOTAL_APPS
    fi
    process_segment $start $end $job_id &
done

wait

echo "Staking complete!"