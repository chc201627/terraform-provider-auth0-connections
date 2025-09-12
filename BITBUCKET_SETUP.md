# Publishing Terraform Provider to Bitbucket

This guide explains how to publish your custom Auth0 Connections Terraform provider to Bitbucket and use it in your Terraform configurations.

## Prerequisites

1. **Bitbucket Account**: You need access to the `cerifi` organization on Bitbucket
2. **SSH Key**: Ensure your SSH key is configured for Bitbucket access
3. **Go Environment**: Go 1.21+ installed and configured

## Step 1: Create Bitbucket Repository

1. Go to [Bitbucket](https://bitbucket.org/cerifi)
2. Click "Create repository"
3. Repository name: `terraform-provider-auth0-connections`
4. Description: `Custom Terraform provider for Auth0 connections management`
5. Set as Private (recommended for internal use)
6. **Do NOT** initialize with README (we already have files)

## Step 2: Push to Bitbucket

```bash
# Add Bitbucket as remote origin
git remote add origin git@bitbucket.org:cerifi/terraform-provider-auth0-connections.git

# Push the main branch
git push -u origin main

# Push the version tag
git push origin v1.0.0
```

## Step 3: Verify Repository

After pushing, verify your repository is accessible:
```bash
git clone git@bitbucket.org:cerifi/terraform-provider-auth0-connections.git
```

## Step 4: Using the Provider in Terraform

### Option A: Git-based Source (Recommended)

Update your Terraform configurations to use the Git-based source:

```hcl
terraform {
  required_providers {
    auth0-connections = {
      source  = "git::ssh://git@bitbucket.org/cerifi/terraform-provider-auth0-connections.git"
      version = "~> 1.0"
    }
  }
}
```

### Option B: Local Development

For local development, you can still use the local provider:

```hcl
terraform {
  required_providers {
    auth0-connections = {
      source  = "local/cerifi/auth0-connections"
      version = "~> 1.0"
    }
  }
}
```

## Step 5: Update Your Modules

The `auth0-connection-manager` module has already been updated to use the Bitbucket source. You can verify this in:

```
terraform-oncerifi-auth0/modules/auth0-connection-manager/versions.tf
```

## Step 6: Testing the Provider

1. **Initialize Terraform**:
   ```bash
   cd /path/to/your/terraform/project
   terraform init
   ```

2. **Plan the changes**:
   ```bash
   terraform plan
   ```

3. **Apply if everything looks good**:
   ```bash
   terraform apply
   ```

## Version Management

### Creating New Versions

1. **Make your changes** to the provider code
2. **Update version** in `go.mod` if needed
3. **Commit changes**:
   ```bash
   git add .
   git commit -m "Description of changes"
   ```
4. **Create and push new tag**:
   ```bash
   git tag -a v1.1.0 -m "Release version 1.1.0"
   git push origin v1.1.0
   ```

### Using Specific Versions

In your Terraform configurations, you can specify exact versions:

```hcl
auth0-connections = {
  source  = "git::ssh://git@bitbucket.org/cerifi/terraform-provider-auth0-connections.git"
  version = "1.0.0"  # Exact version
}
```

Or use version constraints:

```hcl
auth0-connections = {
  source  = "git::ssh://git@bitbucket.org/cerifi/terraform-provider-auth0-connections.git"
  version = "~> 1.0"  # Any version >= 1.0.0, < 2.0.0
}
```

## Troubleshooting

### SSH Key Issues

If you get SSH authentication errors:

1. **Check SSH key**:
   ```bash
   ssh -T git@bitbucket.org
   ```

2. **Add SSH key to Bitbucket**:
   - Go to Bitbucket Settings → SSH Keys
   - Add your public key

### Provider Not Found

If Terraform can't find the provider:

1. **Check repository URL** is correct
2. **Verify tag exists**:
   ```bash
   git ls-remote --tags git@bitbucket.org:cerifi/terraform-provider-auth0-connections.git
   ```
3. **Clear Terraform cache**:
   ```bash
   rm -rf .terraform/
   terraform init
   ```

### Go Module Issues

If you get Go module errors:

1. **Update go.mod**:
   ```bash
   go mod tidy
   ```

2. **Verify module path** in `go.mod` matches Bitbucket URL

## Security Considerations

1. **Private Repository**: Keep the provider repository private
2. **Access Control**: Limit access to team members who need it
3. **Credentials**: Never commit Auth0 credentials to the repository
4. **Environment Variables**: Use environment variables for sensitive data

## Benefits of Bitbucket Publishing

1. **Version Control**: Proper versioning with Git tags
2. **Team Access**: Multiple team members can use the provider
3. **CI/CD Integration**: Can be integrated with Bitbucket Pipelines
4. **Documentation**: Centralized documentation and examples
5. **Backup**: Provider code is safely stored in version control

## Next Steps

1. **Push to Bitbucket** using the commands above
2. **Test the provider** in your Terraform configurations
3. **Update team documentation** with the new provider usage
4. **Set up CI/CD** if needed for automated testing
