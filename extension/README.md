# Open Stream Saver extension

This Chrome Manifest V3 extension observes completed public `http` and `https` requests whose URL paths end in `.m3u8`, `.mpd`, or `.mp4`. It is intentionally **read-only**: it does not alter requests, read cookies, capture credentials, inspect page content, inject hooks, collect headers, or bypass DRM and other access controls.

## Load locally

1. Open `chrome://extensions` in Chrome or Chromium.
2. Turn on **Developer mode**.
3. Choose **Load unpacked** and select this `extension/` directory.
4. Open the extension popup and click **Enable discovery**. Chrome will ask for optional HTTP/HTTPS site access.
5. Visit a page where you are authorized to save the content. The popup lists eligible public URL requests recorded for that tab.

## Optional local save

The popup can send one user-reviewed public URL to the local `open-stream-saver-host` through Chrome Native Messaging. This is opt-in and requires all of the following:

1. Install the matching local-host binary from a release archive.
2. Register the host manifest with **your exact extension ID** in `allowed_origins`.
3. Tick the rights acknowledgement before choosing **Save locally**.

See [`../native-host/README.md`](../native-host/README.md) for the manifest template and platform registration locations. The message contains only an action, URL, worker count, optional HLS variant, and acknowledgement. It never includes Cookie, Referer, User-Agent, authorization headers, tokens, page data, or media keys.

## Data handling

Discovered URLs are kept in `chrome.storage.session`, capped at 40 per tab, and removed when the tab closes. The extension sends no data to a remote service. Disabling discovery removes optional host permissions; use **Clear** to remove the current tab's recorded URLs immediately.

## What is intentionally unsupported

The extension is not a video player or universal downloader. It does not work around logins, subscriptions, paywalls, regional restrictions, DRM, encrypted streams, cookies, authorization headers, request modification, page-script injection, or browser API debugging. It cannot determine whether a URL is authorized; the user must make that judgment before copying a command or using the local-save action.
