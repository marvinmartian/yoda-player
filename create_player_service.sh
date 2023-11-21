#!/bin/bash

# Get the current working directory
CURRENT_DIR="$(pwd)"

# Define the service file content
SERVICE_CONTENT="[Unit]
Description=Go Music Player Service
After=network.target

[Service]
ExecStart=$CURRENT_DIR/player/go_player
WorkingDirectory=$CURRENT_DIR/player
User=melvin
Group=melvin
Restart=always

[Install]
WantedBy=multi-user.target"

# Specify the path for the service file
SERVICE_FILE_PATH="/etc/systemd/system/go_music_player.service"

# Print the service content to the file
echo "$SERVICE_CONTENT" | sudo tee "$SERVICE_FILE_PATH" > /dev/null

# Inform the user that the file has been created
echo "Service file created at $SERVICE_FILE_PATH"

# Reload systemd to pick up the new service file
sudo systemctl daemon-reload

# Enable and start the service
sudo systemctl enable go_music_player.service
sudo systemctl start go_music_player.service
