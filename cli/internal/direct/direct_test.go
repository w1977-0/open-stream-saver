package direct

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestDownloadUsesRangeCompatibleServer(t *testing.T) {
	payload := strings.Repeat("open-stream-saver-", 80_000)
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Accept-Ranges", "bytes")
		writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		if request.Method == http.MethodHead {
			return
		}
		if raw := request.Header.Get("Range"); raw != "" {
			parts := strings.TrimPrefix(raw, "bytes=")
			bounds := strings.Split(parts, "-")
			start, _ := strconv.Atoi(bounds[0])
			end, _ := strconv.Atoi(bounds[1])
			writer.Header().Set("Content-Range", "bytes "+strconv.Itoa(start)+"-"+strconv.Itoa(end)+"/"+strconv.Itoa(len(payload)))
			writer.WriteHeader(http.StatusPartialContent)
			_, _ = io.WriteString(writer, payload[start:end+1])
			return
		}
		_, _ = io.WriteString(writer, payload)
	}))
	defer server.Close()

	directory := t.TempDir()
	output := filepath.Join(directory, "saved.mp4")
	if err := Download(context.Background(), server.URL+"/video.mp4", Options{
		Client: server.Client(), Workers: 3, Output: output, Writer: io.Discard,
	}); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	result, err := os.ReadFile(output)
	if err != nil {
		t.Fatalf("could not read output: %v", err)
	}
	if string(result) != payload {
		t.Fatal("downloaded data differs from original payload")
	}
}
