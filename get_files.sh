#!/bin/bash

# Check if a JSON file is provided as an argument
if [ $# -ne 1 ]; then
    echo "Usage: $0 <json_file>"
    exit 1
fi

mkdir -p files 

json_file="$1"

# Check if the JSON file exists
if [ ! -f "$json_file" ]; then
    echo "Error: JSON file '$json_file' not found."
    exit 1
fi

# Iterate through the JSON array and process each item
for item in $(jq -c '.[]' "$json_file"); do
    download_url=$(echo "$item" | jq -r '.download')
    filename=$(echo "$item" | jq -r '.file')

    echo $download_url
    echo $filename

    # Check if the download URL and filename are empty
    if [ -z "$download_url" ] || [ -z "$filename" ]; then
        echo "Error: 'download' and 'filename' fields must be present in each JSON item."
        exit 1
    fi

    # Download the file and rename it
    if curl -L -o "$filename" "$download_url"; then
        echo "File downloaded and renamed to '$filename'."
    else
        echo "Error: Download failed for '$download_url'."
        # exit 1
    fi
done
