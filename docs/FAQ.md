# Frequently asked questions

## Is this a universal video downloader?

No. Open Stream Saver is deliberately narrow. It helps a user identify a public `.mp4`, `.m3u8`, or `.mpd` URL and save one item only after the user confirms they have the right to do so. It is not designed to support every website or streaming configuration.

## Can it download DRM-protected, paid, login-only, encrypted, or region-restricted media?

No. The extension does not inspect cookies, authorization headers, page scripts, media keys, account details, or browser storage. The CLI and native host do not accept credentials, cookies, custom headers, tokens, decryption keys, or proxy settings. Requests aimed at bypassing technical protection measures or access controls are out of scope.

## Why does the extension ask for site access and Native Messaging permission?

Chrome requires `webRequest` plus matching host permission for read-only observation of a matching request. This project asks for HTTP/HTTPS host access only after the user enables discovery in the popup. The separate `nativeMessaging` permission lets an extension page send one reviewed task to an optional local host; it does not grant access to browser cookies or page data. [1] [2] [3]

## What information does the extension retain or send locally?

For the current browser session, it stores only eligible URL strings, type, discovery time, and a non-sensitive initiator value. It keeps at most 40 records per tab and clears them when the tab closes. When a user presses **Save locally** after ticking the rights confirmation, it sends only a fixed task schema: action, public URL, worker count, optional HLS variant, and the acknowledgement. It never sends a cookie, header, token, credential, page body, key, or downloaded media to a backend.

## Why are some visible videos not shown?

A visible video can use a protected media pipeline, a `blob:` URL, a non-supported format, a page-managed API, or an authenticated request. This project intentionally observes only completed public HTTP(S) requests whose URL path ends in `.m3u8`, `.mpd`, or `.mp4`; it does not add code intended to defeat other delivery mechanisms.

## Why did the CLI reject a URL?

The CLI rejects non-HTTP(S) URLs, embedded credentials, localhost/private-network targets, redirects, and unresolvable hosts. It also refuses to overwrite an existing output. These guardrails reduce the risk of accidental local-service requests, credential exposure, or destructive file handling.

## What HLS playlists are supported?

The CLI supports a completed, unencrypted media playlist and a public Master Playlist containing a supported **muxed** media variant. Use `inspect-hls` to list available indices and `--variant INDEX` to choose one; `--variant -1` selects the highest advertised bandwidth by default. It rejects live/sliding playlists, `EXT-X-KEY`, AES-128/SAMPLE-AES, `EXT-X-MAP`, low-latency partial segments, byte ranges, gaps, separate rendition groups, content steering, and non-public segment URLs. FFmpeg must be installed locally for remuxing.

## What DASH presentations are supported?

The supported profile is intentionally small: one public static MPD period using direct `SegmentTemplate` addressing, a video representation, and an optional audio representation. The engine downloads the public tracks with a bounded worker pool and asks FFmpeg to remux **local temporary files only**. It rejects `ContentProtection`/DRM, dynamic MPDs, `SegmentTimeline`, `SegmentBase`, `SegmentList`, and any authenticated or redirected request.

## Why does a direct-file download use several connections?

When a server advertises stable HTTP byte-range support and a content length, the CLI divides the file into bounded ranges and validates each returned content range before writing it to a temporary file. Otherwise it uses one sequential request. Network-only failures receive bounded exponential backoff; permanent protocol or permission errors do not loop indefinitely.

## Does the output include a hash?

Yes. After a successful new output is written, the CLI prints a local SHA-256 digest. This is a local integrity report for the resulting file; it does not establish ownership or authorize a source.

## Can I submit a feature request for cookies, logins, injected hooks, decryption, DRM, proxy rotation, bulk downloads, or bypassing limits?

No. Those requests conflict with the project’s authorization-first scope and will be closed without implementation. Contributions that improve clear error messages, local accessibility, tests, documentation, build reliability, or support for **unprotected public media** are welcome.

## How can I report a bug safely?

Open an issue using the bug report form. Include a minimal, non-sensitive reproduction, operating system, CLI version, sanitized error output, and test result. Do **not** include private URLs, usernames, account data, cookies, tokens, authorization headers, media keys, or material you are not allowed to share.

## How are releases made?

Pushing a `v*` tag starts three-platform tests and GoReleaser publication. The release contains the CLI, optional native host, unpacked extension source, host-manifest template, documentation, and `SHA256SUMS.txt`. It produces Linux, macOS, and Windows packages for AMD64 and ARM64. FFmpeg is not bundled; users should install it from their operating system or a trusted upstream source.

## References

[1]: https://developer.chrome.com/docs/extensions/reference/api/webRequest "Chrome for Developers — webRequest"
[2]: https://developer.chrome.com/docs/extensions/develop/concepts/declare-permissions "Chrome for Developers — Declare permissions"
[3]: https://developer.chrome.com/docs/extensions/develop/concepts/native-messaging "Chrome for Developers — Native messaging"
