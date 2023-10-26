#!/bin/bash

sudo apt-get update
sudo apt install git python3-dev python3-venv python3-pip mpg123 jq -y
sudo pip3 install spidev
sudo pip3 install mfrc522



git clone https://github.com/waveshare/WM8960-Audio-HAT
cd WM8960-Audio-HAT
sudo ./install.sh
sudo reboot

