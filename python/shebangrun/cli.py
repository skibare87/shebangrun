#!/usr/bin/env python3
"""
shebang - CLI tool for shebang.run
"""

import sys
import os
import argparse
import json
from pathlib import Path

try:
    from shebangrun import ShebangClient, run
except ImportError:
    print("Error: shebangrun library not found", file=sys.stderr)
    print("Install with: pip install shebangrun", file=sys.stderr)
    sys.exit(1)

CONFIG_FILE = Path.home() / '.shebangrc'

def load_config():
    """Load configuration from ~/.shebangrc"""
    config = {}
    if CONFIG_FILE.exists():
        with open(CONFIG_FILE) as f:
            for line in f:
                line = line.strip()
                if line and not line.startswith('#') and '=' in line:
                    key, value = line.split('=', 1)
                    config[key] = value.strip('"')
    return config

def save_config(config):
    """Save configuration to ~/.shebangrc"""
    with open(CONFIG_FILE, 'w') as f:
        f.write('# shebang.run CLI configuration\n')
        for key, value in config.items():
            f.write(f'{key}="{value}"\n')
    CONFIG_FILE.chmod(0o600)

def cmd_login(args):
    """Login and generate API credentials"""
    print("shebang.run Login")
    print("=" * 50)
    print()
    
    url = input("Server URL [https://shebang.run]: ").strip() or "https://shebang.run"
    username = input("Username: ").strip()
    if not username:
        print("Error: Username required", file=sys.stderr)
        sys.exit(1)
    
    import getpass
    password = getpass.getpass("Password: ")
    if not password:
        print("Error: Password required", file=sys.stderr)
        sys.exit(1)
    
    key_path = input("Private Key Path (optional): ").strip()
    
    # Login
    print("Logging in...")
    client = ShebangClient(url=url.replace('https://', '').replace('http://', ''))
    
    try:
        response = client.login(username, password)
        token = response['token']
    except Exception as e:
        print(f"Error: Login failed - {e}", file=sys.stderr)
        sys.exit(1)
    
    # Generate API token
    print("Generating API credentials...")
    try:
        import datetime
        token_name = f"CLI-{datetime.datetime.now().strftime('%Y%m%d-%H%M%S')}"
        api_token = client.create_api_token(token_name)
        
        config = {
            'SHEBANG_URL': url,
            'SHEBANG_USERNAME': username,
            'SHEBANG_CLIENT_ID': api_token['client_id'],
            'SHEBANG_CLIENT_SECRET': api_token['client_secret'],
            'SHEBANG_KEY_PATH': key_path
        }
        save_config(config)
        
        print("✓ API credentials generated!")
        print()
        print(f"Client ID: {api_token['client_id']}")
        print(f"Client Secret: {api_token['client_secret'][:20]}...")
        print()
        print(f"Config saved to {CONFIG_FILE}")
        print("Keep your credentials secure!")
        
    except Exception as e:
        print(f"Error: Failed to generate API credentials - {e}", file=sys.stderr)
        sys.exit(1)

def cmd_list(args):
    """List scripts"""
    config = load_config()
    if not config.get('SHEBANG_CLIENT_ID'):
        print("Error: Not logged in. Run: shebang login", file=sys.stderr)
        sys.exit(1)
    
    client = ShebangClient(url=config['SHEBANG_URL'].replace('https://', '').replace('http://', ''))
    client.session.auth = (config['SHEBANG_CLIENT_ID'], config['SHEBANG_CLIENT_SECRET'])
    
    print("Your Scripts:")
    print("=" * 50)
    
    try:
        scripts = client.list_scripts()
        if not scripts:
            print("(no scripts)")
        else:
            for s in scripts:
                vis = s['visibility']
                color = '\033[0;32m' if vis == 'public' else '\033[1;33m' if vis == 'unlisted' else '\033[0;31m'
                print(f"{color}{s['name']}\033[0m (v{s['version']}) - {s.get('description', 'No description')} [{vis}]")
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)
    
    if args.community:
        print()
        print("Community Scripts:")
        print("=" * 50)
        
        try:
            import requests
            response = requests.get(
                f"{config['SHEBANG_URL']}/api/community/scripts",
                auth=(config['SHEBANG_CLIENT_ID'], config['SHEBANG_CLIENT_SECRET'])
            )
            response.raise_for_status()
            scripts = response.json()
            
            if not scripts:
                print("(no community scripts)")
            else:
                for s in scripts:
                    print(f"\033[0;32m{s['username']}/{s['name']}\033[0m (v{s['version']}) - {s.get('description', 'No description')}")
        except Exception as e:
            print(f"Error loading community scripts: {e}", file=sys.stderr)

