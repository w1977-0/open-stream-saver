# Start here: FAQ, project scope, and contributing

Welcome to **Open Stream Saver**. This is a local-first project for users already authorized to save public media. The goal is not to become a universal downloader; it is to make one small, auditable workflow clear: observe an eligible public URL, review it, confirm your rights, and save it locally.

## What the project contains

The Chrome extension uses Manifest V3 and observes completed public `.m3u8`, `.mpd`, and `.mp4` requests after the user enables optional host access. It is read-only and retains a bounded, session-scoped list per tab. The Go CLI requires `--acknowledge-rights` and handles an ordinary public direct file, a completed unencrypted HLS presentation, or a narrow static unencrypted DASH profile. The optional Native Messaging host lets the user send one reviewed URL to the local engine without copying a terminal command.

The local host accepts only a fixed request schema: action, public URL, worker count, optional HLS variant index, and an explicit acknowledgement. It does not receive Cookie, Referer, User-Agent, authorization headers, tokens, credentials, browser storage, page data, encryption keys, or DRM material.

## Frequently asked questions

**Does the project bypass DRM, login, subscription, paywall, encryption, or regional protections?** No. It does not import cookies or credentials, inspect authorization headers, read media keys, modify requests, inject page scripts, implement proxies, decrypt AES/SAMPLE-AES, or work around DRM. Those requests are outside the project scope.

**Why can’t the extension see every video on a page?** Many pages use protected pipelines, `blob:` URLs, authenticated requests, or formats outside the intentionally narrow public URL observer. The project does not add code to defeat those mechanisms.

**Why must I enable host access manually?** Chrome requires host access in addition to `webRequest` for request observation. This repository keeps the permission optional so people can make an informed choice when they activate discovery.

**Which HLS and DASH media are supported?** HLS supports completed unencrypted media playlists and a supported muxed variant from a public Master Playlist. DASH supports one static, unencrypted `SegmentTemplate` period with video and optional audio tracks. Live content, encrypted streams, DRM, HLS maps/partials/byte ranges, DASH timelines/bases/lists, and authenticated delivery are rejected early and clearly.

**Can I share a failing URL in a public issue?** Only when you have the right to share it and it contains no personal, private, tokenized, credentialed, or protected data. When in doubt, write a minimal synthetic reproduction instead.

For expanded answers, see the repository [FAQ](../blob/main/docs/FAQ.md).

## How to contribute

Contributions are welcome when they make the project safer, easier to use, more accessible, better documented, or more reliable across operating systems. Good starting points include clearer diagnostics, synthetic tests for URL validation and cleanup, keyboard improvements in the popup, documentation translation, cross-platform packaging, and carefully specified **unprotected public** media scenarios.

Please fork the repository, create a focused branch, and run these checks before opening a pull request:

```bash
go test ./cli/...
go vet ./cli/...
node --check extension/background.js
node --check extension/popup.js
```

In a pull request, explain the user-visible change, test result, platform assumptions, and why the change stays within the authorization-first scope. Do not include downloaded media, cookies, tokens, credentials, headers, private links, or content you do not have permission to share.

Read [CONTRIBUTING.md](../blob/main/CONTRIBUTING.md), [SECURITY.md](../blob/main/SECURITY.md), and [CODE_OF_CONDUCT.md](../blob/main/CODE_OF_CONDUCT.md) before participating. Thank you for helping us keep the project small, practical, respectful, and reviewable.
