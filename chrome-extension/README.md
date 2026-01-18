# shebang.run Key Manager - Chrome Extension

Securely manage your private keys and automatically decrypt encrypted scripts on shebang.run without server-side key storage.

## Features

- 🔐 **Secure Key Storage** - Private keys stored in Chrome's encrypted storage
- 🚀 **Auto-Decrypt** - Automatically decrypt encrypted scripts when editing
- 🔑 **Multiple Keys** - Manage multiple private keys with names
- 🎯 **Smart Detection** - Detects encrypted scripts and offers decryption
- 🔒 **Client-Side Only** - Keys never leave your browser
- ⚡ **One-Click** - Select key and decrypt instantly

## Installation

### From Source

1. Clone the repository:
```bash
git clone https://github.com/skibare87/shebangrun.git
cd shebangrun/chrome-extension
```

2. Open Chrome and go to `chrome://extensions/`

3. Enable "Developer mode" (toggle in top right)

4. Click "Load unpacked"

5. Select the `chrome-extension` folder

## Usage

### Adding a Private Key

1. Click the extension icon in Chrome toolbar
2. Click "Add Private Key"
3. Enter a name for the key (e.g., "my-signing-key")
4. Paste your private key in PEM format:
```
-----BEGIN PRIVATE KEY-----
...
-----END PRIVATE KEY-----
```
5. Click "Save Key"

### Auto-Decryption

1. Navigate to an encrypted script on shebang.run
2. Click "Edit" on the encrypted script
3. The extension automatically detects encryption
4. If you have one key: Auto-decrypts immediately
5. If you have multiple keys: Shows a selector to choose which key

### Managing Keys

- View all stored keys in the extension popup
- Each key shows its name and fingerprint
- Delete keys you no longer need
- Keys are stored securely in Chrome's encrypted storage

## Security

✅ **Private keys stored in Chrome's encrypted storage API**
✅ **Keys never transmitted to any server**
✅ **Only accessible on shebang.run domains**
✅ **Isolated from other websites**
✅ **Optional Chrome sync (can be disabled)**

## Permissions

- `storage` - Store private keys securely
- `activeTab` - Inject decryption into shebang.run pages
- `host_permissions` - Access shebang.run and localhost for testing

## How It Works

1. **Content Script** - Runs on shebang.run pages
2. **Detection** - Checks if script is encrypted (needsDecryption flag)
3. **Key Retrieval** - Gets keys from Chrome storage
4. **Injection** - Fills private key textarea
5. **Trigger** - Clicks decrypt button automatically
6. **Cleanup** - Key cleared from memory after decryption

## Development

### File Structure
```
chrome-extension/
├── manifest.json          # Extension configuration
├── popup.html            # Key management UI
├── popup.js              # Popup logic
├── scripts/
│   ├── content.js        # Page injection script
│   └── background.js     # Service worker
├── icons/
│   ├── icon16.png
│   ├── icon48.png
│   └── icon128.png
└── README.md
```

### Testing Locally

The extension works with both:
- Production: `https://shebang.run`
- Local dev: `http://localhost:8080`

## Privacy

- Keys stored locally in your browser only
- No analytics or tracking
- No external network requests
- Open source - audit the code yourself

## Compatibility

- Chrome 88+
- Edge 88+
- Brave (Chromium-based)
- Any Chromium-based browser

## License

MIT License - Same as shebang.run platform

## Links

- Platform: https://shebang.run
- GitHub: https://github.com/skibare87/shebangrun
- Documentation: https://shebang.run/docs
