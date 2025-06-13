#!/bin/bash

# Echo an error message and exit with status code 1 if the file does not exist.
#   $1 - File path
assert_file_exists() {
  local file_path="$1"

  if [ ! -f "$file_path" ]; then
    echo "Error: File '$file_path' not found" >&2
    exit 1
  fi
}

# Combine two lists of new-line delimited strings and remove duplicates
#   $1 - First string of lines
#   $2 - Second string of lines
join_lists() {
  local list_a="$1"
  local list_b="$2"

  # Print both inputs, sort them, and keep only unique lines
  printf "%s\n%s" "$list_a" "$lisb_b" | sort | uniq
}

# Count non-empty lines in input
count_non_empty_lines() {
  # Use grep to exclude empty lines, then count with wc
  grep -v '^$' | wc -l
}

# Convert input text to uppercase
to_uppercase() {
  # Translate lowercase characters to uppercase
  tr '[:lower:]' '[:upper:]'
}

# Convert line-separated text to JSON array
lines_to_json_array() {
  # Split input on newlines, remove the last empty element, and format as JSON array
  # NOTE: -1 index drops the empty value after the trailing newline.
  echo "$1" | jq -R -s 'split("\n")[:-1] | map(.)'
}

# Find lines that exist in lines_a but not in lines_b (set difference lines_a - lines_b)
diff_A_sub_B() {
  local lines_a="$1" # First input set
  local lines_b="$2" # Second input set

  # comm compares two sorted files/streams, line by line.
  # -2: suppress lines only in lines_b
  # -3: suppress lines common to both
  # Result: lines only in lines_a (set difference lines_a - lines_b)
  comm -23 <(echo "$lines_a") <(echo "$lines_b")
}
