package dash

import (
	"net/url"
	"strings"
	"testing"
)

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

func TestPresentationDuration(t *testing.T) {
	seconds, err := presentationDuration("PT1H2M3.5S", "")
	if err != nil || seconds != 3723.5 {
		t.Fatalf("duration = %v, %v", seconds, err)
	}
}
