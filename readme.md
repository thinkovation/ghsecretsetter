# GitHub Secrets Setter

A secure command-line tool for setting GitHub repository secrets using the GitHub API. Supports both single and multiple secrets, with the ability to read secret values from files.

## Features

- **Secure encryption** using libsodium's sealed box format (NaCl)
- **Multiple secrets** support - set many secrets at once
- **File-based secrets** - read secret values directly from files
- **Flexible configuration** - CLI flags or YAML config files
- **Backward compatible** - works with existing single-secret workflows
- **Error resilient** - individual secret failures don't stop the entire operation

## Installation

### Prerequisites

- Go 1.24.1 or later
- GitHub Personal Access Token with `repo` scope

### Build from source

```bash
git clone <repository-url>
cd ghsecretsetter
go build -o ghsecretsetter .
```

## Quick Start

### Single Secret (CLI)

```bash
./ghsecretsetter -owner=myuser -repo=myrepo -secret=API_KEY -value=secret123
```

### Multiple Secrets (Config File)

1. Copy the example config:
```bash
cp config.yaml.example config.yaml
```

2. Edit `config.yaml`:
```yaml
owner: myuser
repo: myrepo
secrets:
  API_KEY: secret123
  DATABASE_URL: postgres://user:pass@host:5432/db
  SSL_CERT: file(./certs/server.pem)
```

3. Run:
```bash
./ghsecretsetter -config=config.yaml
```

## Configuration

### YAML Configuration File

The recommended approach for multiple secrets:

```yaml
owner: yourusername
repo: yourrepo

# Multiple secrets
secrets:
  API_KEY: your_api_key_value
  DATABASE_URL: postgres://user:pass@host:5432/db
  
  # File-based secrets - reads contents from file
  SSL_CERT: file(/path/to/cert.pem)
  PRIVATE_KEY: file(./keys/private.key)
  CONFIG_JSON: file(config.json)

# GitHub token (optional - can use GITHUB_TOKEN env var)
token: ghp_abc123...
```

### CLI Flags

For single secrets or overriding config values:

- `-config` - Path to YAML config file
- `-owner` - GitHub repository owner/organization
- `-repo` - GitHub repository name
- `-secret` - Secret name (single secret mode)
- `-value` - Secret value (single secret mode)
- `-token` - GitHub Personal Access Token (optional)

### Authentication

Provide your GitHub token via:

1. **CLI flag**: `-token=ghp_abc123...`
2. **Config file**: `token: ghp_abc123...`
3. **Environment variable**: `export GITHUB_TOKEN=ghp_abc123...`

## File-Based Secrets

Use the `file(path)` syntax to read secret values from files:

```yaml
secrets:
  SSL_CERTIFICATE: file(/etc/ssl/certs/server.pem)
  PRIVATE_KEY: file(./keys/private.key)
  CONFIG_FILE: file(./config/app.json)
```

**Features:**
- Supports both absolute and relative paths
- Automatically trims trailing newlines
- Clear error messages for missing files
- Mixed usage with regular string secrets

## Examples

### Development Environment Setup

```yaml
owner: myorg
repo: myapp
secrets:
  NODE_ENV: development
  API_URL: https://api.dev.example.com
  DATABASE_URL: file(./env/dev-db-url.txt)
  JWT_SECRET: file(./secrets/jwt.key)
```

### Production Deployment

```bash
# Set multiple production secrets
./ghsecretsetter -config=production.yaml

# Override specific values
./ghsecretsetter -config=production.yaml -owner=myorg-prod
```

### CI/CD Pipeline

```bash
# Set secrets for GitHub Actions
export GITHUB_TOKEN=$GITHUB_PAT
./ghsecretsetter -owner=myorg -repo=myapp \
  -secret=DEPLOY_KEY -value="$(cat ~/.ssh/deploy_key)"
```

## Requirements

- **GitHub Personal Access Token** with `repo` scope
  - Create at: https://github.com/settings/personal-access-tokens
- **Repository access** - you must have admin access to the target repository
- **Go 1.24.1+** for building from source

## Security

- Secrets are encrypted using libsodium's sealed box format before transmission
- No secrets are logged or stored locally
- Uses GitHub's official public key encryption API
- Supports secure file-based secret management

## Error Handling

The tool provides clear feedback for common issues:

- Missing or invalid GitHub tokens
- Repository access denied
- File not found (for file-based secrets)
- Invalid encryption
- Successful secret creation/update

Individual secret failures don't stop processing of other secrets.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Run tests: `go test ./...`
5. Submit a pull request

## License

[Add your license here]

## Support

For issues and feature requests, please use the GitHub issue tracker.