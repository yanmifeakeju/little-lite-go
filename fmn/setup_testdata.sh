#!/bin/bash
#
# This script sets up a predictable directory structure within the 'testdata'
# directory, which can be used for running manual or automated tests for the
# 'fmn' tool.
#
# It creates a 'source' directory with a mix of files and subdirectories,
# and an empty 'destination' directory.

# --- Configuration ---
# Using variables makes the script easier to read and modify.
BASE_DIR="testdata"
SOURCE_DIR="$BASE_DIR/source"
DEST_DIR="$BASE_DIR/destination"
SUB_DIR_A="$SOURCE_DIR/dir1"
SUB_DIR_B="$SOURCE_DIR/dir2"

# --- Cleanup ---
# Always start from a clean slate. The '-f' flag prevents errors if the
# directories don't exist.
echo "Cleaning up old test data..."
rm -rf "$SOURCE_DIR" "$DEST_DIR"

# --- Directory Creation ---
# The '-p' flag tells 'mkdir' to create parent directories if they don't
# already exist. This makes the script more robust.
echo "Creating directory structure..."
mkdir -p "$SUB_DIR_A"
mkdir -p "$SUB_DIR_B"
mkdir -p "$DEST_DIR"

# --- File Creation ---
# Create a variety of files to test different scenarios.
echo "Creating test files..."
echo "This is the root file." > "$SOURCE_DIR/root_file.txt"
echo "This is in the first subdirectory." > "$SUB_DIR_A/sub_file_a.txt"
touch "$SUB_DIR_B/empty_file.txt" # An empty file

echo "Test data setup complete."
echo "Source:      $SOURCE_DIR"
echo "Destination: $DEST_DIR"
