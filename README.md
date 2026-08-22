# Open Stream Saver

<!-- Banner placeholder: add a 1280 × 400 project banner here. -->
<!-- Demo placeholder: add a short local-only extension + CLI GIF here. -->

[![Go](https://img.shields.io/badge/Go-1.25%2B-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Manifest V3](https://img.shields.io/badge/Chrome-Manifest%20V3-4285F4?logo=googlechrome&logoColor=white)](https://developer.chrome.com/docs/extensions/)
[![License](https://img.shields.io/badge/License-Apache--2.0-1769E0)](LICENSE)
[![Build](https://github.com/w1977-0/open-stream-saver/actions/workflows/release.yml/badge.svg)](https://github.com/w1977-0/open-stream-saver/actions/workflows/release.yml)

A local-first Chrome extension and Go CLI for saving **authorized public media**. The extension observes eligible public `.m3u8` and `.mp4` requests without changing traffic; the CLI saves one direct file or completed, unencrypted HLS media playlist after an explicit rights acknowledgement.

> **Scope first.** Open Stream Saver does not decrypt DRM, import cookies or credentials, bypass logins, paywalls, subscriptions, regional restrictions, or proxies, modify network requests, download playlists in bulk, or accept encrypted / live HLS streams.

## Features

| Component | What it does |
| --- | --- |
| Chrome extension | Read-only Manifest V3 observer with optional site access, bounded per-tab deduplication, a modern popup, and a “copy CLI command” action. |
| Go CLI | A single binary with a clear `download` command, explicit authorization flag, local output control, retries, and terminal progress. |
| Direct files | Uses HTTP byte ranges when a server advertises stable range support; otherwise falls back to a single sequential request. |
| HLS media | Parses completed, unencrypted media playlists, fetches segments with a bounded worker pool, and invokes locally installed FFmpeg for a local remux. |
| Automation | Tag-triggered GitHub Actions builds release assets for Linux, macOS, and Windows. |

## Quick start

Download a matching release archive from the [Releases](../../releases) page and place the binary on your `PATH`. FFmpeg is only required when saving a supported HLS media playlist; direct-file downloads do not require it.

```bash
# Copy a command from the extension popup, then review the URL and your rights.
open-stream-saver download \
  --url 'https://example.org/your-authorized-video.mp4' \
  --acknowledge-rights \
  --output ./downloads/my-video.mp4
```

The `--acknowledge-rights` flag is deliberately required. Use it only for your own uploads, openly licensed material, content covered by an agreement, or media a website explicitly permits you to save.

## Extension setup

Until the extension is published through an appropriate browser-store review, load it locally:

1. Open `chrome://extensions` and enable **Developer mode**.
2. Select **Load unpacked** and choose the `extension/` folder.
3. Open the popup and choose **Enable discovery**. Chrome will ask for optional HTTP/HTTPS host access.
4. Open a page where you are authorized to save content. The popup lists the public `.m3u8` / `.mp4` requests it has observed for that tab. Review the source and copy the generated CLI command if appropriate.

The extension stores at most 40 URLs per tab in session storage and removes them when the tab closes. It neither sends data to a remote service nor reads headers, cookies, page bodies, or account information.

## Build from source

The CLI requires Go 1.25 or later. FFmpeg must be installed and available on `PATH` for HLS remuxing.

```bash
git clone https://github.com/w1977-0/open-stream-saver.git
cd open-stream-saver/cli
go test ./...
go build -o ../bin/open-stream-saver ./cmd/open-stream-saver
```

## Supported and intentionally unsupported inputs

| Input | Status | Reason |
| --- | --- | --- |
| Public HTTP(S) direct file | Supported | Uses safe URL checks and ordinary HTTP download behavior. |
| Completed, unencrypted HLS media playlist | Supported | Segment worker pool plus local FFmpeg remux. |
| HLS master playlist or live playlist | Rejected | The CLI does not select renditions or follow endless streams automatically. |
| Encrypted HLS, DRM, initialization maps, partial segments, byte ranges | Rejected | The project does not process protected or complex stream forms. |
| Login-only, paywalled, region-restricted or cookie-gated content | Rejected | The project does not carry credentials or implement access-control bypasses. |

## Architecture

The repository separates the browser extension from the Go CLI so that each component has a small, auditable responsibility. See [ARCHITECTURE.md](ARCHITECTURE.md) for the directory layout, dependency rationale and safety contract.

## Community

Questions and reproducible reports are welcome. Please read [CONTRIBUTING.md](CONTRIBUTING.md), [docs/FAQ.md](docs/FAQ.md), [SECURITY.md](SECURITY.md), and [CODE_OF_CONDUCT.md](CODE_OF_CONDUCT.md) before participating. Do not post private URLs, account information, cookies, tokens, DRM keys, or links to content you are not authorized to share.

## License

Licensed under the [Apache License 2.0](LICENSE). Third-party dependencies retain their own licenses; see [NOTICE.md](NOTICE.md).
