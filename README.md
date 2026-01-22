# shebang.run

A platform for hosting and sharing shell scripts and code snippets with built-in versioning, encryption, signing, and secrets management.

## Features

- **Script Versioning**: Auto-incrementing versions with immutable history
- **Access Control**: Private, unlisted, and public scripts with ACL-based sharing
- **Encryption & Signing**: ChaCha20-Poly1305 encryption and RSA-PSS signatures
- **Secrets Management**: Encrypted key-value store with audit logging
- **Script Sharing**: Share unlisted scripts with specific users or "anyone with link"
- **Secret Injection**: Reference secrets in scripts with ${SECRET:name} syntax
- **Multiple Storage Backends**: S3-compatible or local filesystem
- **OAuth Integration**: GitHub and Google authentication with username selection
- **Rate Limiting**: Configurable per-user limits
- **Docker Deployment**: Complete stack with MariaDB and MinIO

## Quick Start

### Prerequisites

- Docker and Docker Compose
- Go 1.22+ (for local development)

### Running with Docker

```bash
# Clone the repository
git clone <repo-url>
cd shebang.run

# Set environment variables (optional)
export JWT_SECRET="your-secret-key"
export GITHUB_CLIENT_ID="your-github-client-id"
export GITHUB_CLIENT_SECRET="your-github-client-secret"
export GOOGLE_CLIENT_ID="your-google-client-id"
export GOOGLE_CLIENT_SECRET="your-google-client-secret"

# Start the stack
docker-compose up -d

# View logs
docker-compose logs -f app
```

The application will be available at `http://localhost`

### CLI Tool

Install the CLI via pip:
```bash
pip install shebangrun
shebang login
shebang list
```

Or use the Docker CLI image:
```bash
docker run -it --rm -v ~/.shebangrc:/root/.shebangrc dingbatter/shebangcli shebang list
```

See [Python README](python/README.md) for complete CLI documentation.

### Local Development

```bash
# Install dependencies
go mod download

# Run migrations
mysql -u root -p < migrations/001_initial_schema.sql

# Start the server
go run cmd/server/main.go
```

## Usage

### Retrieve a script

```bash
# Latest version
curl https://shebang.run/username/scriptname | sh

# Specific version
curl https://shebang.run/username/scriptname@v5 | sh

# Tagged version
curl https://shebang.run/username/scriptname@dev | sh
```

### Get script metadata

```bash
curl https://shebang.run/username/scriptname/meta
```

### Verify script signature

```bash
curl https://shebang.run/username/scriptname/verify
```

## Configuration

Environment variables:

- `SERVER_PORT`: Server port (default: 8080)
- `DATABASE_URL`: MariaDB connection string
- `JWT_SECRET`: Secret for JWT tokens
- `STORAGE_TYPE`: `s3` or `local`
- `S3_ENDPOINT`: S3 endpoint URL
- `S3_ACCESS_KEY`: S3 access key
- `S3_SECRET_KEY`: S3 secret key
- `S3_BUCKET`: S3 bucket name
- `LOCAL_STORAGE_PATH`: Local storage directory
- `DEFAULT_RATE_LIMIT`: Requests per minute (default: 50)
- `DEFAULT_MAX_SCRIPTS`: Max scripts per user (default: 25)
- `DEFAULT_MAX_SCRIPT_SIZE`: Max script size in bytes (default: 1MB)
- `GITHUB_CLIENT_ID`: GitHub OAuth client ID
- `GITHUB_CLIENT_SECRET`: GitHub OAuth client secret
- `GOOGLE_CLIENT_ID`: Google OAuth client ID
- `GOOGLE_CLIENT_SECRET`: Google OAuth client secret
- `MASTER_ENCRYPTION_KEY`: Base64-encoded 32-byte key for server-side encryption
- `MASTER_KEY_SOURCE`: Key source (`env`, `aws_kms`, `aws_secrets`)
- `SECRETS_BACKEND`: Secrets storage backend (`database`, `redis`, `dynamodb`)

## Architecture

```
shebang.run/
├── cmd/server/          # Application entry point
├── internal/
│   ├── api/             # HTTP handlers
│   ├── auth/            # Authentication
│   ├── crypto/          # Encryption & signing
│   ├── storage/         # Storage backends
│   ├── database/        # Database models
│   ├── middleware/      # HTTP middleware
│   └── config/          # Configuration
├── web/                 # Frontend assets
├── migrations/          # Database migrations
└── docker-compose.yml   # Docker stack
```

## License

MIT
