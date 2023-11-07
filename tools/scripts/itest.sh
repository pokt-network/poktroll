#!/usr/bin/env bash

# Set the current working directory to the repo root.
cd "$(dirname "$0")/../.." || exit 1

itest() {
    # Check if correct arguments are passed
    if [[ -z "$1" || -z "$2" || -z "$3" ]]; then
        echo "Usage (make):               make itest <go_test_count> <loop_limit> <package_path> -- [go test flags...]"
        echo "Usage (bash): ./tools/scripts/itest.sh <go_test_count> <loop_limit> <package_path> [go test flags...]"
        return 1
    fi

    local go_test_count=$1
    local loop_limit=$2
    local pkg_path=$3
    shift 3

    # TODO_HACK: this is a workaround as it seems that make is striping the
    # leading "./" from the pkg_path.
    # Check if pkg_path starts with a "." or "/" and prepend "./" if not.
    if [[ ! "$pkg_path" =~ ^(\.|/) ]]; then
        pkg_path="./$pkg_path"
    fi

    local total_tests_run=0

    trap 'echo -e "\nInterrupted. Total tests run: $total_tests_run"; return 1' SIGINT

    for i in $(seq 1 $loop_limit); do
        echo "Iteration $i of $loop_limit..."

        # Running the go test in a subshell in the background
        ( go test -count=$go_test_count -race "$@" $pkg_path; echo $?>/tmp/ttest_status; echo ""; ) &
        local test_pid=$!

        # Wait for the background test to complete
        wait $test_pid 2>/dev/null

        local test_exit_status=$(cat /tmp/ttest_status)
        rm -f /tmp/ttest_status

        total_tests_run=$((total_tests_run + go_test_count))

        # If go test fails, exit the loop.
        if [[ $test_exit_status -ne 0 ]]; then
            echo "go test failed on iteration $i; exiting early. Total tests run: $total_tests_run"
            return 1
        fi
    done

    echo "All iterations completed. Total tests run: $total_tests_run"
}

itest "$@"