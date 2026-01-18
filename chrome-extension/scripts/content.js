// Content script - runs on shebang.run pages
// Detects encrypted scripts and auto-injects private keys

(function() {
  'use strict';
  
  // Check if we're on the script editor page with an ID (editing existing script)
  if (window.location.pathname === '/script-editor' && window.location.search.includes('id=')) {
    initScriptEditor();
  }
  
  async function initScriptEditor() {
    console.log('[shebang.run] Extension loaded on script editor');
    
    // Wait for page to load
    await waitForElement('#code-editor');
    console.log('[shebang.run] Code editor found');
    
    // Check if script needs decryption
    const needsDecryption = await checkIfEncrypted();
    console.log('[shebang.run] Needs decryption:', needsDecryption);
    
    if (needsDecryption) {
      console.log('[shebang.run] Encrypted script detected');
      
      // Get stored keys
      const { keys = [] } = await chrome.storage.local.get('keys');
      console.log('[shebang.run] Found keys:', keys.length);
      
      if (keys.length === 0) {
        showNotification('No keys found. Add a key in the extension popup.');
        return;
      }
      
      // If only one key, auto-decrypt
      if (keys.length === 1) {
        console.log('[shebang.run] Auto-decrypting with single key');
        await autoDecrypt(keys[0]);
      } else {
        // Show key selector
        console.log('[shebang.run] Showing key selector');
        showKeySelector(keys);
      }
    }
  }
  
  async function checkIfEncrypted() {
    console.log('[shebang.run] Checking if script is encrypted...');
    
    // Wait for Alpine.js to initialize
    await new Promise(resolve => setTimeout(resolve, 1000));
    
    // Check if decrypt button exists and is visible
    const decryptBtn = document.querySelector('button[\\@click="decryptScript"]');
    console.log('[shebang.run] Decrypt button found:', !!decryptBtn);
    
    if (!decryptBtn) return false;
    
    // Check if it's actually visible (not hidden by x-show)
    const isVisible = decryptBtn.offsetParent !== null;
    console.log('[shebang.run] Decrypt button visible:', isVisible);
    
    return isVisible;
  }
  
  async function autoDecrypt(key) {
    console.log('[shebang.run] Auto-decrypting with key:', key.name);
    
    // Find the private key textarea
    const textarea = document.querySelector('textarea[x-model="privateKey"]');
    if (!textarea) {
      console.error('[shebang.run] Private key textarea not found');
      return;
    }
    
    // Inject the private key
    textarea.value = key.content;
    textarea.dispatchEvent(new Event('input', { bubbles: true }));
    
    // Trigger Alpine.js update
    const editorEl = document.querySelector('[x-data*="scriptEditor"]');
    if (editorEl && editorEl.__x) {
      editorEl.__x.$data.privateKey = key.content;
    }
    
    // Wait a bit for Alpine to update
    await new Promise(resolve => setTimeout(resolve, 100));
    
    // Click the decrypt button
    const decryptBtn = document.querySelector('button[type="button"][\\@click="decryptScript"]');
    if (decryptBtn) {
      decryptBtn.click();
      showNotification(`Decrypting with key: ${key.name}`);
    }
  }
  
  function showKeySelector(keys) {
    const selector = document.createElement('div');
    selector.id = 'shebang-key-selector';
    selector.style.cssText = `
      position: fixed;
      top: 80px;
      right: 20px;
      background: white;
      border: 2px solid #4f46e5;
      border-radius: 8px;
      padding: 16px;
      box-shadow: 0 4px 6px rgba(0,0,0,0.1);
      z-index: 10000;
      min-width: 250px;
    `;
    
    selector.innerHTML = `
      <div style="font-weight: 600; margin-bottom: 12px; color: #4f46e5;">
        🔐 Select Key to Decrypt
      </div>
      ${keys.map((key, index) => `
        <button data-index="${index}" style="
          display: block;
          width: 100%;
          padding: 8px;
          margin-bottom: 8px;
          background: #f3f4f6;
          border: 1px solid #d1d5db;
          border-radius: 4px;
          cursor: pointer;
          text-align: left;
        ">
          <div style="font-weight: 600;">${escapeHtml(key.name)}</div>
          <div style="font-size: 10px; color: #6b7280; font-family: monospace;">${key.fingerprint}</div>
        </button>
      `).join('')}
      <button id="closeSelector" style="
        width: 100%;
        padding: 8px;
        background: #e5e7eb;
        border: 1px solid #d1d5db;
        border-radius: 4px;
        cursor: pointer;
      ">Cancel</button>
    `;
    
    document.body.appendChild(selector);
    
    // Add click handlers
    selector.querySelectorAll('button[data-index]').forEach(btn => {
      btn.addEventListener('click', async () => {
        const index = parseInt(btn.dataset.index);
        await autoDecrypt(keys[index]);
        selector.remove();
      });
    });
    
    document.getElementById('closeSelector').addEventListener('click', () => {
      selector.remove();
    });
  }
  
  function showNotification(message) {
    const notification = document.createElement('div');
    notification.style.cssText = `
      position: fixed;
      top: 20px;
      right: 20px;
      background: #10b981;
      color: white;
      padding: 12px 20px;
      border-radius: 6px;
      box-shadow: 0 4px 6px rgba(0,0,0,0.1);
      z-index: 10001;
      font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif;
    `;
    notification.textContent = message;
    document.body.appendChild(notification);
    
    setTimeout(() => {
      notification.remove();
    }, 3000);
  }
  
  function waitForElement(selector) {
    return new Promise(resolve => {
      if (document.querySelector(selector)) {
        return resolve(document.querySelector(selector));
      }
      
      const observer = new MutationObserver(() => {
        if (document.querySelector(selector)) {
          observer.disconnect();
          resolve(document.querySelector(selector));
        }
      });
      
      observer.observe(document.body, {
        childList: true,
        subtree: true
      });
    });
  }
  
  function escapeHtml(text) {
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
  }
})();
