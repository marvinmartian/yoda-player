#!/bin/bash

sudo apt-get update
sudo apt install git python3-dev python3-venv python3-pip mpg123 libsystemd-dev jq -y

python3 -m venv venv
source venv/bin/activate
cd reader/
pip3 install -r requirements.txt

# sudo pip3 install spidev
# sudo pip3 install mfrc522
# sudo pip3 install systemd-python



git clone https://github.com/waveshare/WM8960-Audio-HAT
cd WM8960-Audio-HAT
sudo ./install.sh
sudo reboot

