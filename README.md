# Terraform Provider for Auth0 Connections

A custom Terraform provider that provides a data source to dynamically retrieve Auth0 connections via the Management API.

## Project Location

This provider is now located at: /terraform-provider-auth0-connections`

## Quick Start

```bash
# Navigate to the provider directory
cd /terraform-provider-auth0-connections

# Build and install the provider
./build.sh

# Test it
cd test-local
terraform init
terraform plan
```

## Features

- **Dynamic Connection Discovery**: Retrieve all Auth0 connections without hardcoding IDs
- **Multiple Output Formats**: Get connections as a list, map, or individual attributes
- **Auth0 Management API Integration**: Uses official Auth0 Management API endpoints
- **Secure Authentication**: Supports client credentials flow for API access

## Installation

### Local Installation

1. Build the provider:
```bash
make build
```

2. Install locally:
```bash
make install
```

### Usage

```hcl
terraform {
  required_providers {
    auth0-connections = {
      source  = "registry.terraform.io/cerifi/auth0-connections"
      version = "~> 1.0"
    }
  }
}

provider "auth0-connections" {
  domain       = "your-tenant.auth0.com"
  client_id     = "your-management-api-client-id"
  client_secret = "your-management-api-client-secret"
}

data "auth0-connections_connections" "all" {
}

# Use the connections data
output "connection_ids" {
  value = data.auth0-connections_connections.all.connection_ids
}

output "connection_map" {
  value = data.auth0-connections_connections.all.connection_map
}

# Example: Get all connection IDs except specific ones
locals {
  all_connection_ids = data.auth0-connections_connections.all.connection_ids
  excluded_connections = ["con_123", "con_456"]
  filtered_connection_ids = [
    for id in local.all_connection_ids : id
    if !contains(local.excluded_connections, id)
  ]
}
```

## Data Source: `auth0-connections_connections`

### Arguments

No arguments required.

### Attributes

- `id` (String) - Identifier of the data source
- `connections` (List of Object) - List of all Auth0 connections with the following attributes:
  - `id` (String) - Connection ID
  - `name` (String) - Connection name
  - `strategy` (String) - Connection strategy (e.g., auth0, google-oauth2)
  - `display_name` (String) - Connection display name
  - `enabled` (Boolean) - Whether the connection is enabled
- `connection_ids` (List of String) - List of all connection IDs
- `connection_map` (Map of String) - Map of connection names to IDs

## Use Cases

### 1. Dynamic Connection Management

Instead of hardcoding connection IDs, you can now dynamically retrieve them:

```hcl
data "auth0-connections_connections" "all" {}

resource "auth0_connection_clients" "example" {
  for_each = toset(data.auth0-connections_connections.all.connection_ids)
  
  connection_id   = each.value
  enabled_clients = []
}
```

### 2. Filter Connections by Strategy

```hcl
locals {
  database_connections = [
    for conn in data.auth0-connections_connections.all.connections : conn.id
    if conn.strategy == "auth0"
  ]
}
```

### 3. Exclude Specific Connections

```hcl
locals {
  excluded_connections = ["con_123", "con_456"]
  filtered_connections = [
    for id in data.auth0-connections_connections.all.connection_ids : id
    if !contains(local.excluded_connections, id)
  ]
}
```

## Development

### Prerequisites

- Go 1.21 or later
- Terraform 1.0 or later

### Building

```bash
# Build the provider
make build

# Run tests
make test

# Format code
make fmt

# Generate documentation
make docs
```

### Testing

```bash
# Run unit tests
make test

# Run acceptance tests
make testacc
```

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is licensed under the MIT License.
