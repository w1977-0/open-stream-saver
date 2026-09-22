# Safe setup troubleshooting

This guide covers the failures that happen **before any media is handled** — the ones where nothing has been downloaded, no URL has been reviewed, and no rights question has come up. If you are here, the problem is almost always the local toolchain, not the media path.

Every entry gives a diagnostic command, the smallest result that counts as healthy, and a next step that stays inside the authorization-first scope. Command output in this guide is sanitized: no real URLs, paths with user names, extension IDs, or host paths appear.

If you have not installed anything yet, start with [INSTALLATION_VERIFICATION.md](INSTALLATION_VERIFICATION.md). If you changed something and want to record that setup still works, use the [contributor installation checklist](CONTRIBUTOR_INSTALL_CHECKLIST.md).

A reminder that applies to all five sections: when you ask for help, never post private URLs, account data, cookies, headers, tokens, DRM keys, media keys, or content you are not authorized to share. Paste the sanitized output of the diagnostic command instead.

## 1. Go is missing, or older than 1.25

**Symptom.** `go: command not found`, or a build that stops on a language feature the installed compiler does not know.

**Diagnostic.**

```bash
go version
```

**Expected result.** `go version go1.25` or later. The project sets `go 1.25.0` in the root `go.mod`, so anything older fails before it reaches your code.

**Next step.** Install a current Go toolchain from go.dev or your operating system's package manager, then reopen the shell and re-run the diagnostic. Confirm the active toolchain actually changed — a stale `go` earlier on `PATH` is the usual reason the version does not move.

## 2. The CLI builds but the shell cannot find it

**Symptom.** `open-stream-saver: command not found` after a successful `go install`, or after building into `bin/`.

**Diagnostic.**

```bash
go env GOPATH
```

On Linux and macOS, then check whether the binary is where you expect:

```bash
ls "$(go env GOPATH)/bin"
```

On Windows PowerShell:

```powershell
go env GOPATH
Get-ChildItem "$(go env GOPATH)\bin"
```

**Expected result.** `go env GOPATH` prints a directory, and that directory's `bin` subfolder contains `open-stream-saver` (or `open-stream-saver.exe` on Windows).

**Next step.** Add `$GOPATH/bin` to `PATH` — `$(go env GOPATH)/bin` on Linux and macOS, `%GOPATH%\bin` on Windows — then open a **new** shell and re-run `open-stream-saver --help`. If you built locally instead of installing, the binary is at `bin/open-stream-saver` inside your clone and is not on `PATH` at all; call it by path, or build from the clone root rather than from `cli/`.

## 3. FFmpeg is not found

**Symptom.** A remux step reports that FFmpeg is missing, or `ffmpeg -version` fails.

**Diagnostic.**

```bash
ffmpeg -version
```

**Expected result.** Either FFmpeg prints its version, or the shell reports that the command is not found. **Both are acceptable outcomes.** FFmpeg is optional: direct-file handling never needs it, and it is required only for local remuxing of an authorized, completed, unencrypted HLS or DASH playlist.

**Next step.** If your workflow does not remux, stop here — nothing is broken. If it does, install FFmpeg from your operating system's package manager or another distributor you already trust, then re-run the diagnostic. Releases intentionally do not bundle FFmpeg, and you should not fetch a prebuilt binary from an unfamiliar source just to clear this message.

## 4. Chrome refuses to load the unpacked extension

**Symptom.** Loading `extension/` reports a manifest error, or the extension appears with an error badge instead of a working popup.

**Diagnostic.** Validate the manifest and both scripts from the clone root:

```bash
python3 -m json.tool extension/manifest.json > /dev/null
node --check extension/background.js
node --check extension/popup.js
```

On Windows PowerShell, the same three commands work; `python` may be `python.exe` and `$null` replaces `/dev/null`.

**Expected result.** The manifest parses as JSON with no output, and both `node --check` calls exit silently. Any line of output here is the actual error.

**Next step.** Fix whichever file the diagnostic named, then reload the extension at `chrome://extensions`. Select the repository's `extension/` directory itself — not its parent, and not a packaged archive. If the popup still does not render, confirm you are on a Chromium browser that supports Manifest V3; this extension is not built for other engines.

## 5. The Native Messaging host is not reachable

**Symptom.** Choosing **Save locally** does nothing, or Chrome reports that the native host has exited or is not registered.

**Diagnostic.** Confirm the host binary exists and runs, then open the host manifest you installed and check two fields: `path` and `allowed_origins`.

```bash
./bin/open-stream-saver-host --help
```

**Expected result.** The host binary prints help text. In the installed manifest, `path` is an absolute path to **that** binary, and `allowed_origins` contains exactly one entry: `chrome-extension://` followed by **your own** extension ID as shown at `chrome://extensions`.

**Next step.** Fix the mismatch — a stale path from a previous build, or an extension ID copied from someone else's machine or from an example, are the two causes that account for nearly all of these. Start from `native-host/com.w1977_0.open_stream_saver.template.json`, put your own extension ID in the allowlist, and keep the host binary at a stable local path. Then follow the [Native Messaging host guide](../native-host/README.md) to re-register it.

Do not widen `allowed_origins` to accept arbitrary extensions, and do not put credentials, cookies, tokens, or media keys in the host configuration — the host is a local bridge for a fixed public-URL task schema, not a place to store secrets.

## If none of these match

Collect the sanitized output of the relevant diagnostic above, your operating system, and the Go version, then open an issue. State which step of [INSTALLATION_VERIFICATION.md](INSTALLATION_VERIFICATION.md) you reached and what you saw there. That is enough for someone to reproduce your setup without you sharing anything private.
