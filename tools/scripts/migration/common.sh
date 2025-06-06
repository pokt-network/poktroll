#!/bin/bash

PRINT_COUNTS=false
TESTNET=false

# Parse args
function parse_args() {
  for arg in "$@"; do
    case $arg in
      --count)
        PRINT_COUNTS=true
        shift
        ;;
      --testnet)
        TESTNET=true
        shift
        ;;
      *)
        ;;
    esac
  done
}

function join_lists() {
  printf "%s\n%s" "$1" "$2" | sort | uniq
}

function count_non_empty_lines() {
  grep -v '^$' | wc -l
}

function to_uppercase() {
  tr '[:lower:]' '[:upper:]'
}

function lines_to_json_array() {
  # NOTE: -1 index drops the empty value after the trailing newline.
  jq -R -s 'split("\n")[:-1] | map(.)' <<< "$1"
}

function diff_A_sub_B() {
  A="$1"
  B="$2"

  # comm compares two sorted files/streams, line by line.
  # -2: suppress lines only in B
  # -3: suppress lines common to both
  # Result: lines only in A
  comm -23 <(echo "$A") <(echo "$B")
}
