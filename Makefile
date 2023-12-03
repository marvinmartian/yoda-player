# Makefile for building a Go application

# Set the target OS, architecture, and ARM version
GOOS=linux
GOARCH=arm
GOARM=7

# Output binary name
OUTPUT_NAME=go_player

.PHONY: build clean

build:
	cd player && env GOOS=$(GOOS) GOARCH=$(GOARCH) GOARM=$(GOARM) go build -o ../$(OUTPUT_NAME)

clean:
	rm -f $(OUTPUT_NAME)
