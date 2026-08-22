# Open Stream Saver — Architecture

## Purpose

Open Stream Saver is a local-first learning project for **authorized public media**. It is intentionally split into a Chrome extension that observes publicly loaded media URLs and a Go CLI that downloads a user-supplied public URL. The extension never sends browser credentials to the CLI, and the CLI never accepts cookies, authorization headers, DRM keys, or account details.

## Directory tree

```text
open-stream-saver/
├── extension/                         # Chrome Manifest V3 extension
│   ├── manifest.json                   # Minimal API / optional host permissions
│   ├── background.js                   # Read-only request observer, dedupe and storage
│   ├── popup.html                      # Discovered public-stream list
│   ├── popup.js                        # Popup rendering and clipboard command generation
│   └── popup.css                       # Lightweight modern styling
├── cli/                               # Go command-line application
│   ├── cmd/open-stream-saver/main.go  # Executable entry point
│   ├── internal/app/                  # Cobra command and user-facing validation
│   ├── internal/direct/               # Public direct-file HTTP downloader with Range support
│   ├── internal/hls/                  # Unencrypted media-playlist parsing and segment worker pool
│   ├── internal/safety/               # URL/IP checks and explicit authorization acknowledgement
│   ├── internal/ffmpeg/               # Local FFmpeg availability and merge invocation
│   └── internal/progress/             # Terminal progress abstraction
├── docs/
│   ├── FAQ.md                         # English FAQ for users and contributors
│   ├── CONTRIBUTING.md                 # English contribution guide
│   ├── SECURITY.md                     # Responsible disclosure and boundaries
│   └── ARCHITECTURE.md                 # This document in repository form
├── .github/
│   ├── ISSUE_TEMPLATE/                # Safe issue forms without credential fields
│   └── workflows/release.yml          # Tag-triggered cross-build and GitHub Release
├── README.md                           # English project front page
├── README.zh-CN.md                     # Chinese companion guide
├── LICENSE                             # Apache License 2.0
└── go.mod
```

## Core dependencies

| Component | Dependency | Why it is included |
| --- | --- | --- |
| CLI command model | `github.com/spf13/cobra` | Provides clear commands, POSIX-style flags, generated help, validation hooks and future shell completion without building a custom flag framework. [1] |
| Terminal progress | `github.com/vbauerster/mpb/v8` | Shows byte and segment progress for local, concurrent work without hiding what is being downloaded. |
| HLS parsing | `github.com/Eyevinn/hls-m3u8` | A maintained Go HLS playlist library; the CLI uses it to enumerate only unencrypted media-playlist segments and rejects key-protected or master playlists. [2] |
| Cancellation | `golang.org/x/sync/errgroup` | Coordinates worker goroutines, cancellation and first-error propagation. |
| HTTP / filesystem / URL validation | Go standard library | Keeps direct-range downloading, retries, temp directories and safe filename handling transparent and easy to audit. |

## Safe interface contract

The extension is a **read-only observer**. It watches completed `http` and `https` requests matching `.m3u8` or `.mp4`, records only the URL, type, tab title and discovery time, limits each tab to a small number of entries, and stores them in `chrome.storage.session`. It does not inspect page bodies, request headers, response headers, cookies, accounts, or content protected by access control.

The CLI requires `--acknowledge-rights` before a network request is made. It only accepts public `http` / `https` URLs, rejects localhost and private IP destinations, downloads a single direct file or a non-encrypted HLS media playlist, and can invoke an already-installed local FFmpeg only for local merge/remux work. It rejects master playlists, live/endless playlists, `EXT-X-KEY`, `EXT-X-MAP`, byte-range segments and non-HTTP segment targets.

## References

[1]: https://github.com/spf13/cobra "Cobra — modern Go CLI"
[2]: https://github.com/Eyevinn/hls-m3u8 "Eyevinn — hls-m3u8"
