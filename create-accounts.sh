#!/bin/bash

TOTAL_ADDRESSES=${1:-50000}
PARALLEL_JOBS=${2:-8}
SEGMENTS_DIR="./segments"
OUTPUT_FILE="app_addresses.txt"

create_segment() {
    local job_id=$1
    local start_idx=$2
    local num_addresses=$3
    local segment_file="$SEGMENTS_DIR/segment_$job_id.txt"

    > "$segment_file"
    for i in $(seq 0 $(($num_addresses-1))); do
        addr_idx=$(($start_idx + $i + 1))
        output=$(poktrolld keys add "app-$addr_idx" --output json | jq -r .address 2>&1)
        echo "$output" >> "$segment_file"
    done
}

merge_segments() {
    > "$OUTPUT_FILE"
    for i in $(seq 0 $((PARALLEL_JOBS-1))); do
        if [ -f "$SEGMENTS_DIR/segment_$i.txt" ]; then
            cat "$SEGMENTS_DIR/segment_$i.txt" >> "$OUTPUT_FILE"
        else
            echo "Missing segment file: segment_$i.txt" >&2
            return 1
        fi
    done
}

main() {
    rm -rf $SEGMENTS_DIR
    mkdir -p $SEGMENTS_DIR

    ADDRS_PER_JOB=$(( (TOTAL_ADDRESSES + PARALLEL_JOBS - 1) / PARALLEL_JOBS ))
    echo "Creating $TOTAL_ADDRESSES addresses using $PARALLEL_JOBS parallel jobs"

    for job_id in $(seq 0 $((PARALLEL_JOBS-1))); do
        start_idx=$((job_id * ADDRS_PER_JOB))
        create_segment "$job_id" "$start_idx" "$ADDRS_PER_JOB" &
    done

    wait
    merge_segments
    rm -rf $SEGMENTS_DIR
    echo "Complete - addresses written to $OUTPUT_FILE"
}

main