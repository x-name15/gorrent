#!/bin/bash
# Example post-processing script for Gorrent (v1.6.0)
# This script extracts any .rar or .zip files found in the torrent download directory.

echo "--- Gorrent Post-Processing ---"
echo "Torrent: $GORRENT_NAME"
echo "Path: $GORRENT_PATH"
echo "Category: $GORRENT_CATEGORY"

if [ -z "$GORRENT_PATH" ]; then
    echo "Error: GORRENT_PATH is not set."
    exit 1
fi

cd "$GORRENT_PATH" || exit 1

# Check for unrar
if command -v unrar &> /dev/null; then
    for rar_file in *.rar; do
        if [ -f "$rar_file" ]; then
            echo "Extracting $rar_file..."
            unrar x -o- "$rar_file"
        fi
    done
else
    echo "Warning: 'unrar' command not found. Skipping RAR extraction."
fi

# Check for unzip
if command -v unzip &> /dev/null; then
    for zip_file in *.zip; do
        if [ -f "$zip_file" ]; then
            echo "Extracting $zip_file..."
            unzip -n "$zip_file"
        fi
    done
else
    echo "Warning: 'unzip' command not found. Skipping ZIP extraction."
fi

echo "Post-processing complete."
