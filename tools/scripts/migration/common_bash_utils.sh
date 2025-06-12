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

  # Combine inputs into a single stream safely, handling:
  # - empty input
  # - no trailing newline
  # - no leading/trailing blank lines

  {
    if [[ -n "$list_a" ]]; then
      printf "%s\n" "$list_a"
    fi

    if [[ -n "$list_b" ]]; then
      printf "%s\n" "$list_b"
    fi
  } | awk 'NF' | sort | uniq
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

# Attempt to match a single file by the given glob pattern.
# Exits with non-zero status if zero or more than one file is found.
#   $1 - Glob pattern of the file name to match
match_single_file_by_glob() {
  local path_glob="$1"
  local glob_matched_files=(${path_glob})
  local num_glob_matched_files=${#glob_matched_files[@]}

  if (( $num_glob_matched_files == 0 )); then
    echo "No files matched '$path_glob'" >&2
    exit 1
  elif (( $num_glob_matched_files > 1 )); then
    echo "${num_glob_matched_files} files matched ${path_glob}:" >&2
    for file in "${glob_matched_files[@]}"; do
      echo "  - $file" >&2
    done
    exit 2
  fi

  if ! assert_file_exists "${glob_matched_files[0]}"; then
    exit $?
  fi

  echo "${glob_matched_files[0]}"
}

# Get the path to the state export file with the given height.
#   $1 - Height to look for in the state export file name
#   $2 - Directory to prepend to the state export file name (optional; default is $SCRIPT_DIR)
get_state_export_path_by_height() {
  local height="$1"
  local artifact_dir="${2:-$SCRIPT_DIR}"

  local morse_state_export_glob="${artifact_dir}/morse_state_export_${height}*.json"
  if ! match_single_file_by_glob "$morse_state_export_glob"; then
    exit $?
  fi
}

# Get the path to the import message file with the given MainNet height and optional TestNet height.
#   $1 - MainNet height to look for in the import message file name
#   $2 - TestNet height to look for in the import message file name (optional; default is "0", ignored if "0")
#   $3 - Directory to prepend to the import message file name (optional; default is $SCRIPT_DIR)
get_import_message_path_by_height() {
  local mainnet_height="$1"
  local testnet_height="${2:-"0"}"
  local artifact_dir="${3:-$SCRIPT_DIR}"

  # Default to MainNet only import message, override with MainNet + TestNet import message if testnet height is set.
  local msg_import_morse_accounts_glob="${artifact_dir}/msg_import_morse_accounts_${mainnet_height}*.json"
  if [ "$testnet_height" != "0" ]; then
    msg_import_morse_accounts_glob="${artifact_dir}/msg_import_morse_accounts_m${mainnet_height}_t${testnet_height}.json"
  fi

  if ! match_single_file_by_glob "$msg_import_morse_accounts_glob"; then
    exit $?
  fi
}