def cmd_search(args):
    """Search scripts"""
    config = load_config()
    if not config.get('SHEBANG_CLIENT_ID'):
        print("Error: Not logged in. Run: shebang login", file=sys.stderr)
        sys.exit(1)
    
    if not args.query:
        print("Error: Search query required", file=sys.stderr)
        sys.exit(1)
    
    client = ShebangClient(url=config['SHEBANG_URL'].replace('https://', '').replace('http://', ''))
    client.session.auth = (config['SHEBANG_CLIENT_ID'], config['SHEBANG_CLIENT_SECRET'])
    
    query = args.query.lower()
    
    print(f"Searching for: {args.query}")
    print()
    
    try:
        scripts = client.list_scripts()
        found = False
        for s in scripts:
            if query in s['name'].lower() or query in s.get('description', '').lower():
                found = True
                vis = s['visibility']
                color = '\033[0;32m' if vis == 'public' else '\033[1;33m' if vis == 'unlisted' else '\033[0;31m'
                print(f"{color}{s['name']}\033[0m (v{s['version']}) - {s.get('description', 'No description')} [{vis}]")
        
        if not found:
            print("No matches in your scripts")
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
    
    if args.community:
        print()
        print("Community Results:")
        
        try:
            import requests
            response = requests.get(
                f"{config['SHEBANG_URL']}/api/community/scripts",
                auth=(config['SHEBANG_CLIENT_ID'], config['SHEBANG_CLIENT_SECRET'])
            )
            scripts = response.json()
            found = False
            for s in scripts:
                if query in s['name'].lower() or query in s.get('description', '').lower() or query in s['username'].lower():
                    found = True
                    print(f"\033[0;32m{s['username']}/{s['name']}\033[0m (v{s['version']}) - {s.get('description', 'No description')}")
            
            if not found:
                print("No matches in community")
        except Exception as e:
            print(f"Error: {e}", file=sys.stderr)

def cmd_get(args):
    """Get a script"""
    config = load_config()
    
    user = args.user or config.get('SHEBANG_USERNAME')
    if not user:
        print("Error: Username required. Use -u or login first.", file=sys.stderr)
        sys.exit(1)
    
    key_path = args.key or config.get('SHEBANG_KEY_PATH')
    key_content = None
    if key_path and os.path.exists(key_path):
        with open(key_path) as f:
            key_content = f.read()
    
    try:
        content = run(
            username=user,
            script=args.script,
            key=key_content,
            url=config.get('SHEBANG_URL', 'shebang.run').replace('https://', '').replace('http://', '')
        )
        
        if args.output:
            with open(args.output, 'w') as f:
                f.write(content)
            print(f"✓ Script saved to: {args.output}")
        else:
            print(content)
            
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

def cmd_run(args):
    """Run a script"""
    config = load_config()
    
    user = args.user or config.get('SHEBANG_USERNAME')
    if not user:
        print("Error: Username required. Use -u or login first.", file=sys.stderr)
        sys.exit(1)
    
    key_path = args.key or config.get('SHEBANG_KEY_PATH')
    key_content = None
    if key_path and os.path.exists(key_path):
        with open(key_path) as f:
            key_content = f.read()
    
    try:
        content = run(
            username=user,
            script=args.script,
            key=key_content,
            url=config.get('SHEBANG_URL', 'shebang.run').replace('https://', '').replace('http://', '')
        )
        
        # Save to temp file or specified output
        import tempfile
        if args.output:
            script_file = args.output
        else:
            fd, script_file = tempfile.mkstemp(suffix=f'-{args.script}')
            os.close(fd)
        
        with open(script_file, 'w') as f:
            f.write(content)
        os.chmod(script_file, 0o755)
        
        # Show script and prompt if not auto-accept
        if not args.accept:
            print()
            print("Script content:")
            print("=" * 60)
            print(content)
            print("=" * 60)
            print()
            confirm = input("Execute this script? (y/N): ").strip().lower()
            if confirm != 'y':
                if not args.output:
                    os.unlink(script_file)
                print("Execution cancelled")
                sys.exit(0)
        
        # Execute script
        print("Executing script...")
        print()
        
        import subprocess
        result = subprocess.run(
            [script_file] + (args.script_args or []),
            capture_output=False
        )
        
        # Cleanup
        if args.delete or not args.output:
            os.unlink(script_file)
        elif not args.output:
            print(f"\n✓ Script saved to: {script_file}")
        
        sys.exit(result.returncode)
        
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

