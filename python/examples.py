#!/usr/bin/env python3
"""
Example usage of shebangrun library
"""

from shebangrun import run, ShebangClient

def example_simple_fetch():
    """Example: Simple script fetching"""
    print("=== Simple Fetch ===")
    content = run(username="mpruitt", script="bashtest")
    print(content)
    print()

def example_with_confirmation():
    """Example: Execute with confirmation"""
    print("=== Execute with Confirmation ===")
    # This will show the script and ask for confirmation
    run(username="mpruitt", script="bashtest", eval=True, accept=False)
    print()

def example_client_usage():
    """Example: Using the full client"""
    print("=== Client Usage ===")
    
    client = ShebangClient(url="shebang.run")
    
    # Get script metadata
    meta = client.get_metadata(username="mpruitt", script="bashtest")
    print(f"Script: {meta['name']}")
    print(f"Version: {meta['version']}")
    print(f"Size: {meta['size']} bytes")
    print(f"Checksum: {meta['checksum']}")
    print()

def example_authenticated():
    """Example: Authenticated operations"""
    print("=== Authenticated Operations ===")
    
    client = ShebangClient(url="shebang.run")
    
    # Login (replace with your credentials)
    # client.login(username="myuser", password="mypassword")
    
    # List scripts
    # scripts = client.list_scripts()
    # for script in scripts:
    #     print(f"{script['name']} - v{script['version']}")
    
    print("(Login required - example commented out)")
    print()

if __name__ == "__main__":
    example_simple_fetch()
    example_client_usage()
    # example_with_confirmation()  # Uncomment to test interactive execution
    example_authenticated()
