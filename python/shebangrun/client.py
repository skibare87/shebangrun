"""
Main client for interacting with shebang.run
"""

import requests
from typing import Optional, Any


class ShebangClient:
    """Client for interacting with shebang.run API"""
    
    def __init__(self, url: str = "shebang.run", token: Optional[str] = None):
        """
        Initialize the client
        
        Args:
            url: Base URL (default: shebang.run)
            token: JWT token for authenticated requests
        """
        self.base_url = f"https://{url}"
        self.token = token
        self.session = requests.Session()
        if token:
            self.session.headers.update({"Authorization": f"Bearer {token}"})
    
    def get_script(self, username: str, script: str, version: Optional[str] = None, 
                   token: Optional[str] = None) -> tuple:
        """
        Retrieve a script from shebang.run
        
        Args:
            username: Script owner's username
            script: Script name
            version: Optional version tag (@latest, @v1, @dev, etc.)
            token: Optional share token for private scripts
        
        Returns:
            Tuple of (content, metadata) where metadata includes encryption info
        """
        script_path = f"{script}@{version}" if version else script
        url = f"{self.base_url}/{username}/{script_path}"
        
        params = {}
        if token:
            params["token"] = token
        
        response = self.session.get(url, params=params)
        response.raise_for_status()
        
        metadata = {
            "encrypted": response.headers.get("X-Encrypted") == "true",
            "version": response.headers.get("X-Script-Version"),
            "checksum": response.headers.get("X-Script-Checksum"),
            "key_id": response.headers.get("X-Encryption-KeyID"),
            "wrapped_key": response.headers.get("X-Wrapped-Key")
        }
        
        return response.content if metadata["encrypted"] else response.text, metadata
    
    def get_metadata(self, username: str, script: str) -> dict:
        """Get script metadata"""
        url = f"{self.base_url}/{username}/{script}/meta"
        response = self.session.get(url)
        response.raise_for_status()
        return response.json()
    
    def verify_signature(self, username: str, script: str) -> dict:
        """Verify script signature"""
        url = f"{self.base_url}/{username}/{script}/verify"
        response = self.session.get(url)
        response.raise_for_status()
        return response.json()
    
    # Authentication
    def register(self, username: str, email: str, password: str) -> dict:
        """Register a new user"""
        url = f"{self.base_url}/api/auth/register"
        response = self.session.post(url, json={
            "username": username,
            "email": email,
            "password": password
        })
        response.raise_for_status()
        data = response.json()
        if "token" in data:
            self.token = data["token"]
            self.session.headers.update({"Authorization": f"Bearer {self.token}"})
        return data
    
    def login(self, username: str, password: str) -> dict:
        """Login and get JWT token"""
        url = f"{self.base_url}/api/auth/login"
        response = self.session.post(url, json={
            "username": username,
            "password": password
        })
        response.raise_for_status()
        data = response.json()
        if "token" in data:
            self.token = data["token"]
            self.session.headers.update({"Authorization": f"Bearer {self.token}"})
        return data
    
    # Script Management
    def list_scripts(self) -> list:
        """List user's scripts (requires authentication)"""
        url = f"{self.base_url}/api/scripts"
        response = self.session.get(url)
        response.raise_for_status()
        return response.json()
    
    def create_script(self, name: str, content: str, description: str = "", 
                     visibility: str = "private", keypair_id: Optional[int] = None) -> dict:
        """Create a new script"""
        url = f"{self.base_url}/api/scripts"
        data = {
            "name": name,
            "content": content,
            "description": description,
            "visibility": visibility
        }
        if keypair_id:
            data["keypair_id"] = keypair_id
        
        response = self.session.post(url, json=data)
        response.raise_for_status()
        return response.json()
    
    def update_script(self, script_id: int, content: Optional[str] = None,
                     description: Optional[str] = None, visibility: Optional[str] = None,
                     tag: Optional[str] = None, keypair_id: Optional[int] = None) -> dict:
        """Update a script (creates new version if content changed)"""
        url = f"{self.base_url}/api/scripts/{script_id}"
        data = {}
        if content:
            data["content"] = content
        if description:
            data["description"] = description
        if visibility:
            data["visibility"] = visibility
        if tag:
            data["tag"] = tag
        if keypair_id:
            data["keypair_id"] = keypair_id
        
        response = self.session.put(url, json=data)
        response.raise_for_status()
        return response.json() if response.text else {}
    
    def delete_script(self, script_id: int):
        """Delete a script"""
        url = f"{self.base_url}/api/scripts/{script_id}"
        response = self.session.delete(url)
        response.raise_for_status()
    
    def generate_share_token(self, script_id: int) -> str:
        """Generate a share token for a private script"""
        url = f"{self.base_url}/api/scripts/{script_id}/share"
        response = self.session.post(url)
        response.raise_for_status()
        return response.json()["token"]
    
    def revoke_share_token(self, script_id: int, token: str):
        """Revoke a share token"""
        url = f"{self.base_url}/api/scripts/{script_id}/share/{token}"
        response = self.session.delete(url)
        response.raise_for_status()
    
    # Key Management
    def list_keys(self) -> list:
        """List user's keypairs"""
        url = f"{self.base_url}/api/keys"
        response = self.session.get(url)
        response.raise_for_status()
        return response.json()
    
    def generate_key(self, name: str) -> dict:
        """Generate a new keypair (returns private key - save it!)"""
        url = f"{self.base_url}/api/keys/generate"
        response = self.session.post(url, json={"name": name})
        response.raise_for_status()
        return response.json()
    
    def import_key(self, name: str, public_key: str) -> dict:
        """Import an existing public key"""
        url = f"{self.base_url}/api/keys/import"
        response = self.session.post(url, json={
            "name": name,
            "public_key": public_key
        })
        response.raise_for_status()
        return response.json()
    
    def delete_key(self, key_id: int):
        """Delete a keypair"""
        url = f"{self.base_url}/api/keys/{key_id}"
        response = self.session.delete(url)
        response.raise_for_status()
    
    # Account Management
    def change_password(self, current_password: str, new_password: str):
        """Change account password"""
        url = f"{self.base_url}/api/account/password"
        response = self.session.put(url, json={
            "current_password": current_password,
            "new_password": new_password
        })
        response.raise_for_status()
    
    def export_data(self) -> dict:
        """Export all user data (GDPR)"""
        url = f"{self.base_url}/api/account/export"
        response = self.session.get(url)
        response.raise_for_status()
        return response.json()
    
    def delete_account(self):
        """Delete account permanently"""
        url = f"{self.base_url}/api/account"
        response = self.session.delete(url)
        response.raise_for_status()
    
    def create_api_token(self, name: str) -> dict:
        """Create API token for CLI access"""
        url = f"{self.base_url}/api/account/tokens"
        response = self.session.post(url, json={"name": name})
        response.raise_for_status()
        return response.json()
    
    # Secrets Management
    def list_secrets(self) -> list:
        """List all secrets"""
        url = f"{self.base_url}/api/secrets"
        response = self.session.get(url)
        response.raise_for_status()
        return response.json()
    
    def create_secret(self, key_name: str, value: str, expires_at: Optional[str] = None) -> dict:
        """Create or update a secret"""
        url = f"{self.base_url}/api/secrets"
        payload = {"key_name": key_name, "value": value}
        if expires_at:
            payload["expires_at"] = expires_at
        response = self.session.post(url, json=payload)
        response.raise_for_status()
        return response.json()
    
    def get_secret(self, key_name: str) -> str:
        """Get secret value"""
        url = f"{self.base_url}/api/secrets/{key_name}/value"
        response = self.session.get(url)
        response.raise_for_status()
        return response.json()["value"]
    
    def delete_secret(self, key_name: str):
        """Delete a secret"""
        url = f"{self.base_url}/api/secrets/{key_name}"
        response = self.session.delete(url)
        response.raise_for_status()
    
    def get_secret_audit(self, key_name: str) -> list:
        """Get audit log for a secret"""
        url = f"{self.base_url}/api/secrets/{key_name}/audit"
        response = self.session.get(url)
        response.raise_for_status()
        return response.json()
    
    # Script Sharing
    def list_script_access(self, script_id: int) -> list:
        """List access control for a script"""
        url = f"{self.base_url}/api/scripts/{script_id}/access"
        response = self.session.get(url)
        response.raise_for_status()
        return response.json()
    
    def add_script_access(self, script_id: int, usernames: list):
        """Add users to script ACL"""
        url = f"{self.base_url}/api/scripts/{script_id}/access"
        response = self.session.post(url, json={
            "access_type": "user",
            "usernames": usernames
        })
        response.raise_for_status()
    
    def remove_script_access(self, script_id: int, access_id: int):
        """Remove access from script ACL"""
        url = f"{self.base_url}/api/scripts/{script_id}/access/{access_id}"
        response = self.session.delete(url)
        response.raise_for_status()
    
    def list_shared_scripts(self) -> list:
        """List scripts shared with you"""
        url = f"{self.base_url}/api/shared/scripts"
        response = self.session.get(url)
        response.raise_for_status()
        return response.json()


