// Background service worker

chrome.runtime.onInstalled.addListener(() => {
  console.log('shebang.run Key Manager installed');
});

// Listen for messages from content script
chrome.runtime.onMessage.addListener((request, sender, sendResponse) => {
  if (request.action === 'getKeys') {
    chrome.storage.local.get('keys', (result) => {
      sendResponse({ keys: result.keys || [] });
    });
    return true; // Keep channel open for async response
  }
});
