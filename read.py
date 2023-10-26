#!/usr/bin/env python

import subprocess
import RPi.GPIO as GPIO
from mfrc522 import SimpleMFRC522
import json

reader = SimpleMFRC522()
current_process = None
last_read_id = None

mpg123_path = '/usr/bin/mpg123'

# Read the RFID IDs and actions for controls from controls.json
with open('controls.json', 'r') as controls_file:
    controls_data = json.load(controls_file)

stop_rfid_id = controls_data.get("stop_rfid_id", None)
volume_up_rfid_id = controls_data.get("volume_up_rfid_id", None)
volume_down_rfid_id = controls_data.get("volume_down_rfid_id", None)

# Read the RFID IDs and MP3 file data from mp3.json
with open('mp3.json', 'r') as mp3_file:
    rfid_mp3_mapping = json.load(mp3_file)

# Rest of your code remains the same
# ...

# Handle KeyboardInterrupt and cleanup
try:
    while True:
        id, text = reader.read()
        print(id)
        print(text)

        if id == stop_rfid_id:
            if current_process:
                current_process.terminate()
                current_process.wait()
                current_process = None
        elif id == volume_up_rfid_id:
            print("Volume increased")
            # Add volume increase logic here
        elif id == volume_down_rfid_id:
            print("Volume decreased")
            # Add volume decrease logic here
        elif str(id) in rfid_mp3_mapping:
            if id != last_read_id:
                mp3_info = rfid_mp3_mapping[str(id)]
                mp3_file = mp3_info['file']
                offset = mp3_info['offset']

                if current_process:
                    current_process.terminate()
                    current_process.wait()

                # Construct the command with the starting offset
                command = [mpg123_path, mp3_file]

                try:
                    current_process = subprocess.Popen(command)
                except FileNotFoundError:
                    print("mpg123 not found. Make sure it is installed and the path is correctly set.")
                except Exception as e:
                    print(f"An error occurred: {e}")
            else:
                print("Same RFID tag read again, not restarting.")
        else:
            print("Unrecognized RFID tag: ", id)

        last_read_id = id

except KeyboardInterrupt:
    print("KeyboardInterrupt: Program terminated by the user.")
    if current_process:
        current_process.terminate()
    GPIO.cleanup()
finally:
    if current_process:
        current_process.terminate()
    GPIO.cleanup()
