#!/bin/bash

# Release script for Terraform Provider
# This script creates a proper release with SHASUMS files

set -e

VERSION=${1:-"1.0.0"}
PROVIDER_NAME="terraform-provider-auth0-connections"
NAMESPACE="chc201627"

echo "Creating release for version: $VERSION"

# Create release directory
mkdir -p releases

# Build for all platforms
echo "Building for all platforms..."

# Darwin AMD64
GOOS=darwin GOARCH=amd64 go build -ldflags="-s -w -X main.version=$VERSION" -o releases/${PROVIDER_NAME}_${VERSION}_darwin_amd64

# Darwin ARM64
GOOS=darwin GOARCH=arm64 go build -ldflags="-s -w -X main.version=$VERSION" -o releases/${PROVIDER_NAME}_${VERSION}_darwin_arm64

# Linux AMD64
GOOS=linux GOARCH=amd64 go build -ldflags="-s -w -X main.version=$VERSION" -o releases/${PROVIDER_NAME}_${VERSION}_linux_amd64

# Linux ARM64
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w -X main.version=$VERSION" -o releases/${PROVIDER_NAME}_${VERSION}_linux_arm64

# Windows AMD64
GOOS=windows GOARCH=amd64 go build -ldflags="-s -w -X main.version=$VERSION" -o releases/${PROVIDER_NAME}_${VERSION}_windows_amd64.exe

# Windows ARM64
GOOS=windows GOARCH=arm64 go build -ldflags="-s -w -X main.version=$VERSION" -o releases/${PROVIDER_NAME}_${VERSION}_windows_arm64.exe

echo "Creating SHASUMS file..."

# Create SHASUMS file
cd releases
shasum -a 256 ${PROVIDER_NAME}_${VERSION}_* > terraform-provider-auth0-connections_${VERSION}_SHA256SUMS

echo "SHASUMS file created:"
cat terraform-provider-auth0-connections_${VERSION}_SHA256SUMS

echo "Release files created in releases/ directory"
echo "Upload these files to your GitHub release:"
ls -la

cd ..
