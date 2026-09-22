# Contributor installation checklist

This checklist is for someone who has **just changed something** — documentation, setup wording, module layout, or build instructions — and wants to leave a record that the toolchain still comes up clean.

It is not a first-install guide. If you are installing the tool to use it, follow [INSTALLATION_VERIFICATION.md](INSTALLATION_VERIFICATION.md) instead. The two overlap on purpose only where a broken setup would otherwise go unnoticed.

Everything below runs against your own clone, needs no network media, and touches no media path. You do not need a URL, an account, a cookie, a token, or a sample file to complete any step.

## When to run it

Run the whole checklist after any of these:

- a change to the root `go.mod`, to the `cli/` package layout, or to any documented build command
- a documentation change that touches installation, setup, or troubleshooting wording
- your first contribution to this repository

## 1. Confirm the module boundary

The Go module is the **repository root**. The CLI source stays under `cli/`, and that is deliberate — the two are not the same thing, and confusing them is what produced the broken version-pinned install in v0.3.0.

```bash
go version
go env GOMOD
```

Expected result: `go version` reports `go1.25` or later, and `go env GOMOD` prints the absolute path to the `go.mod` in your clone root — not to a `cli/go.mod`, which does not exist.

On Windows the same two commands work in PowerShell. Paths print with backslashes; that is the only difference.

## 2. Run the root-module checks

From the clone root, not from `cli/`:

```bash
go test ./cli/...
go vet ./cli/...
go build -o bin/open-stream-saver ./cli/cmd/open-stream-saver
go build -o bin/open-stream-saver-host ./cli/cmd/open-stream-saver-host
```

Expected result: the tests pass, `go vet` prints no diagnostics, and both binaries appear under `bin/`. On Windows the binaries are `bin/open-stream-saver.exe` and `bin/open-stream-saver-host.exe`.

If you are installing by package path rather than building locally, pin a **root** release tag:

```bash
go install github.com/w1977-0/open-stream-saver/cli/cmd/open-stream-saver@v0.3.1
```

Do not write the version as `cli/vX.Y.Z`. Published versions are root repository tags; a `cli/`-prefixed version does not resolve.

## 3. Confirm the CLI answers

```bash
./bin/open-stream-saver --help
```

Expected result: help text listing `download`, `inspect-hls`, and `completion`, plus the global flags. A CLI that builds but prints nothing here is a broken CLI.

This is the minimum success signal for a CLI change. It does not require and must not use a media URL.

## 4. Optional: verify FFmpeg locally

FFmpeg is **optional**. It is needed only for local remuxing of an authorized, completed, unencrypted HLS or DASH playlist. Direct-file handling never needs it, and releases never bundle it.

```bash
ffmpeg -version
```

Expected result when installed: FFmpeg prints its version. Expected result when absent: a "command not found" style error — which is fine, and is not a failure of this checklist. Record which of the two you saw.

Do not install FFmpeg from an untrusted source just to make this step pass, and do not download media to exercise it.

## 5. Load the unpacked extension

1. Open `chrome://extensions` and enable **Developer mode**.
2. Choose **Load unpacked** and select the repository's `extension/` directory.
3. Open the popup.

Expected result: Chrome loads the extension with no manifest error, and the popup renders showing that discovery is off. Loading is the whole check — you do not enable discovery, grant site access, or supply a URL.

## 6. Record what you saw

When you open the issue or pull request, report only:

- your operating system
- the Go version from step 1
- whether the FFmpeg check found it, when your change touches remuxing
- whether the tests, vet, and both builds passed
- whether the unpacked popup loaded

Never include private URLs, account data, cookies, headers, tokens, DRM or media keys, browser profile data, or any media you are not authorized to share. A sanitized paste of the command output above is enough; if you are unsure whether a line is safe to post, leave it out and describe it instead.

## Troubleshooting

If any step above fails before you reach a media path, [SETUP_TROUBLESHOOTING.md](SETUP_TROUBLESHOOTING.md) covers the five failures that account for most of them.
