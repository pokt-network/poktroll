#!/bin/bash

TOTAL_APPS=50000
PARALLEL_JOBS=32
OUTPUT_FILE="app_addresses.txt"
TEMP_DIR=/tmp/addrs
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

    echo "Job $job_id processing apps $start to $end"
    for i in $(seq $start $end); do
        local app_name="app-$i"
        local addr=$(poktrolld keys show $app_name -a --keyring-backend test)
        echo "$addr" >> "$output"
    done
}

export -f process_segment
export TEMP_DIR

# Launch parallel jobs
for job_id in $(seq 0 $((PARALLEL_JOBS - 1))); do
    start=$((job_id * SEGMENT_SIZE + 1))
    end=$((start + SEGMENT_SIZE - 1))
    # Adjust last segment to include remainder
    if [ $job_id -eq $((PARALLEL_JOBS - 1)) ]; then
        end=$TOTAL_APPS
    fi
    echo "Launching job $job_id for apps $start to $end"
    parallel -j1 process_segment ::: $start ::: $end ::: $job_id &
done

wait

# Combine results in order
for job_id in $(seq 0 $((PARALLEL_JOBS - 1))); do
    cat "$TEMP_DIR/segment_$job_id.txt"
done > $OUTPUT_FILE

echo "Generated addresses for $TOTAL_APPS apps in $OUTPUT_FILE"