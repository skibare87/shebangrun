// Popup UI logic for key management

document.addEventListener('DOMContentLoaded', () => {
  loadKeys();
  
  document.getElementById('addKeyBtn').addEventListener('click', () => {
    document.getElementById('addKeyForm').classList.add('active');
  });
  
  document.getElementById('cancelBtn').addEventListener('click', () => {
    document.getElementById('addKeyForm').classList.remove('active');
    clearForm();
  });
  
  document.getElementById('saveKeyBtn').addEventListener('click', saveKey);
});

async function loadKeys() {
  const { keys = [] } = await chrome.storage.local.get('keys');
  const keyList = document.getElementById('keyList');
  
  if (keys.length === 0) {
    keyList.innerHTML = '<div class="empty-state">No keys stored.<br>Add a private key to get started.</div>';
    return;
  }
  
  keyList.innerHTML = keys.map((key, index) => `
    <div class="key-item">
      <div class="key-name">${escapeHtml(key.name)}</div>
      <div class="key-fingerprint">${key.fingerprint}</div>
      <div class="key-actions">
        <button class="btn-danger" data-index="${index}">Delete</button>
      </div>
    </div>
  `).join('');
  
  // Add delete handlers
  keyList.querySelectorAll('.btn-danger').forEach(btn => {
    btn.addEventListener('click', (e) => {
      const index = parseInt(e.target.dataset.index);
      deleteKey(index);
    });
  });
}

async function saveKey() {
  const name = document.getElementById('keyName').value.trim();
  const content = document.getElementById('keyContent').value.trim();
  
  if (!name || !content) {
    showStatus('Please provide both name and key content', 'error');
    return;
  }
  
  // Validate PEM format
  if (!content.includes('BEGIN') || !content.includes('PRIVATE KEY')) {
    showStatus('Invalid PEM format', 'error');
    return;
  }
  
  // Generate fingerprint (first 16 chars of key hash)
  const fingerprint = await generateFingerprint(content);
  
  const { keys = [] } = await chrome.storage.local.get('keys');
  
  // Check for duplicate names
  if (keys.some(k => k.name === name)) {
    showStatus('Key name already exists', 'error');
    return;
  }
  
  keys.push({
    name,
    content,
    fingerprint,
    created: new Date().toISOString()
  });
  
  await chrome.storage.local.set({ keys });
  
  showStatus('Key saved successfully!', 'success');
  document.getElementById('addKeyForm').classList.remove('active');
  clearForm();
  loadKeys();
}

async function deleteKey(index) {
  if (!confirm('Delete this key?')) return;
  
  const { keys = [] } = await chrome.storage.local.get('keys');
  keys.splice(index, 1);
  await chrome.storage.local.set({ keys });
  
  showStatus('Key deleted', 'success');
  loadKeys();
}

async function generateFingerprint(content) {
  const encoder = new TextEncoder();
  const data = encoder.encode(content);
  const hashBuffer = await crypto.subtle.digest('SHA-256', data);
  const hashArray = Array.from(new Uint8Array(hashBuffer));
  const hashHex = hashArray.map(b => b.toString(16).padStart(2, '0')).join('');
  return hashHex.substring(0, 16);
}

function clearForm() {
  document.getElementById('keyName').value = '';
  document.getElementById('keyContent').value = '';
}

function showStatus(message, type) {
  const status = document.getElementById('status');
  status.textContent = message;
  status.className = `status ${type}`;
  setTimeout(() => {
    status.textContent = '';
    status.className = 'status';
  }, 3000);
}

function escapeHtml(text) {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}