def cmd_list_keys(args):
    """List keypairs"""
    config = load_config()
    if not config.get('SHEBANG_CLIENT_ID'):
        print("Error: Not logged in. Run: shebang login", file=sys.stderr)
        sys.exit(1)
    
    client = ShebangClient(url=config['SHEBANG_URL'].replace('https://', '').replace('http://', ''))
    client.session.auth = (config['SHEBANG_CLIENT_ID'], config['SHEBANG_CLIENT_SECRET'])
    
    try:
        keys = client.list_keys()
        if not keys:
            print("No keys found")
            return
        
        print("Your Keys:")
        print("=" * 50)
        for k in keys:
            print(f"{k['name']}")
            print(f"  Created: {k['created_at']}")
            print()
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

def cmd_create_key(args):
    """Create a new keypair"""
    config = load_config()
    if not config.get('SHEBANG_CLIENT_ID'):
        print("Error: Not logged in. Run: shebang login", file=sys.stderr)
        sys.exit(1)
    
    name = input("Key name: ").strip()
    if not name:
        print("Error: Key name required", file=sys.stderr)
        sys.exit(1)
    
    client = ShebangClient(url=config['SHEBANG_URL'].replace('https://', '').replace('http://', ''))
    client.session.auth = (config['SHEBANG_CLIENT_ID'], config['SHEBANG_CLIENT_SECRET'])
    
    try:
        print("Generating RSA-4096 keypair...")
        key = client.generate_key(name)
        
        if args.output:
            with open(args.output, 'w') as f:
                f.write(key['private_key'])
            print(f"✓ Private key saved to: {args.output}")
            print(f"✓ Public key stored in account: {name}")
        else:
            print(key['private_key'])
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

def cmd_delete_key(args):
    """Delete a keypair"""
    config = load_config()
    if not config.get('SHEBANG_CLIENT_ID'):
        print("Error: Not logged in. Run: shebang login", file=sys.stderr)
        sys.exit(1)
    
    client = ShebangClient(url=config['SHEBANG_URL'].replace('https://', '').replace('http://', ''))
    client.session.auth = (config['SHEBANG_CLIENT_ID'], config['SHEBANG_CLIENT_SECRET'])
    
    try:
        # Find key by name
        keys = client.list_keys()
        key = next((k for k in keys if k['name'] == args.keyname), None)
        
        if not key:
            print(f"Error: Key '{args.keyname}' not found", file=sys.stderr)
            sys.exit(1)
        
        confirm = input(f"Delete key '{args.keyname}'? (y/N): ").strip().lower()
        if confirm != 'y':
            print("Cancelled")
            sys.exit(0)
        
        client.delete_key(key['id'])
        print(f"✓ Key '{args.keyname}' deleted")
        
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

def cmd_put(args):
    """Upload a script"""
    config = load_config()
    if not config.get('SHEBANG_CLIENT_ID'):
        print("Error: Not logged in. Run: shebang login", file=sys.stderr)
        sys.exit(1)
    
    # Read content
    if args.stdin:
        content = sys.stdin.read()
    elif args.file:
        with open(args.file) as f:
            content = f.read()
    else:
        print("Error: Must specify -s/--stdin or -f/--file", file=sys.stderr)
        sys.exit(1)
    
    # Validate visibility
    visibility_map = {'priv': 'private', 'unlist': 'unlisted', 'public': 'public'}
    visibility = visibility_map.get(args.visibility, args.visibility)
    if visibility not in ['private', 'unlisted', 'public']:
        print("Error: Visibility must be priv, unlist, or public", file=sys.stderr)
        sys.exit(1)
    
    client = ShebangClient(url=config['SHEBANG_URL'].replace('https://', '').replace('http://', ''))
    client.session.auth = (config['SHEBANG_CLIENT_ID'], config['SHEBANG_CLIENT_SECRET'])
    
    try:
        # Get keypair ID if specified
        keypair_id = None
        if args.keyname:
            keys = client.list_keys()
            key = next((k for k in keys if k['name'] == args.keyname), None)
            if not key:
                print(f"Error: Key '{args.keyname}' not found", file=sys.stderr)
                sys.exit(1)
            keypair_id = key['id']
        elif visibility == 'private':
            print("Error: Private scripts require -k/--keyname", file=sys.stderr)
            sys.exit(1)
        
        print(f"Uploading script '{args.name}'...")
        result = client.create_script(
            name=args.name,
            content=content,
            description=args.description or "",
            visibility=visibility,
            keypair_id=keypair_id
        )
        
        print(f"✓ Script created: {args.name} (v{result['version']})")
        print(f"  URL: {config['SHEBANG_URL']}/{config['SHEBANG_USERNAME']}/{args.name}")
        
    except Exception as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)

