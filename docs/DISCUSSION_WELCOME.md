# Start here: FAQ, project scope, and contributing

Welcome to **Open Stream Saver**. This is a local-first learning project for users who are already authorized to save public media. The goal is not to become a universal downloader; it is to make one small, auditable workflow clear: observe an eligible public URL, review it, confirm your rights, and save it locally.

## What the project contains

The Chrome extension uses Manifest V3 and observes completed public `.m3u8` and `.mp4` requests after the user explicitly enables optional host access. It is read-only and retains only a bounded, session-scoped list for each tab. The Go CLI accepts a single public URL and requires `--acknowledge-rights`. It supports ordinary direct files and completed, unencrypted HLS media playlists; FFmpeg is invoked locally only after all supported segments have been saved.

## Frequently asked questions

**Does the project bypass DRM, login, subscription, paywall, or regional protections?** No. It does not import cookies or credentials, inspect authorization headers, read media keys, modify requests, inject page scripts, or implement proxies. Those requests are outside the project scope.

**Why can’t the extension see every video on a page?** Many pages use protected pipelines, `blob:` URLs, authenticated requests, or formats outside the intentionally narrow `.m3u8` / `.mp4` observer. The project does not add code to defeat those mechanisms.

**Why must I enable host access manually?** Chrome requires host access in addition to `webRequest` for request observation. This repository keeps that access optional so people can make an informed choice when they activate discovery.

**Which HLS media are supported?** Only completed, unencrypted media playlists. Master playlists, live playlists, `EXT-X-KEY`, `EXT-X-MAP`, partial segments, byte-range segments, unavailable segments, and non-HTTP segment URLs are rejected early and clearly.

**Can I share a failing URL in a public issue?** Only when you have the right to share it and it contains no personal, private, tokenized, credentialed, or protected data. When in doubt, write a minimal synthetic reproduction instead.

For expanded answers, see the repository [FAQ](../blob/main/docs/FAQ.md).

## How to contribute

Contributions are welcome when they make the project safer, easier to use, more accessible, better documented, or more reliable across operating systems. Good starting points include clarifying errors, adding tests for URL validation and cleanup, improving keyboard behavior in the popup, translating documentation, and extending support for clearly documented **unprotected public** cases.

Please fork the repository, create a focused branch, and run these checks before opening a pull request:

```bash
cd cli
go test ./...
go vet ./...
node --check ../extension/background.js
node --check ../extension/popup.js
```

In a pull request, explain the user-visible change, test result, platform assumptions, and why the change stays within the authorization-first scope. Do not include downloaded media, cookies, tokens, credentials, private links, or content that you do not have permission to share.

Read [CONTRIBUTING.md](../blob/main/CONTRIBUTING.md), [SECURITY.md](../blob/main/SECURITY.md), and [CODE_OF_CONDUCT.md](../blob/main/CODE_OF_CONDUCT.md) before participating. Thank you for helping us keep the project small, practical, respectful, and reviewable.
