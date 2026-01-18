# shebangrun Python Client

Python client library for [shebang.run](https://shebang.run) - a platform for hosting and sharing shell scripts with versioning, encryption, and signing.

## Installation

```bash
pip install shebangrun
```

Or install from source:

```bash
cd python
pip install -e .
```

## Quick Start

### Simple Script Fetching

```python
from shebangrun import run

# Fetch a script (returns content as string)
content = run(username="mpruitt", script="bashtest")
print(content)
```

### Execute Python Scripts

```python
from shebangrun import run

# Fetch and execute with confirmation prompt
run(username="mpruitt", script="myscript", eval=True)

# Execute without confirmation (use with caution!)
run(username="mpruitt", script="myscript", eval=True, accept=True)
```

### Working with Versions

```python
from shebangrun import run

# Get latest version
content = run(username="mpruitt", script="deploy", version="latest")

# Get specific version
content = run(username="mpruitt", script="deploy", version="v5")

# Get tagged version
content = run(username="mpruitt", script="deploy", version="dev")
```

### Private Scripts

```python
from shebangrun import run

# Access private script with share token
content = run(
    username="mpruitt", 
    script="private-script",
    token="your-share-token-here"
)
```

## Full API Client

For more advanced usage, use the `ShebangClient` class:

```python
from shebangrun import ShebangClient

# Initialize client
client = ShebangClient(url="shebang.run")

# Login
client.login(username="myuser", password="mypassword")

# Create a script
client.create_script(
    name="hello",
    content="#!/bin/bash\necho 'Hello World'",
    description="My first script",
    visibility="public"
)

# List your scripts
scripts = client.list_scripts()
for script in scripts:
    print(f"{script['name']} - v{script['version']}")

# Update a script (creates new version)
client.update_script(
    script_id=1,
    content="#!/bin/bash\necho 'Hello World v2'",
    tag="dev"
)

# Generate share token for private script
token = client.generate_share_token(script_id=1)
print(f"Share URL: https://shebang.run/myuser/myscript?token={token}")

# Get script metadata
meta = client.get_metadata(username="mpruitt", script="bashtest")
print(f"Version: {meta['version']}, Size: {meta['size']} bytes")

# Verify signature
verification = client.verify_signature(username="mpruitt", script="bashtest")
print(f"Signed: {verification['signed']}")
```

## Key Management

```python
from shebangrun import ShebangClient

client = ShebangClient(url="shebang.run")
client.login(username="myuser", password="mypassword")

# Generate a new keypair
key = client.generate_key(name="my-signing-key")
print(f"Public Key: {key['public_key']}")
print(f"Private Key: {key['private_key']}")  # Save this securely!

# List keys
keys = client.list_keys()
for key in keys:
    print(f"{key['name']} - Created: {key['created_at']}")

# Import existing public key
client.import_key(
    name="imported-key",
    public_key="-----BEGIN PUBLIC KEY-----\n..."
)

# Delete a key
client.delete_key(key_id=1)
```

## Account Management

```python
from shebangrun import ShebangClient

client = ShebangClient(url="shebang.run")
client.login(username="myuser", password="mypassword")

# Change password
client.change_password(
    current_password="oldpass",
    new_password="newpass"
)

# Export all data (GDPR)
data = client.export_data()
print(f"Exported {len(data['scripts'])} scripts")

# Delete account (permanent!)
client.delete_account()
```

## API Reference

### `run()` Function

```python
run(username, script, key=None, eval=False, accept=False, 
    url="shebang.run", version=None, token=None)
```

**Parameters:**
- `username` (str, required): Script owner's username
- `script` (str, required): Script name
- `key` (str, optional): Private key for decryption (not yet implemented)
- `eval` (bool, optional): Execute the script in Python (default: False)
- `accept` (bool, optional): Skip confirmation when eval=True (default: False)
- `url` (str, optional): Base URL (default: "shebang.run")
- `version` (str, optional): Version tag (e.g., "latest", "v1", "dev")
- `token` (str, optional): Share token for private scripts

**Returns:**
- String content if `eval=False`
- Execution result if `eval=True`

### `ShebangClient` Class

#### Authentication
- `register(username, email, password)` - Register new user
- `login(username, password)` - Login and get JWT token

#### Script Management
- `list_scripts()` - List user's scripts
- `get_script(username, script, version=None, token=None)` - Fetch script content
- `get_metadata(username, script)` - Get script metadata
- `verify_signature(username, script)` - Verify script signature
- `create_script(name, content, description="", visibility="private", keypair_id=None)` - Create script
- `update_script(script_id, content=None, description=None, visibility=None, tag=None, keypair_id=None)` - Update script
- `delete_script(script_id)` - Delete script
- `generate_share_token(script_id)` - Generate share token
- `revoke_share_token(script_id, token)` - Revoke share token

#### Key Management
- `list_keys()` - List keypairs
- `generate_key(name)` - Generate new keypair
- `import_key(name, public_key)` - Import public key
- `delete_key(key_id)` - Delete keypair

#### Account Management
- `change_password(current_password, new_password)` - Change password
- `export_data()` - Export all data (GDPR)
- `delete_account()` - Delete account

## Security Notes

⚠️ **Warning:** Using `eval=True` with `accept=True` will execute remote code without confirmation. Only use this with scripts you trust completely.

✅ **Best Practices:**
- Always review scripts before executing with `eval=True`
- Use `accept=False` (default) to see the script before execution
- Store private keys securely, never commit them to version control
- Use environment variables for tokens and credentials
- Verify script signatures when available

## Examples

### Automation Script

```python
#!/usr/bin/env python3
from shebangrun import run

# Fetch deployment script and execute with confirmation
run(
    username="devops",
    script="deploy-prod",
    version="latest",
    eval=True,
    accept=False  # Always confirm production deployments!
)
```

### CI/CD Integration

```python
import os
from shebangrun import ShebangClient

client = ShebangClient()
client.login(
    username=os.environ["SHEBANG_USER"],
    password=os.environ["SHEBANG_PASS"]
)

# Update deployment script
client.update_script(
    script_id=int(os.environ["DEPLOY_SCRIPT_ID"]),
    content=open("deploy.sh").read(),
    tag="latest"
)

print("Deployment script updated!")
```

## License

MIT

## Links

- Website: https://shebang.run
- Documentation: https://shebang.run/docs
- GitHub: https://github.com/shebangrun/python-client
