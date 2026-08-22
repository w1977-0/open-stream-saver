package integrity

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSHA256FileAndReport(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sample.bin")
	if err := os.WriteFile(path, []byte("open-stream-saver"), 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := SHA256File(path)
	if err != nil {
		t.Fatalf("SHA256File returned error: %v", err)
	}
	const want = "1ad0748bc281655cc20c8f64b8126a68bd9ad20c0297f9e3dfaaf7ccbb150314"
	if digest != want {
		t.Fatalf("digest = %s, want %s", digest, want)
	}
	var output bytes.Buffer
	if err := Report(&output, path); err != nil {
		t.Fatalf("Report returned error: %v", err)
	}
	if !strings.Contains(output.String(), "SHA256 "+want+"  "+path) {
		t.Fatalf("unexpected report: %q", output.String())
	}
}
