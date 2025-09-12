#!/bin/bash

# Build and install the Auth0 Connections Terraform Provider

set -e

echo "🔨 Building Auth0 Connections Terraform Provider..."

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -f terraform-provider-auth0-connections

# Download dependencies
echo "📦 Downloading dependencies..."
go mod tidy
go mod download

# Build the provider
echo "🔨 Building provider..."
go build -o terraform-provider-auth0-connections

# Create plugin directory
PLUGIN_DIR="$HOME/.terraform.d/plugins/registry.terraform.io/cerifi/auth0-connections/1.0.0/linux_amd64"
echo "📁 Creating plugin directory: $PLUGIN_DIR"
mkdir -p "$PLUGIN_DIR"

# Install the provider
echo "📥 Installing provider..."
cp terraform-provider-auth0-connections "$PLUGIN_DIR/"

echo "✅ Provider built and installed successfully!"
echo ""
echo "You can now use the provider in your Terraform configurations:"
echo ""
echo "terraform {"
echo "  required_providers {"
echo "    auth0-connections = {"
echo "      source  = \"registry.terraform.io/cerifi/auth0-connections\""
echo "      version = \"~> 1.0\""
echo "    }"
echo "  }"
echo "}"
echo ""
echo "provider \"auth0-connections\" {"
echo "  domain       = \"your-tenant.auth0.com\""
echo "  client_id     = \"your-management-api-client-id\""
echo "  client_secret = \"your-management-api-client-secret\""
echo "}"
echo ""
echo "data \"auth0-connections_connections\" \"all\" {}"
