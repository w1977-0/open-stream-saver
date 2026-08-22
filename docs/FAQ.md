# Frequently asked questions

## Is this a universal video downloader?

No. Open Stream Saver is deliberately narrow. It helps a user identify public `.mp4` or `.m3u8` URLs observed by the browser and save a single URL only after the user has confirmed that they have the right to do so. It is not designed to support every site or every streaming configuration.

## Can it download DRM-protected, paid, login-only, or region-restricted media?

No. The extension does not inspect cookies, authorization headers, page scripts, media keys, or account details. The CLI does not accept credentials, cookies, decryption keys, or proxy settings. Requests aimed at bypassing technical protection measures or access controls are out of scope.

## Why does the extension ask for site access?

Chrome requires both the `webRequest` API permission and matching host permissions before an extension can monitor network requests. This project makes broad HTTP/HTTPS host access optional: discovery only starts after the user actively enables it in the popup. [1] [2]

## What information does the extension retain?

For the current browser session, it stores only eligible URL strings, their detected extension, a timestamp, and a non-sensitive initiator value. It limits storage to 40 entries per tab and clears those entries when the tab closes. It does not send this data to a backend.

## Why are some visible videos not shown?

A visible video can be loaded through a different format, a protected media pipeline, a `blob:` URL, a page-managed API, or an authenticated request. This project intentionally observes only completed public HTTP(S) requests whose URL path ends in `.m3u8` or `.mp4`; it does not add code intended to defeat other delivery mechanisms.

## Why did the CLI reject a URL?

The CLI rejects non-HTTP(S) URLs, embedded credentials, localhost / private-network targets, redirected requests, and unresolvable hosts. These guardrails reduce the risk of accidentally making requests to local services or exposing account information.

## What HLS playlists are supported?

The CLI supports completed, unencrypted **media** playlists. It rejects master playlists, live / sliding playlists, `EXT-X-KEY`, `EXT-X-MAP`, low-latency partial segments, byte-range segments and unavailable segments. FFmpeg must be installed locally for the final remux.

## Why does a direct-file download use several connections?

When a server advertises stable HTTP byte-range support and provides a content length, the CLI divides the file into bounded ranges and writes them to a local temporary file. If either condition is missing, it falls back to one sequential request. It never uses credentials or modifies browser traffic.

## Can I submit a feature request for cookies, logins, DRM, proxy rotation, bulk downloads, or bypassing limits?

No. Those requests conflict with the project’s authorization-first scope and will be closed without implementation. Contributions that improve clear error messages, local accessibility, tests, documentation, build reliability, or support for **unprotected public media** are welcome.

## How can I report a bug safely?

Open an issue using the bug report form. Include a minimal, non-sensitive reproduction, operating system, CLI version, sanitized error output, and the test result. Do **not** include private URLs, user names, account data, cookies, tokens, authorization headers, media keys, or material that you do not have permission to share.

## How are releases made?

Pushing a version tag such as `v0.1.0` starts the release workflow. It runs tests and cross-compiles Go binaries for Linux, macOS, and Windows before creating a GitHub Release with checksums. Release assets are source-controlled binaries; users should verify the checksums before executing them.

## References

[1]: https://developer.chrome.com/docs/extensions/reference/api/webRequest "Chrome for Developers — webRequest"
[2]: https://developer.chrome.com/docs/extensions/develop/concepts/declare-permissions "Chrome for Developers — Declare permissions"