def main():
    parser = argparse.ArgumentParser(
        prog='shebang',
        description='CLI tool for shebang.run',
        formatter_class=argparse.RawDescriptionHelpFormatter
    )
    
    subparsers = parser.add_subparsers(dest='command', help='Commands')
    
    # Login
    subparsers.add_parser('login', help='Login and generate API credentials')
    
    # List
    list_parser = subparsers.add_parser('list', help='List scripts')
    list_parser.add_argument('-c', '--community', action='store_true', help='Include community scripts')
    
    # Search
    search_parser = subparsers.add_parser('search', help='Search scripts')
    search_parser.add_argument('query', help='Search query')
    search_parser.add_argument('-c', '--community', action='store_true', help='Include community scripts')
    
    # Get
    get_parser = subparsers.add_parser('get', help='Download a script')
    get_parser.add_argument('script', help='Script name')
    get_parser.add_argument('-u', '--user', help='Username')
    get_parser.add_argument('-O', '--output', help='Output file')
    get_parser.add_argument('-k', '--key', help='Private key path')
    
    # Run
    run_parser = subparsers.add_parser('run', help='Download and execute a script')
    run_parser.add_argument('script', help='Script name')
    run_parser.add_argument('-u', '--user', help='Username')
    run_parser.add_argument('-O', '--output', help='Output file')
    run_parser.add_argument('-k', '--key', help='Private key path')
    run_parser.add_argument('-a', '--accept', action='store_true', help='Auto-accept execution')
    run_parser.add_argument('-d', '--delete', action='store_true', help='Delete after execution')
    run_parser.add_argument('script_args', nargs='*', help='Arguments to pass to script')
    
    # List keys
    subparsers.add_parser('list-keys', help='List your keypairs')
    
    # Create key
    create_key_parser = subparsers.add_parser('create-key', help='Generate a new keypair')
    create_key_parser.add_argument('-O', '--output', help='Save private key to file')
    
    # Delete key
    delete_key_parser = subparsers.add_parser('delete-key', help='Delete a keypair')
    delete_key_parser.add_argument('keyname', help='Key name to delete')
    
    # Put (upload script)
    put_parser = subparsers.add_parser('put', help='Upload a script')
    put_parser.add_argument('-n', '--name', required=True, help='Script name')
    put_parser.add_argument('-v', '--visibility', required=True, choices=['priv', 'unlist', 'public'], help='Visibility')
    put_parser.add_argument('-d', '--description', default='', help='Description')
    put_parser.add_argument('-k', '--keyname', help='Key name for encryption (required for private)')
    put_parser.add_argument('-s', '--stdin', action='store_true', help='Read from stdin')
    put_parser.add_argument('-f', '--file', help='Read from file')
    
    args = parser.parse_args()
    
    if not args.command:
        parser.print_help()
        sys.exit(0)
    
    if args.command == 'login':
        cmd_login(args)
    elif args.command == 'list':
        cmd_list(args)
    elif args.command == 'search':
        cmd_search(args)
    elif args.command == 'get':
        cmd_get(args)
    elif args.command == 'run':
        cmd_run(args)
    elif args.command == 'list-keys':
        cmd_list_keys(args)
    elif args.command == 'create-key':
        cmd_create_key(args)
    elif args.command == 'delete-key':
        cmd_delete_key(args)
    elif args.command == 'put':
        cmd_put(args)

if __name__ == '__main__':
    main()
