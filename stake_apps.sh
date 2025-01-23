#!/bin/bash

TOTAL_APPS=50000
PARALLEL_JOBS=8
CONFIG_DIR="localnet/poktrolld/config"
TEMP_DIR=/tmp/stake_apps
SEGMENT_SIZE=$((TOTAL_APPS / PARALLEL_JOBS))

# Create and setup temp directory
rm -rf $TEMP_DIR
mkdir -p $TEMP_DIR
chmod 777 $TEMP_DIR
trap 'rm -rf $TEMP_DIR' EXIT

# Function to process a segment of apps
process_segment() {
    local start=$1
    local end=$2
    local job_id=$3
    local output="$TEMP_DIR/segment_$job_id.txt"
    local config_file="${CONFIG_DIR}/application_stake_config.yaml"

    echo "Job $job_id staking apps $start to $end"
    for i in $(seq $start $end); do
        local app_name="app-$i"
        if poktrolld tx application stake-application -y \
            --config "$config_file" \
            --keyring-backend test \
            --from "$app_name" > /dev/null 2>&1; then
            echo "$app_name" >> "$output.success"
        else
            echo "$app_name" >> "$output.failed"
        fi
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

# Report results
total_success=0
total_failed=0
for job_id in $(seq 0 $((PARALLEL_JOBS - 1))); do
    if [ -f "$TEMP_DIR/segment_$job_id.txt.success" ]; then
        success=$(wc -l < "$TEMP_DIR/segment_$job_id.txt.success")
        total_success=$((total_success + success))
    fi
    if [ -f "$TEMP_DIR/segment_$job_id.txt.failed" ]; then
        failed=$(wc -l < "$TEMP_DIR/segment_$job_id.txt.failed")
        total_failed=$((total_failed + failed))
        echo "Failed apps in job $job_id:"
        cat "$TEMP_DIR/segment_$job_id.txt.failed"
    fi
done

echo "Staking complete!"
echo "Successfully staked: $total_success applications"
echo "Failed: $total_failed applications"