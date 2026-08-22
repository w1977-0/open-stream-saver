package ffmpeg

import (
	"context"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestMergeTSExplainsMissingFFmpeg(t *testing.T) {
	restore := replaceLookPath(t, func(string) (string, error) { return "", exec.ErrNotFound })
	defer restore()

	err := MergeTS(context.Background(), "segments.txt", "output.mp4")
	if err == nil || !strings.Contains(err.Error(), "ffmpeg is required") || !strings.Contains(err.Error(), "PATH") {
		t.Fatalf("expected a helpful missing-ffmpeg error, got %v", err)
	}
}

func TestMergeDASHWrapsFFmpegFailure(t *testing.T) {
	restorePath := replaceLookPath(t, func(string) (string, error) { return "fixture-ffmpeg", nil })
	defer restorePath()
	restoreCommand := replaceCommandContext(t, helperCommand)
	defer restoreCommand()

	err := MergeDASH(context.Background(), "video.txt", "audio.txt", "output.mp4")
	if err == nil || !strings.Contains(err.Error(), "ffmpeg DASH merge failed") || !strings.Contains(err.Error(), "fixture ffmpeg failure") {
		t.Fatalf("expected wrapped DASH merge error, got %v", err)
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_FFMPEG_HELPER") != "1" {
		return
	}
	_, _ = os.Stderr.WriteString("fixture ffmpeg failure")
	os.Exit(17)
}

func helperCommand(ctx context.Context, _ string, _ ...string) *exec.Cmd {
	command := exec.CommandContext(ctx, os.Args[0], "-test.run=TestHelperProcess", "--")
	command.Env = append(os.Environ(), "GO_WANT_FFMPEG_HELPER=1")
	return command
}

func replaceLookPath(t *testing.T, replacement func(string) (string, error)) func() {
	t.Helper()
	original := lookupExecutable
	lookupExecutable = replacement
	return func() { lookupExecutable = original }
}

func replaceCommandContext(t *testing.T, replacement func(context.Context, string, ...string) *exec.Cmd) func() {
	t.Helper()
	original := commandContext
	commandContext = replacement
	return func() { commandContext = original }
}
