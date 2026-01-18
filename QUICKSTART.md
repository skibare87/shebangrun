# Quick Start Guide

## Prerequisites

- Docker and Docker Compose
- Go 1.22+ (optional, for local development)

## Getting Started

### 1. Start the Stack

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f app

# Check status
docker-compose ps
```

The application will be available at `http://localhost`

### 2. First User Setup

The first user to register will automatically become an admin.

1. Navigate to `http://localhost/register`
2. Create your account
3. You'll be redirected to the dashboard

### 3. Create Your First Script

1. Click "New Script" in the dashboard
2. Fill in the details:
   - Name: `hello`
   - Description: `My first script`
   - Visibility: `public`
   - Content: `#!/bin/bash\necho "Hello from shebang.run!"`
3. Click "Create"

### 4. Use Your Script

```bash
# Get the latest version
curl http://localhost/yourusername/hello | sh

# Get a specific version
curl http://localhost/yourusername/hello@v1 | sh

# Get metadata
curl http://localhost/yourusername/hello/meta
```

## Configuration

### Environment Variables

Create a `.env` file or export these variables:

```bash
# Required
export JWT_SECRET="your-secret-key-change-this"

# Optional OAuth (for GitHub/Google login)
export GITHUB_CLIENT_ID="your-github-client-id"
export GITHUB_CLIENT_SECRET="your-github-client-secret"
export GOOGLE_CLIENT_ID="your-google-client-id"
export GOOGLE_CLIENT_SECRET="your-google-client-secret"

# Storage (defaults to S3/MinIO)
export STORAGE_TYPE="s3"  # or "local"
export S3_ENDPOINT="minio:9000"
export S3_ACCESS_KEY="minioadmin"
export S3_SECRET_KEY="minioadmin"
export S3_BUCKET="scripts"

# Limits
export DEFAULT_RATE_LIMIT="50"
export DEFAULT_MAX_SCRIPTS="25"
export DEFAULT_MAX_SCRIPT_SIZE="1048576"
```

### Database

The database is automatically initialized with the schema on first run.

To manually run migrations:

```bash
docker-compose exec mariadb mysql -u root -prootpassword shebang < migrations/001_initial_schema.sql
```

## Key Features

### Script Versioning

Every update creates a new immutable version:

```bash
# Update a script (creates v2)
curl -X PUT http://localhost/api/scripts/1 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "#!/bin/bash\necho \"Updated!\""}'

# Access specific versions
curl http://localhost/username/script@v1
curl http://localhost/username/script@v2
curl http://localhost/username/script@latest
```

### Tags

Tag versions for easy reference:

```bash
# Tag a version as "dev"
curl -X PUT http://localhost/api/scripts/1 \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"content": "...", "tag": "dev"}'

# Use the tag
curl http://localhost/username/script@dev | sh
```

### Private Scripts with Sharing

```bash
# Generate a share token
curl -X POST http://localhost/api/scripts/1/share \
  -H "Authorization: Bearer YOUR_TOKEN"

# Returns: {"token": "abc123..."}

# Share the script with the token
curl "http://localhost/username/script?token=abc123..." | sh

# Revoke the token
curl -X DELETE http://localhost/api/scripts/1/share/abc123 \
  -H "Authorization: Bearer YOUR_TOKEN"
```

### Key Management

Generate or import RSA keypairs for signing/encryption:

```bash
# Generate a new keypair
curl -X POST http://localhost/api/keys/generate \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "my-key"}'

# Returns public key and private key (save the private key!)

# Import an existing public key
curl -X POST http://localhost/api/keys/import \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"name": "imported", "public_key": "-----BEGIN PUBLIC KEY-----\n..."}'
```

## Admin Functions

As an admin, you can manage users and limits:

```bash
# List all users
curl http://localhost/api/admin/users \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"

# Set user limits
curl -X PUT http://localhost/api/admin/users/2/limits \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"max_scripts": 100, "max_script_size": 5242880, "rate_limit": 100}'

# Get system config
curl http://localhost/api/admin/config \
  -H "Authorization: Bearer YOUR_ADMIN_TOKEN"
```

## Troubleshooting

### Database Connection Issues

```bash
# Check if MariaDB is running
docker-compose ps mariadb

# View MariaDB logs
docker-compose logs mariadb

# Restart MariaDB
docker-compose restart mariadb
```

### Storage Issues

```bash
# Check MinIO status
docker-compose ps minio

# Access MinIO console
open http://localhost:9001
# Login: minioadmin / minioadmin
```

### Application Logs

```bash
# View application logs
docker-compose logs -f app

# Check for errors
docker-compose logs app | grep -i error
```

## Development

### Local Development (without Docker)

```bash
# Install dependencies
go mod download

# Set up database
mysql -u root -p < migrations/001_initial_schema.sql

# Set environment variables
export DATABASE_URL="root:password@tcp(localhost:3306)/shebang?parseTime=true"
export STORAGE_TYPE="local"
export LOCAL_STORAGE_PATH="./data/scripts"

# Run the server
go run cmd/server/main.go
```

### Building

```bash
# Build the binary
go build -o shebang-server cmd/server/main.go

# Run
./shebang-server
```

## API Reference

See `ProjectPlan.md` for complete API documentation.

## Support

For issues or questions, check:
- README.md for general information
- ProjectPlan.md for architecture details
- PROGRESS.md for implementation status
