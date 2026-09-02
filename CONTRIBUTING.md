# Contributing to Media Archiver

Thank you for considering a contribution. This is a small project with a deliberately limited purpose: improve local handling of media that the user is **already authorized** to save. We value clear reasoning, respectful review, reproducible tests, and changes that make the project safer or easier to understand.

## Before you start

Please read the [README](README.md), [architecture](ARCHITECTURE.md), [FAQ](docs/FAQ.md), [security policy](SECURITY.md), and [code of conduct](CODE_OF_CONDUCT.md). By contributing code, documentation, translations, or tests, you agree that your work can be distributed under the Apache License 2.0.

Do not open an issue or pull request containing a private URL, browser profile data, user name, password, cookie, token, authorization header, DRM key, media key, or unlicensed content. If a security concern may expose sensitive information, follow [SECURITY.md](SECURITY.md) instead.

## Good contribution areas

| Area | Examples |
| --- | --- |
| Reliability | Better error handling, retry tests, safe cleanup, output-path handling, and clear diagnostics. |
| Accessibility | Keyboard-friendly popup behavior, readable contrast, localized documentation, or screen-reader improvements. |
| Cross-platform support | Reproducible fixes for Windows, macOS, and Linux that are covered by tests or CI. |
| Documentation | Installation corrections, user-facing explanations, architecture diagrams, or carefully translated guides. |
| Scope-aligned media handling | Support for a well-specified, non-encrypted public direct-file or completed media-playlist scenario with tests. |

Requests or contributions that add DRM decryption, credential / cookie import, login handling, paywall bypasses, regional-restriction bypasses, proxy rotation, bulk harvesting, playlist traversal, or request modification are not accepted.

## Development setup

The repository contains two independently runnable components.

```bash
# Clone the project
git clone https://github.com/w1977-0/open-stream-saver.git
cd open-stream-saver

# Test the Go CLI from the repository root module
go test ./cli/...
go vet ./cli/...
go build ./cli/cmd/open-stream-saver

# Check Chrome extension JavaScript syntax
node --check extension/background.js
node --check extension/popup.js
```

For a clean first setup, follow the [installation verification checklist](docs/INSTALLATION_VERIFICATION.md) before testing any media path. It records expected results for Go, FFmpeg, the CLI, and the unpacked extension without requiring a private or protected media sample.

FFmpeg is only needed when you manually test an authorized, completed, unencrypted HLS media playlist. Do not use private, protected, or third-party material as a test fixture.

## Your first contribution

Start with a documentation task when you are new to the project. Choose one small, user-visible correction, such as verifying an installation step or adding a troubleshooting entry, then open an Issue before making a broad change. Fork the repository, make one focused commit, and run the root-module checks shown above before opening a pull request.

A strong first contribution explains what a new user would see, which operating system or toolchain was checked, and why the wording stays within the authorization-first scope. Do not use private URLs, downloaded media, cookies, credentials, tokens, DRM material, or protected content as a reproduction. If you are unsure whether a proposed capability is in scope, open a Feature request first rather than implementing it.

## Making a change

Create a focused branch from `main`. Keep the change small enough to review, explain the user-visible effect, and update the documentation when behavior changes. Use descriptive commit messages such as `fix: reject redirects before download` or `docs: clarify optional extension permissions`.

For Go changes, run `gofmt -w`, `go test ./cli/...`, and `go vet ./cli/...` from the repository root. Add tests for behavior changes, especially validation, error paths, temporary-file cleanup, concurrency limits, and unsupported-stream rejection. For extension changes, keep Manifest V3 permissions minimal and explain any new permission in the pull request.

## Pull request checklist

Before opening a pull request, confirm the following statements in your description:

- I have read the project scope and did not add access-control or DRM bypass behavior.
- I did not commit credentials, personal data, private URLs, media keys, or downloaded media.
- I ran the relevant tests and included the results.
- I updated documentation or translations where users would otherwise be surprised.
- I kept the change focused and explained any platform-specific trade-offs.

Maintainers may ask for a smaller change, additional tests, or an alternative that better protects users and content owners. Review is collaborative rather than adversarial; thank you for being patient.
