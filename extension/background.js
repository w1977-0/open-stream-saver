const OPTIONAL_ORIGINS = ["http://*/*", "https://*/*"];
const MAX_STREAMS_PER_TAB = 40;
const MAX_URL_LENGTH = 4096;
const NATIVE_HOST = "com.w1977_0.open_stream_saver";

function storageKey(tabId) {
  return `streams:${tabId}`;
}

function isCandidate(url) {
  if (typeof url !== "string" || url.length === 0 || url.length > MAX_URL_LENGTH) return false;
  try {
    const parsed = new URL(url);
    if (parsed.protocol !== "http:" && parsed.protocol !== "https:") return false;
    const pathname = parsed.pathname.toLowerCase();
    return pathname.endsWith(".m3u8") || pathname.endsWith(".mp4") || pathname.endsWith(".mpd");
  } catch {
    return false;
  }
}

function kindFor(url) {
  const pathname = new URL(url).pathname.toLowerCase();
  if (pathname.endsWith(".m3u8")) return "m3u8";
  if (pathname.endsWith(".mpd")) return "mpd";
  return "mp4";
}

function safeFileName(value) {
  if (typeof value !== "string" || value.length === 0 || value.length > 180) return undefined;
  if (value.includes("/") || value.includes("\\") || value === "." || value === "..") return undefined;
  return value;
}

async function recordCandidate(details) {
  if (details.tabId < 0 || !isCandidate(details.url)) return;

  const key = storageKey(details.tabId);
  const existing = (await chrome.storage.session.get(key))[key] || [];
  const normalizedUrl = details.url;
  const withoutDuplicate = existing.filter((entry) => entry.url !== normalizedUrl);
  const next = [
    {
      url: normalizedUrl,
      kind: kindFor(normalizedUrl),
      discoveredAt: Date.now(),
      initiator: typeof details.initiator === "string" ? details.initiator : "",
    },
    ...withoutDuplicate,
  ].slice(0, MAX_STREAMS_PER_TAB);

  await chrome.storage.session.set({ [key]: next });
}

const requestFilter = { urls: ["http://*/*", "https://*/*"] };

async function discoveryIsEnabled() {
  return chrome.permissions.contains({ origins: OPTIONAL_ORIGINS });
}

async function syncObserver() {
  const enabled = await discoveryIsEnabled();
  const registered = chrome.webRequest.onCompleted.hasListener(recordCandidate);

  if (enabled && !registered) {
    chrome.webRequest.onCompleted.addListener(recordCandidate, requestFilter);
  }
  if (!enabled && registered) {
    chrome.webRequest.onCompleted.removeListener(recordCandidate);
  }
}

async function requestLocalSave(message) {
  if (!message?.acknowledgeRights) {
    return { error: "Confirm that you have the right to save this public media first." };
  }
  if (!isCandidate(message.url)) {
    return { error: "Only discovered public HTTP(S) .mp4, .m3u8, or .mpd URLs can be sent locally." };
  }
  const workers = Number(message.workers);
  const variant = Number(message.variant);
  const response = await chrome.runtime.sendNativeMessage(NATIVE_HOST, {
    action: "download",
    url: message.url,
    fileName: safeFileName(message.fileName),
    workers: Number.isInteger(workers) && workers >= 1 && workers <= 32 ? workers : 4,
    variant: Number.isInteger(variant) ? variant : -1,
    acknowledgeRights: true,
  });
  return response || { error: "The local host returned no response." };
}

chrome.runtime.onInstalled.addListener(() => {
  void syncObserver();
});
chrome.runtime.onStartup.addListener(() => {
  void syncObserver();
});
chrome.permissions.onAdded.addListener(() => {
  void syncObserver();
});
chrome.permissions.onRemoved.addListener(() => {
  void syncObserver();
});
chrome.tabs.onRemoved.addListener((tabId) => {
  void chrome.storage.session.remove(storageKey(tabId));
});

chrome.runtime.onMessage.addListener((message, sender, sendResponse) => {
  const run = async () => {
    switch (message?.type) {
      case "status":
        return { enabled: await discoveryIsEnabled() };
      case "enableDiscovery": {
        const granted = await chrome.permissions.request({ origins: OPTIONAL_ORIGINS });
        await syncObserver();
        return { granted, enabled: await discoveryIsEnabled() };
      }
      case "disableDiscovery": {
        await chrome.permissions.remove({ origins: OPTIONAL_ORIGINS });
        await syncObserver();
        return { enabled: false };
      }
      case "getStreams": {
        const tabId = Number(message.tabId);
        if (!Number.isInteger(tabId) || tabId < 0) return { streams: [] };
        const key = storageKey(tabId);
        const streams = (await chrome.storage.session.get(key))[key] || [];
        return { streams };
      }
      case "clearStreams": {
        const tabId = Number(message.tabId);
        if (Number.isInteger(tabId) && tabId >= 0) {
          await chrome.storage.session.remove(storageKey(tabId));
        }
        return { cleared: true };
      }
      case "saveLocally":
        return requestLocalSave(message);
      default:
        return { error: "Unknown message." };
    }
  };

  run()
    .then(sendResponse)
    .catch((error) => sendResponse({ error: error?.message || "Extension error." }));
  return true;
});
