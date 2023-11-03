import requests
import time
import RPi.GPIO as GPIO
from mfrc522 import SimpleMFRC522
from requests.exceptions import ConnectionError
import systemd.journal

reader = SimpleMFRC522()

while True:
    try:
        systemd.journal.send("Place an RFID tag near the reader...")
        id, text = reader.read()
        systemd.journal.send(f"Tag ID: {id}")

        # Define the URL to which you want to send the data
        url = 'http://yoda:3001/play'  # Update the URL as needed

        # Create a dictionary with the RFID ID to send as JSON data
        data = {'id': str(id)}

        while True:
            try:
                # Send the POST request with JSON data
                response = requests.post(url, json=data, timeout=5, headers={'Content-Type': 'application/json'})

                if response.status_code == 200:
                    systemd.journal.send("Data sent successfully.")
                    break
                else:
                    systemd.journal.send(f"Failed to send data. Status code: {response.status_code}")
                    systemd.journal.send(f"Response body: {response.text}")
                    break

            except ConnectionError as e:
                systemd.journal.send(f"Connection error: {e}")
                time.sleep(5)  # Wait for a moment before retrying
        
        time.sleep(2)

    except KeyboardInterrupt:
        GPIO.cleanup()
        break
