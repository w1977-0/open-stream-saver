# Contribution security and scope policy

The canonical contribution guide is [../CONTRIBUTING.md](../CONTRIBUTING.md). This repository-level policy makes the non-negotiable project boundary visible to contributors using GitHub's standard community-health locations.

> Open Stream Saver is for user-authorized, public, unencrypted media only. It must not collect, import, intercept, retain, or forward cookies, credentials, authorization headers, tokens, browser storage, media keys, or DRM material.

Pull requests and issues are out of scope when they add, request, or depend on any of the following:

- page-injected `fetch` / `XMLHttpRequest` hooks intended to recover authorization material;
- cookie, header, token, credential, browser-profile, or login import/export;
- AES-128, SAMPLE-AES, key-retrieval, decryption, Widevine, FairPlay, PlayReady, or any DRM bypass work;
- login, paywall, subscription, region, proxy, anti-bot, or anti-leech bypasses;
- bulk harvesting, background monitoring, request rewriting, or traffic blocking.

Contributions are welcome when they improve safe public-media handling, reliability, tests, documentation, accessibility, local-only privacy, or cross-platform behavior. Use synthetic or clearly authorized test material. Never post private URLs, account information, downloaded media, cookies, tokens, headers, or keys in public issues or pull requests.
