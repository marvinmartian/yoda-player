#!/bin/bash

# Get the current working directory
CURRENT_DIR="$(pwd)"

# Define the service file content
SERVICE_CONTENT="[Unit]
Description=Yoda RFID Reader
After=network.target go-music-player.service
Wants=go-music-player.service

[Service]
ExecStart=$CURRENT_DIR/.venv/bin/python $CURRENT_DIR/reader/read.py
WorkingDirectory=$CURRENT_DIR/reader/
User=melvin
Group=melvin
Restart=always

[Install]
WantedBy=multi-user.target"

# Specify the path for the service file
SERVICE_FILE_PATH="/etc/systemd/system/yoda_rfid_reader.service"

# Print the service content to the file
echo "$SERVICE_CONTENT" | sudo tee "$SERVICE_FILE_PATH" > /dev/null

# Inform the user that the file has been created
echo "Service file created at $SERVICE_FILE_PATH"

# Reload systemd to pick up the new service file
sudo systemctl daemon-reload

# Enable and start the service
sudo systemctl enable yoda_rfid_reader.service
sudo systemctl start yoda_rfid_reader.service
