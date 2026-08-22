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
	"time"
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

func TestDownloadFallsBackToSequentialWithoutRangeSupport(t *testing.T) {
	payload := "local fixture without byte-range support"
	var sawRange bool
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Content-Length", strconv.Itoa(len(payload)))
		if request.Method == http.MethodHead {
			return
		}
		sawRange = request.Header.Get("Range") != ""
		_, _ = io.WriteString(writer, payload)
	}))
	defer server.Close()

	output := filepath.Join(t.TempDir(), "sequential.mp4")
	if err := Download(context.Background(), server.URL+"/fixture.mp4", Options{
		Client: server.Client(), Workers: 4, Output: output, Writer: io.Discard,
	}); err != nil {
		t.Fatalf("Download returned error: %v", err)
	}
	if sawRange {
		t.Fatal("sequential fallback unexpectedly sent a Range header")
	}
	result, err := os.ReadFile(output)
	if err != nil || string(result) != payload {
		t.Fatalf("sequential output = %q, %v", result, err)
	}
}

func TestDownloadRejectsPermanentHTTPStatusWithoutRetry(t *testing.T) {
	getRequests := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodGet {
			getRequests++
		}
		writer.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	directory := t.TempDir()
	output := filepath.Join(directory, "forbidden.mp4")
	err := Download(context.Background(), server.URL+"/fixture.mp4", Options{
		Client: server.Client(), Output: output, Writer: io.Discard,
	})
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("expected permanent HTTP 403 error, got %v", err)
	}
	if getRequests != 1 {
		t.Fatalf("GET requests = %d, want 1 for a non-retryable status", getRequests)
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("output should not exist after a rejected download: %v", statErr)
	}
}

func TestDownloadRefusesExistingOutputBeforeNetworkRequest(t *testing.T) {
	output := filepath.Join(t.TempDir(), "existing.mp4")
	if err := os.WriteFile(output, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	called := false
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		called = true
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	err := Download(context.Background(), server.URL+"/fixture.mp4", Options{Client: server.Client(), Output: output, Writer: io.Discard})
	if err == nil || !strings.Contains(err.Error(), "refusing to overwrite") {
		t.Fatalf("expected existing-output rejection, got %v", err)
	}
	if called {
		t.Fatal("Download made a network request after detecting an existing output")
	}
}

func TestDownloadCleansTemporaryFileAfterShortResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodHead {
			return
		}
		writer.Header().Set("Content-Length", "10")
		_, _ = io.WriteString(writer, "short")
	}))
	defer server.Close()

	directory := t.TempDir()
	output := filepath.Join(directory, "short.mp4")
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	err := Download(ctx, server.URL+"/fixture.mp4", Options{Client: server.Client(), Output: output, Writer: io.Discard})
	if err == nil {
		t.Fatal("expected short response to fail")
	}
	if _, statErr := os.Stat(output); !os.IsNotExist(statErr) {
		t.Fatalf("output should not exist after a short response: %v", statErr)
	}
	matches, globErr := filepath.Glob(filepath.Join(directory, ".open-stream-saver-*"))
	if globErr != nil || len(matches) != 0 {
		t.Fatalf("temporary files after failed download = %v, %v", matches, globErr)
	}
}
