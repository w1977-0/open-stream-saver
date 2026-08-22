package ffmpeg

import (
	"context"
	"fmt"
	"os/exec"
)

// MergeTS invokes a locally installed FFmpeg to remux already-downloaded local
// transport-stream segments. It never supplies remote URLs, credentials, keys,
// cookies, or user-provided shell fragments to FFmpeg.
func MergeTS(ctx context.Context, concatFile, output string) error {
	path, err := exec.LookPath("ffmpeg")
	if err != nil {
		return fmt.Errorf("ffmpeg is required for HLS merge: install FFmpeg and ensure it is on PATH")
	}
	command := exec.CommandContext(ctx, path,
		"-hide_banner", "-loglevel", "error", "-y",
		"-f", "concat", "-safe", "0", "-i", concatFile,
		"-c", "copy", output,
	)
	if outputBytes, err := command.CombinedOutput(); err != nil {
		return fmt.Errorf("ffmpeg merge failed: %w (%s)", err, string(outputBytes))
	}
	return nil
}
