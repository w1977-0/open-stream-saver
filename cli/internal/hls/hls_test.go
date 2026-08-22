package hls

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

func playlistURL(t *testing.T) *url.URL {
	t.Helper()
	parsed, err := url.Parse("https://example.com/media/playlist.m3u8")
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func fixtureClient(responses map[string]string, status int) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		body, ok := responses[request.URL.String()]
		if !ok {
			body = "missing fixture"
		}
		return &http.Response{
			StatusCode: status,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})}
}

func TestParseSegmentsRejectsMasterPlaylist(t *testing.T) {
	playlist := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1000\nvideo.m3u8\n"
	if _, err := parseSegments(playlistURL(t), []byte(playlist)); err == nil || !strings.Contains(err.Error(), "master") {
		t.Fatalf("expected master playlist rejection, got %v", err)
	}
}

func TestParseSegmentsRejectsEncryptedPlaylist(t *testing.T) {
	playlist := "#EXTM3U\n#EXT-X-TARGETDURATION:10\n#EXT-X-KEY:METHOD=AES-128,URI=\"key.bin\"\n#EXTINF:10,\nsegment.ts\n#EXT-X-ENDLIST\n"
	if _, err := parseSegments(playlistURL(t), []byte(playlist)); err == nil {
		t.Fatal("expected encrypted playlist rejection")
	}
}

func TestInspectListsOnlyMuxedPublicVariants(t *testing.T) {
	masterURL := "https://example.com/media/master.m3u8"
	master := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1000,RESOLUTION=640x360\nlow.m3u8\n#EXT-X-STREAM-INF:BANDWIDTH=4000,RESOLUTION=1920x1080\nhigh.m3u8\n"
	variants, err := Inspect(context.Background(), masterURL, fixtureClient(map[string]string{masterURL: master}, http.StatusOK))
	if err != nil {
		t.Fatalf("Inspect returned error: %v", err)
	}
	if len(variants) != 2 {
		t.Fatalf("variants = %#v", variants)
	}
	if variants[0].URI != "https://example.com/media/low.m3u8" || variants[1].Bandwidth != 4000 {
		t.Fatalf("unexpected variants: %#v", variants)
	}
	selected, err := chooseVariant(variants, -1)
	if err != nil || selected.Index != 1 {
		t.Fatalf("highest-bandwidth variant = %#v, %v", selected, err)
	}
	if _, err := chooseVariant(variants, 9); err == nil {
		t.Fatal("expected an unavailable variant to be rejected")
	}
}

func TestResolveMediaPlaylistFetchesRequestedVariant(t *testing.T) {
	masterURL := "https://example.com/media/master.m3u8"
	highURL := "https://example.com/media/high.m3u8"
	master := "#EXTM3U\n#EXT-X-STREAM-INF:BANDWIDTH=1000\nlow.m3u8\n#EXT-X-STREAM-INF:BANDWIDTH=4000\nhigh.m3u8\n"
	media := "#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\nfirst.ts\n#EXT-X-ENDLIST\n"
	client := fixtureClient(map[string]string{highURL: media}, http.StatusOK)
	resolved, data, err := resolveMediaPlaylist(context.Background(), mustURL(t, masterURL), []byte(master), client, 1)
	if err != nil {
		t.Fatalf("resolveMediaPlaylist returned error: %v", err)
	}
	if resolved.String() != highURL || string(data) != media {
		t.Fatalf("resolved playlist = %s, %q", resolved, data)
	}
}

func TestParseSegmentsResolvesCompletedMediaPlaylist(t *testing.T) {
	playlist := "#EXTM3U\n#EXT-X-TARGETDURATION:1\n#EXTINF:1,\nfirst.ts\n#EXTINF:1,\nsecond.ts\n#EXT-X-ENDLIST\n"
	segments, err := parseSegments(playlistURL(t), []byte(playlist))
	if err != nil {
		t.Fatalf("parseSegments returned error: %v", err)
	}
	if len(segments) != 2 || segments[0].url != "https://example.com/media/first.ts" || segments[1].index != 1 {
		t.Fatalf("segments = %#v", segments)
	}
}

func TestDownloadSegmentRejectsForbiddenAndLeavesNoFile(t *testing.T) {
	directory := t.TempDir()
	client := fixtureClient(nil, http.StatusForbidden)
	_, err := downloadSegment(context.Background(), client, directory, segment{index: 3, url: "https://example.com/forbidden.ts"})
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("expected forbidden segment error, got %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(directory, "00000003.ts")); !os.IsNotExist(statErr) {
		t.Fatalf("failed segment file should be removed: %v", statErr)
	}
}

func TestWriteConcatFileAndOutputGuards(t *testing.T) {
	directory := t.TempDir()
	concat, err := writeConcatFile(directory, []string{filepath.Join(directory, "00000001.ts"), filepath.Join(directory, "00000000.ts")})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(concat)
	if err != nil || !strings.Contains(string(content), "00000000.ts") {
		t.Fatalf("concat content = %q, %v", content, err)
	}
	if _, err := writeConcatFile(directory, []string{"quoted'path.ts"}); err == nil {
		t.Fatal("expected quote in concat path to be rejected")
	}
	output := filepath.Join(directory, "existing.mp4")
	if err := os.WriteFile(output, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureNewOutput(output); err == nil {
		t.Fatal("expected existing output to be rejected")
	}
	if workerCount(0) != 1 || workerCount(maxWorkers+1) != maxWorkers {
		t.Fatal("worker count bounds were not enforced")
	}
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}
