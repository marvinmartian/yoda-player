#!/bin/bash

sudo apt-get update
sudo apt install git python3-dev python3-venv python3-pip mpg123 libsystemd-dev jq -y

curl -L https://github.com/marvinmartian/yoda-player/releases/download/v0.0.1/go_player -o player/go_player

python3 -m venv venv
source venv/bin/activate
cd reader/
pip3 install -r requirements.txt

# sudo pip3 install spidev
# sudo pip3 install mfrc522
# sudo pip3 install systemd-python

# Turn SPI on
echo "dtparam=spi=on" | sudo tee -a /boot/config.txt


git clone https://github.com/waveshare/WM8960-Audio-HAT
cd WM8960-Audio-HAT
sudo ./install.sh
sudo reboot

