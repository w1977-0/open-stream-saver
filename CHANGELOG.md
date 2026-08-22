# Changelog

This project prefers accurate, reviewable release notes over broad claims.

## 0.1.0 — Initial public architecture

The initial release contains a Manifest V3 Chrome extension with optional host access, read-only `.m3u8` / `.mp4` discovery, per-tab deduplication, session-only storage and copyable CLI commands.

It also includes a Go CLI for a single public direct file or completed, unencrypted HLS media playlist. Direct downloads can use HTTP byte ranges when supported. HLS downloads use a bounded segment worker pool and a locally installed FFmpeg for a local remux. The release intentionally rejects DRM, credentials, cookie handling, access-control bypasses, master / live / encrypted HLS playlists, byte-range segments, partial segments and bulk downloads.

The repository includes English and Chinese documentation, FAQ, contribution and security policies, tests, a cross-platform build workflow and tag-triggered release packaging.
