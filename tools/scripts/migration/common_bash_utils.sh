#!/bin/bash

# Combine two lists and remove duplicates
function join_lists() {
  # Print both inputs, sort them, and keep only unique lines
  printf "%s\n%s" "$1" "$2" | sort | uniq
}

# Count non-empty lines in input
function count_non_empty_lines() {
  # Use grep to exclude empty lines, then count with wc
  grep -v '^$' | wc -l
}

# Convert input text to uppercase
function to_uppercase() {
  # Translate lowercase characters to uppercase
  tr '[:lower:]' '[:upper:]'
}

# Convert line-separated text to JSON array
function lines_to_json_array() {
  # Split input on newlines, remove the last empty element, and format as JSON array
  # NOTE: -1 index drops the empty value after the trailing newline.
  jq -R -s 'split("\n")[:-1] | map(.)' <<<"$1"
}

# Find lines that exist in A but not in B (set difference A - B)
function diff_A_sub_B() {
  A="$1" # First input set
  B="$2" # Second input set

  # comm compares two sorted files/streams, line by line.
  # -2: suppress lines only in B
  # -3: suppress lines common to both
  # Result: lines only in A (set difference A - B)
  comm -23 <(echo "$A") <(echo "$B")
}
