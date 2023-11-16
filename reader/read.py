import requests
import time
import RPi.GPIO as GPIO
from mfrc522 import SimpleMFRC522
from requests.exceptions import ConnectionError
import systemd.journal

reader = SimpleMFRC522()


# Tag read history structure
# tagHistory is a dict, expected keys being the tag IDs of scanned tags
# HISTORY_DEPTH is a constant value that defines the depth of the list used as a ring for timestamps of the scan events
# a Reset Intent is defined as INTENT_RATE read events for the same within 10 seconds

HISTORY_DEPTH = 4
INTENT_RATE = 3
YODA_URL = 'http://yoda:3001/play'
YODA_TIMEOUT_RETRY = 5
tagHistory = {}

def cardRead(tagID,tagText):
    systemd.journal.send(f"Tag ID: {tagID}")
    updateTagHistory(tagID)
    if detectResetIntent(tagID):
        systemd.journal.send(f"Tag {tagID} playback reset request")
    yodaPlay(tagID)

def yodaPlay(tagID):
    data = {'id': str(tagID)}
    while True:
        try:
            # Send the POST request with JSON data
            response = requests.post(YODA_URL, json=data, timeout=5, headers={'Content-Type': 'application/json'})
            if response.status_code == 200:
                systemd.journal.send("Data sent successfully.")
                break
            else:
                systemd.journal.send(f"Failed to send data. Status code: {response.status_code}")
                systemd.journal.send(f"Response body: {response.text}")
                break
        except ConnectionError as e:
            systemd.journal.send(f"Connection error: {e}")
            time.sleep(YODA_TIMEOUT_RETRY)  # Wait for a moment before retrying

def updateTagHistory(tagID):
    if not tagID in tagHistory:
        tagHistory[tagID] = [time.struct_time]*HISTORY_DEPTH
    insertIndex = 0
    insertTime = time.gmtime()
    oldest = time.mktime(insertTime)
    for index in range(len(tagHistory[tagID])):
        if not isinstance(tagHistory[tagID][index], time.struct_time):
            insertIndex = index
            break;
        if time.mktime(tagHistory[tagID][index]) < oldest:
            oldest = time.mktime(tagHistory[tagID][index])
            insertIndex = index
    tagHistory[tagID][insertIndex] = insertTime

def detectResetIntent(tagID):
    floorTime=time.gmtime()-10
    count=0
    for index in range(len(tagHistory[tagID])):
        if time.mktime(tagHistory[tagID][index]) > floorTime:
            count+=1
    if count>=INTENT_RATE:
        return True
    return False


while True:
    try:
        systemd.journal.send("Place an RFID tag near the reader...")
        id, text = reader.read()
        cardRead(id, text)
        time.sleep(2)
    except KeyboardInterrupt:
        GPIO.cleanup()
        break
