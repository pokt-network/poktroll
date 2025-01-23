#!/bin/bash

TOTAL_ACCOUNTS=50000
PARALLEL_JOBS=8
ACCOUNTS_PER_JOB=$((TOTAL_ACCOUNTS / PARALLEL_JOBS))

create_accounts() {
    local start=$1
    local end=$2
    local job_id=$3

    for i in $(seq $start $end); do
        if ! poktrolld keys add "app-$i" > /dev/null 2>&1; then
            echo "Job $job_id: Error creating account app-$i"
            continue
        fi

        if [ $((i % 100)) -eq 0 ]; then
            echo "Job $job_id: Progress $i/$end accounts created"
        fi
    done
}

echo "Starting parallel account creation with $PARALLEL_JOBS jobs..."

# Launch parallel jobs
for job in $(seq 0 $((PARALLEL_JOBS-1))); do
    start=$((job * ACCOUNTS_PER_JOB + 1))
    if [ $job -eq $((PARALLEL_JOBS-1)) ]; then
        end=$TOTAL_ACCOUNTS
    else
        end=$((start + ACCOUNTS_PER_JOB - 1))
    fi

    create_accounts $start $end $job &
done

# Wait for all background jobs to complete
wait

echo "All account creation jobs completed!"