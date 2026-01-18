"""
shebangrun - Python client library for shebang.run

A helper library to interact with shebang.run API and execute remote scripts.
"""

__version__ = "0.1.0"

from .client import ShebangClient, run

__all__ = ["ShebangClient", "run"]
