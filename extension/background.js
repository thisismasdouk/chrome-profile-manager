importScripts("config.js", "ExtPay.js");
if (EXTPAY_ID) ExtPay(EXTPAY_ID).startBackground();

// Open the manager right after install so new users see how it works.
chrome.runtime.onInstalled.addListener((details) => {
  if (details.reason === "install") chrome.tabs.create({ url: chrome.runtime.getURL("manager.html?welcome=1") });
});

// Clicking the toolbar icon opens the manager in a full tab (or focuses it if already open).
chrome.action.onClicked.addListener(async () => {
  const url = chrome.runtime.getURL("manager.html");
  try {
    const contexts = await chrome.runtime.getContexts({ contextTypes: ["TAB"], documentUrls: [url] });
    if (contexts.length && contexts[0].tabId >= 0) {
      await chrome.tabs.update(contexts[0].tabId, { active: true });
      if (contexts[0].windowId >= 0) await chrome.windows.update(contexts[0].windowId, { focused: true });
      return;
    }
  } catch (e) { /* older Chrome: just open a new tab */ }
  await chrome.tabs.create({ url });
});
