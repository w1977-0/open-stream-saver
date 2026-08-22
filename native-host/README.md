# Native Messaging host setup

The optional native host makes the extension's **Save locally** button send one user-reviewed, public URL to the local Go downloader. It is not a server, does not run in the background, and never accepts cookies, credentials, request headers, tokens, DRM keys, or arbitrary output paths.

Chrome starts a native host as a local process and exchanges length-prefixed JSON through standard input/output. The host manifest must explicitly list the extension origin; wildcards are not permitted. Keep the host binary and its manifest under your own user account whenever possible. [Chrome Native Messaging documentation](https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging)

## 1. Install the local host binary

Download the `open-stream-saver-host` binary matching your platform from the release assets and place it in a stable directory. Do not use an untrusted binary or a path writable by other users.

## 2. Load or install the extension, then copy its extension ID

Open `chrome://extensions`, enable Developer mode if you are loading `extension/` locally, and copy the ID shown for **Open Stream Saver**. You must place that exact ID into `allowed_origins`; the native host will reject other extensions.

## 3. Create the host manifest

Copy [`com.w1977_0.open_stream_saver.template.json`](com.w1977_0.open_stream_saver.template.json). Replace both placeholders:

| Field | Required value |
| --- | --- |
| `path` | Absolute path to your local `open-stream-saver-host` executable. |
| `allowed_origins[0]` | `chrome-extension://YOUR_EXTENSION_ID/`, including the final slash. |

The host name in the manifest must remain `com.w1977_0.open_stream_saver` because that is the name used by the extension.

## 4. Register the manifest for your browser

| Platform | Per-user Chrome location or registration |
| --- | --- |
| Linux, Google Chrome | `~/.config/google-chrome/NativeMessagingHosts/com.w1977_0.open_stream_saver.json` |
| Linux, Chromium | `~/.config/chromium/NativeMessagingHosts/com.w1977_0.open_stream_saver.json` |
| macOS, Google Chrome | `~/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.w1977_0.open_stream_saver.json` |
| Windows | Store the JSON anywhere protected by your user account and create `HKCU\Software\Google\Chrome\NativeMessagingHosts\com.w1977_0.open_stream_saver` with its default value set to the JSON's full path. |

Restart Chrome after registration. The popup will explain if Chrome cannot find the host.

## Security contract

The extension passes only this fixed request shape: action, public URL, worker count, optional HLS variant index, and an explicit acknowledgement. Unknown fields are rejected by the host. It writes to your `Downloads` directory with a generated safe base filename; it refuses arbitrary paths and will not overwrite an existing file.

> Do not edit the extension to relay cookies, authorization headers, tokens, browser storage, or DRM material. Requests that add those capabilities are out of scope.
