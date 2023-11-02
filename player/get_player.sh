#!/bin/bash

ARTIFACT_FILE="go_player"

# Get the URL for the latest artifact of the workflow run
ARTIFACT_URL="https://github.com/marvinmartian/yoda-player/releases/download/v0.0.1/player"

# Download the artifact using curl
curl -L -o "$ARTIFACT_FILE" "$ARTIFACT_URL"

chmod +x $ARTIFACT_FILE
