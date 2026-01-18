# Quick Install Guide

## Install the Extension

1. **Open Chrome Extensions**
   - Go to `chrome://extensions/`
   - Or click the puzzle icon → "Manage Extensions"

2. **Enable Developer Mode**
   - Toggle "Developer mode" in the top right corner

3. **Load the Extension**
   - Click "Load unpacked"
   - Navigate to: `shebangrun/chrome-extension`
   - Click "Select Folder"

4. **Pin the Extension** (optional)
   - Click the puzzle icon in Chrome toolbar
   - Find "shebang.run Key Manager"
   - Click the pin icon

## Add Your First Key

1. Click the extension icon
2. Click "Add Private Key"
3. Enter a name: `my-key`
4. Paste your private key (PKCS8 format)
5. Click "Save Key"

## Test It

1. Go to https://shebang.run
2. Create an encrypted private script
3. Click "Edit" on the encrypted script
4. The extension will automatically:
   - Detect encryption
   - Inject your private key
   - Click decrypt
   - Show the decrypted content

## Done!

You can now edit encrypted scripts without manually pasting your private key every time.

## Troubleshooting

**Extension not detecting encryption:**
- Make sure you're on the script editor page
- Refresh the page after installing the extension
- Check that the script is actually encrypted

**Decryption fails:**
- Verify you're using the correct private key
- Ensure the key is in PKCS8 format (not PKCS1)
- Check browser console for errors (F12)

**Keys not saving:**
- Check Chrome storage permissions
- Try reloading the extension
- Clear extension data and re-add keys
