import requests
import time
import RPi.GPIO as GPIO
from mfrc522 import SimpleMFRC522
from requests.exceptions import ConnectionError

reader = SimpleMFRC522()

while True:
    try:
        print("Place an RFID tag near the reader...")
        id, text = reader.read()
        print(f"Tag ID: {id}")

        # Define the URL to which you want to send the data
        url = 'http://yoda:3001/play'  # Update the URL as needed

        # Create a dictionary with the RFID ID to send as JSON data
        data = {'id': str(id)}

        while True:
            try:
                # Send the POST request with JSON data
                response = requests.post(url, json=data, timeout=5, headers={'Content-Type': 'application/json'})

                if response.status_code == 200:
                    print("Data sent successfully.")
                    break
                    # time.sleep(2)
                    # GPIO.cleanup()
                    # exit()  # Exit the script on success
                else:
                    print("Failed to send data. Status code:", response.status_code)
                    print("Response body:", response.text)
                    break

            except ConnectionError as e:
                print("Connection error:", e)
                time.sleep(5)  # Wait for a moment before retrying
        
        time.sleep(2)

    except KeyboardInterrupt:
        GPIO.cleanup()
        break
