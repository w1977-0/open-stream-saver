package dash

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

func manifestURL(t *testing.T) *url.URL {
	t.Helper()
	value, err := url.Parse("https://example.com/media/manifest.mpd")
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestParseManifestRejectsContentProtection(t *testing.T) {
	_, err := parseManifest([]byte(`<MPD type="static" mediaPresentationDuration="PT4S"><ContentProtection schemeIdUri="urn:uuid:test"/><Period/></MPD>`))
	if err == nil || !strings.Contains(err.Error(), "DRM-protected") {
		t.Fatalf("expected DRM rejection, got %v", err)
	}
}

func TestParseManifestRejectsDynamicAndMultiPeriodPresentations(t *testing.T) {
	for name, data := range map[string]string{
		"dynamic":      `<MPD type="dynamic"><Period/></MPD>`,
		"multi-period": `<MPD type="static"><Period/><Period/></MPD>`,
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := parseManifest([]byte(data)); err == nil {
				t.Fatal("expected unsupported presentation to be rejected")
			}
		})
	}
}

func TestBuildPlanCreatesPublicVideoAndAudioTracks(t *testing.T) {
	data := []byte(`<MPD type="static" mediaPresentationDuration="PT4S">
  <Period>
    <AdaptationSet contentType="video">
      <SegmentTemplate timescale="1" duration="2" startNumber="1" media="video-$Number$.m4s" initialization="video-init.mp4"/>
      <Representation id="v-low" bandwidth="100"><BaseURL>low/</BaseURL></Representation>
      <Representation id="v-high" bandwidth="200"><BaseURL>high/</BaseURL></Representation>
    </AdaptationSet>
    <AdaptationSet contentType="audio">
      <SegmentTemplate timescale="1" duration="2" startNumber="1" media="audio-$Number$.m4s" initialization="audio-init.mp4"/>
      <Representation id="a-main" bandwidth="64"/>
    </AdaptationSet>
  </Period>
</MPD>`)
	document, err := parseManifest(data)
	if err != nil {
		t.Fatal(err)
	}
	job, err := buildPlan(manifestURL(t), document)
	if err != nil {
		t.Fatalf("buildPlan returned error: %v", err)
	}
	if job.video == nil || job.audio == nil {
		t.Fatalf("expected video and audio plans: %#v", job)
	}
	if job.video.initURL != "https://example.com/media/high/video-init.mp4" {
		t.Fatalf("video init = %s", job.video.initURL)
	}
	if got, want := strings.Join(job.video.segmentURLs, ","), "https://example.com/media/high/video-1.m4s,https://example.com/media/high/video-2.m4s"; got != want {
		t.Fatalf("video URLs = %s, want %s", got, want)
	}
	if job.audio.initURL != "https://example.com/media/audio-init.mp4" {
		t.Fatalf("audio init = %s", job.audio.initURL)
	}
}

func TestBuildPlanRejectsUnsupportedAddressing(t *testing.T) {
	for name, template := range map[string]string{
		"segment-timeline": `<SegmentTemplate timescale="1" duration="2" media="video-$Number$.m4s" initialization="init.mp4"><SegmentTimeline><S d="2"/></SegmentTimeline></SegmentTemplate>`,
		"format-token":     `<SegmentTemplate timescale="1" duration="2" media="video-$Number%05d$.m4s" initialization="init.mp4"/>`,
	} {
		t.Run(name, func(t *testing.T) {
			data := []byte(`<MPD type="static" mediaPresentationDuration="PT4S"><Period><AdaptationSet contentType="video">` + template + `<Representation id="v1" bandwidth="1"/></AdaptationSet></Period></MPD>`)
			document, err := parseManifest(data)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := buildPlan(manifestURL(t), document); err == nil {
				t.Fatal("expected unsupported DASH addressing to be rejected")
			}
		})
	}
}

func TestResolveBaseAndTemplateRejectUnsafeValues(t *testing.T) {
	base := manifestURL(t)
	if _, err := resolveBase(base, []string{"", "second/"}); err == nil {
		t.Fatal("expected multiple or empty BaseURL values to be rejected")
	}
	if _, err := resolveBase(base, []string{"http://127.0.0.1/private/"}); err == nil {
		t.Fatal("expected private BaseURL to be rejected")
	}
	if _, err := resolveTemplateURL(base, "video-$Unknown$.m4s", "v1", 1, 0); err == nil {
		t.Fatal("expected unresolved template token to be rejected")
	}
	if _, err := resolveTemplateURL(base, "http://127.0.0.1/private.m4s", "v1", 1, 0); err == nil {
		t.Fatal("expected private segment URL to be rejected")
	}
}

func TestDownloadItemRejectsForbiddenAndCleansPath(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "00000001.m4s")
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusForbidden, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("forbidden")), Request: request}, nil
	})}
	err := downloadItemFile(context.Background(), client, downloadItem{url: "https://example.com/forbidden.m4s", path: path})
	if err == nil || !strings.Contains(err.Error(), "HTTP 403") {
		t.Fatalf("expected forbidden item error, got %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("failed item should not remain on disk: %v", statErr)
	}
}

func TestWriteConcatFileAndOutputGuards(t *testing.T) {
	directory := t.TempDir()
	concat, err := writeConcatFile(directory, []string{filepath.Join(directory, "00000001.m4s"), filepath.Join(directory, "00000000.init")})
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(concat)
	if err != nil || !strings.Contains(string(content), "00000000.init") {
		t.Fatalf("concat content = %q, %v", content, err)
	}
	if _, err := writeConcatFile(directory, []string{"bad'path"}); err == nil {
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

func TestPresentationDuration(t *testing.T) {
	seconds, err := presentationDuration("PT1H2M3.5S", "")
	if err != nil || seconds != 3723.5 {
		t.Fatalf("duration = %v, %v", seconds, err)
	}
	if _, err := presentationDuration("not-a-duration", ""); err == nil {
		t.Fatal("expected invalid duration to be rejected")
	}
}

func TestFetchManifestReadsStaticFixture(t *testing.T) {
	fixture := `<MPD type="static" mediaPresentationDuration="PT1S"><Period/></MPD>`
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader(fixture)), Request: request}, nil
	})}
	data, err := fetchManifest(context.Background(), client, "https://example.com/fixture.mpd")
	if err != nil || string(data) != fixture {
		t.Fatalf("manifest = %q, %v", data, err)
	}
}

func TestDownloadItemWritesSuccessfulFixture(t *testing.T) {
	directory := t.TempDir()
	path := filepath.Join(directory, "00000001.m4s")
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: make(http.Header), Body: io.NopCloser(strings.NewReader("segment-data")), Request: request}, nil
	})}
	if err := downloadItemFile(context.Background(), client, downloadItem{url: "https://example.com/segment.m4s", path: path}); err != nil {
		t.Fatalf("downloadItemFile returned error: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "segment-data" {
		t.Fatalf("written item = %q, %v", data, err)
	}
}
