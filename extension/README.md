# Chrome extension

This Manifest V3 extension observes completed public `http` and `https` requests whose URL paths end in `.m3u8` or `.mp4`. It is intentionally **read-only**: it does not alter requests, read cookies, capture credentials, inspect page content, or bypass DRM and other access controls.

## Load locally

1. Open `chrome://extensions` in Chrome or Chromium.
2. Turn on **Developer mode**.
3. Choose **Load unpacked** and select this `extension/` directory.
4. Open the extension popup and click **Enable discovery**. Chrome will ask for permission to observe public HTTP/HTTPS pages.
5. Visit a page where you are authorized to save the content. The popup can show eligible public URL requests recorded for that tab and copy a CLI command.

## Data handling

Discovered URLs are kept in `chrome.storage.session`, capped at 40 per tab, and removed when the tab closes. The extension sends no data to a remote service. Disabling discovery removes the optional host permissions; use **Clear** to remove the current tab's recorded URLs immediately.

## What is intentionally unsupported

The extension is not a video player or universal downloader. It does not work around logins, subscriptions, regional restrictions, DRM, encrypted streams, cookies, authorization headers, request modification, or page-script injection. It cannot determine whether a URL is authorized; the user must make that judgment before using the copied CLI command.
