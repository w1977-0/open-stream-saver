# Installation verification checklist

This checklist verifies a **local, authorization-first** installation of Media Archiver. It does not require a private URL, account, cookie, token, protected media sample, or browser-site permission. Passing these steps confirms that the CLI, optional local dependencies, and unpacked extension can start; it does **not** grant permission to save any media.

## 1. Confirm prerequisites

Install Go **1.25 or newer** and check the active toolchain:

```bash
go version
```

Expected result: the command prints a Go version of `go1.25` or later.

For HLS or DASH remuxing, also install FFmpeg through your operating system or another trusted distributor:

```bash
ffmpeg -version
```

Expected result: FFmpeg prints its version. FFmpeg is not needed to build the CLI or to validate a direct-file workflow, and it is not bundled in Media Archiver releases.

## 2. Install the CLI from the root module

Media Archiver releases the repository root as the Go module. The CLI source remains under `cli/`, so install its package path with a root release tag:

```bash
go install github.com/w1977-0/media-archiver/cli/cmd/open-stream-saver@v0.3.1
open-stream-saver --help
```

Do not use a `cli/vX.Y.Z` suffix as the version in `go install`; published versions are root repository tags.

Expected result: `open-stream-saver --help` lists commands including `download`, `inspect-hls`, and `completion`. If your shell cannot find the binary, add the Go binary directory reported by `go env GOPATH` to your `PATH`.

## 3. Verify a source build

Clone the canonical repository and run the same checks used for local development:

```bash
git clone https://github.com/w1977-0/media-archiver.git
cd media-archiver
go test ./cli/...
go vet ./cli/...
go build -o bin/open-stream-saver ./cli/cmd/open-stream-saver
go build -o bin/open-stream-saver-host ./cli/cmd/open-stream-saver-host
./bin/open-stream-saver --help
```

Expected result: the tests and vet complete without diagnostics, both binaries build, and the CLI prints help text. This only validates local setup; do not test with private, protected, or unauthorized content.

## 4. Verify the unpacked Chrome extension

Until an appropriate store review is complete, use Chrome's local developer workflow:

1. Open `chrome://extensions` and enable **Developer mode**.
2. Select **Load unpacked** and choose this repository's `extension/` directory.
3. Open the extension popup. Before you enable discovery, it should show that discovery is off and explain that it is read-only.
4. Do not grant site access or provide a media URL merely to test installation. The popup itself is the minimum success criterion for this checklist.

Expected result: Chrome loads the extension without a manifest error, and the popup renders its local, explicit-authorization interface. Enabling discovery is optional and should only be done on a page where you have permission to inspect the relevant public request.

## 5. Optional Native Messaging host

The local host is optional. Only configure it after the CLI and popup checks above pass. Follow the [Native Messaging host guide](../native-host/README.md), add **your own exact extension ID** to the host manifest allowlist, and use a stable local path for `open-stream-saver-host`.

Expected result: the host manifest names only your extension ID and the local host binary. Do not edit it to accept arbitrary origins, and do not store credentials, cookies, tokens, or media keys in the host configuration.

## 6. Record a safe verification result

When opening an Issue or Pull Request about installation, report only your operating system, Go version, FFmpeg version when applicable, sanitized CLI output, and whether the unpacked popup loaded. Never include private URLs, account data, cookies, headers, tokens, DRM keys, or media you are not authorized to share.