def run(username: str, script: str, key: Optional[str] = None, 
        eval: bool = False, accept: bool = False, url: str = "shebang.run",
        version: Optional[str] = None, token: Optional[str] = None) -> Any:
    """
    Convenience function to fetch and optionally execute a script
    
    Args:
        username: Script owner's username (required)
        script: Script name (required)
        key: Private key contents for decryption (optional)
        eval: If True, evaluate the script in Python (default: False)
        accept: If True, skip confirmation prompt when eval=True (default: False)
        url: Base URL (default: shebang.run)
        version: Version tag (optional, e.g., "latest", "v1", "dev")
        token: Share token for private scripts (optional)
    
    Returns:
        Script content as string if eval=False, or result of eval() if eval=True
    
    Examples:
        # Just fetch the script
        content = run(username="mpruitt", script="bashtest")
        
        # Fetch encrypted script with private key
        content = run(username="mpruitt", script="private", key="-----BEGIN PRIVATE KEY-----\\n...")
        
        # Fetch and evaluate with confirmation
        run(username="mpruitt", script="myscript", eval=True)
    """
    client = ShebangClient(url=url)
    
    # Fetch the script
    content, metadata = client.get_script(username, script, version=version, token=token)
    
    # If encrypted and key is provided, decrypt
    if metadata.get("encrypted") and key:
        try:
            from cryptography.hazmat.primitives import serialization, hashes
            from cryptography.hazmat.primitives.asymmetric import padding
            from cryptography.hazmat.backends import default_backend
            import nacl.bindings
            
            # Parse private key
            private_key = serialization.load_pem_private_key(
                key.encode() if isinstance(key, str) else key,
                password=None,
                backend=default_backend()
            )
            
            # Get wrapped key from headers
            wrapped_key_hex = metadata.get("wrapped_key")
            if not wrapped_key_hex:
                raise Exception("No wrapped key found in response")
            
            # Unwrap symmetric key with RSA private key
            wrapped_key = bytes.fromhex(wrapped_key_hex)
            symmetric_key = private_key.decrypt(
                wrapped_key,
                padding.OAEP(
                    mgf=padding.MGF1(algorithm=hashes.SHA256()),
                    algorithm=hashes.SHA256(),
                    label=None
                )
            )
            
            # Decrypt content with XChaCha20-Poly1305
            nonce_size = 24
            nonce = content[:nonce_size]
            ciphertext = content[nonce_size:]
            
            decrypted = nacl.bindings.crypto_aead_xchacha20poly1305_ietf_decrypt(
                ciphertext,
                None,  # no additional data
                nonce,
                symmetric_key
            )
            
            content = decrypted.decode('utf-8')
            
        except ImportError:
            raise ImportError("Decryption requires: pip install cryptography pynacl")
        except Exception as e:
            raise Exception(f"Decryption failed: {e}")
    elif metadata.get("encrypted") and not key:
        # Return encrypted bytes if no key provided
        return content
    
    # Convert bytes to string if needed
    if isinstance(content, bytes):
        content = content.decode('utf-8')
    
    # If not evaluating, just return the content
    if not eval:
        return content
    
    # If evaluating, show confirmation unless accept=True
    if not accept:
        print("=" * 60)
        print(f"Script: {username}/{script}")
        print("=" * 60)
        print(content)
        print("=" * 60)
        response = input("Execute this script? (y/N): ").strip().lower()
        if response != 'y':
            print("Execution cancelled.")
            return None
    
    # Execute the script
    try:
        import subprocess
        import tempfile
        import os
        import sys
        
        # Check for shebang
        lines = content.split('\n')
        if lines[0].startswith('#!'):
            shebang = lines[0].lower()
            
            # If shebang contains 'python', execute in current context with exec
            if 'python' in shebang:
                exec_globals = {}
                exec(content, exec_globals)
                return exec_globals
            else:
                # Execute with specified interpreter (bash, sh, etc.)
                with tempfile.NamedTemporaryFile(mode='w', suffix='.sh', delete=False) as f:
                    f.write(content)
                    temp_path = f.name
                
                try:
                    os.chmod(temp_path, 0o755)
                    result = subprocess.run(
                        [temp_path],
                        capture_output=True,
                        text=True,
                        shell=False
                    )
                    
                    if result.stdout:
                        print(result.stdout, end='')
                    if result.stderr:
                        print(result.stderr, end='', file=sys.stderr)
                    
                    return result.returncode
                finally:
                    os.unlink(temp_path)
        else:
            # No shebang, execute as Python
            exec_globals = {}
            exec(content, exec_globals)
            return exec_globals
    except Exception as e:
        print(f"Error executing script: {e}")
        raise
