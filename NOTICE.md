# Notice

Copyright 2026 w1977-0

Open Stream Saver is distributed under the Apache License 2.0. This repository uses third-party dependencies with their own licenses and notices:

| Dependency | Purpose | License / source |
| --- | --- | --- |
| Cobra | Command-line interface | Apache-2.0 — https://github.com/spf13/cobra |
| mpb | Terminal progress display | MIT — https://github.com/vbauerster/mpb |
| hls-m3u8 | HLS playlist parsing | BSD-3-Clause — https://github.com/Eyevinn/hls-m3u8 |
| golang.org/x/sync | Goroutine coordination | BSD-3-Clause — https://pkg.go.dev/golang.org/x/sync |

The project invokes a user-installed FFmpeg binary for local remuxing of supported unencrypted HLS segments. FFmpeg is not bundled with this repository; its distribution and licensing are separate from this project.
