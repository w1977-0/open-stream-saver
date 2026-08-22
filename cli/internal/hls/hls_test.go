package hls

import (
	"net/url"
	"strings"
	"testing"
)

func playlistURL(t *testing.T) *url.URL {
	t.Helper()
	parsed, err := url.Parse("https://example.com/media/playlist.m3u8")
	if err != nil {
		t.Fatal(err)
	}
	return parsed
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
