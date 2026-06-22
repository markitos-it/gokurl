.PHONY: help build-all build-linux build-windows build-darwin release-all clean 

all: help

APP_ID=it.markitos.gokurl
MACOS_SDK_PATH=/home/markitos/.sdk/MacOSX-SDKs/MacOSX13.3.sdk

build-all:
	fyne-cross linux -arch=amd64
	fyne-cross windows -app-id=$(APP_ID) -arch=amd64

build-linux:
	fyne-cross linux -arch=amd64

build-windows:
	fyne-cross windows -app-id=$(APP_ID) -arch=amd64

build-darwin:
	fyne-cross darwin -app-id=$(APP_ID) -macosx-sdk-path=$(MACOS_SDK_PATH) -metadata appBuild=1 -arch=arm64,amd64

release-all:
	@echo "Creating production packages..."
	
	@echo "Packaging for Linux..."
	fyne-cross linux -app-id=$(APP_ID) -release -arch=amd64
	
	@echo "Packaging for Windows..."
	fyne-cross windows -app-id=$(APP_ID) -ldflags="-s -w -H=windowsgui" -arch=amd64
	
	@echo "Packaging for macOS (Con bypass de metadatos)..."
	fyne-cross darwin -app-id=$(APP_ID) -macosx-sdk-path=$(MACOS_SDK_PATH) -metadata appBuild=1 -release -arch=arm64,amd64

clean:
	@echo "Cleaning up build artifacts..."
	rm -rf fyne-cross-build

help:
	@echo "Available targets:"
	@echo "  all             - Builds all supported platforms (linux, windows)."
	@echo "  build-all       - Builds all supported platforms (linux, windows)."
	@echo "  build-linux     - Builds for Linux (amd64)."
	@echo "  build-windows   - Builds for Windows (amd64)."
	@echo "  build-darwin    - Builds for macOS (arm64, amd64)."
	@echo "  release-all     - Creates production packages for all supported platforms."
	@echo "  clean           - Removes all build artifacts."
	@echo "  help            - Displays this help message."