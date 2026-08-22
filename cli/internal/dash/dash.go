// Package dash implements a deliberately narrow MPEG-DASH downloader for media
// that is public, static, unencrypted, and explicitly authorized by the user.
// It rejects ContentProtection and authenticated delivery rather than attempting
// to work around them.
package dash

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"github.com/w1977-0/media-archiver/cli/internal/ffmpeg"
	"github.com/w1977-0/media-archiver/cli/internal/integrity"
	"github.com/w1977-0/media-archiver/cli/internal/progress"
	"github.com/w1977-0/media-archiver/cli/internal/retry"
	"github.com/w1977-0/media-archiver/cli/internal/safety"
	"golang.org/x/sync/errgroup"
)

const (
	maxManifestBytes = 10 * 1024 * 1024
	maxWorkers       = 32
	maxSegments      = 100_000
)

type Options struct {
	Client  *http.Client
	Workers int
	Output  string
	Writer  io.Writer
}

type mpd struct {
	XMLName           xml.Name   `xml:"MPD"`
	Type              string     `xml:"type,attr"`
	MediaPresentation string     `xml:"mediaPresentationDuration,attr"`
	BaseURLs          []string   `xml:"BaseURL"`
	ContentProtection []struct{} `xml:"ContentProtection"`
	Periods           []period   `xml:"Period"`
}

type period struct {
	Duration          string          `xml:"duration,attr"`
	BaseURLs          []string        `xml:"BaseURL"`
	ContentProtection []struct{}      `xml:"ContentProtection"`
	AdaptationSets    []adaptationSet `xml:"AdaptationSet"`
}

type adaptationSet struct {
	ContentType       string           `xml:"contentType,attr"`
	MimeType          string           `xml:"mimeType,attr"`
	BaseURLs          []string         `xml:"BaseURL"`
	ContentProtection []struct{}       `xml:"ContentProtection"`
	SegmentTemplate   *segmentTemplate `xml:"SegmentTemplate"`
	Representations   []representation `xml:"Representation"`
}

type representation struct {
	ID                string           `xml:"id,attr"`
	Bandwidth         int64            `xml:"bandwidth,attr"`
	MimeType          string           `xml:"mimeType,attr"`
	BaseURLs          []string         `xml:"BaseURL"`
	ContentProtection []struct{}       `xml:"ContentProtection"`
	SegmentTemplate   *segmentTemplate `xml:"SegmentTemplate"`
}

type segmentTemplate struct {
	Timescale      int64            `xml:"timescale,attr"`
	Duration       int64            `xml:"duration,attr"`
	StartNumber    int64            `xml:"startNumber,attr"`
	Media          string           `xml:"media,attr"`
	Initialization string           `xml:"initialization,attr"`
	Timeline       *segmentTimeline `xml:"SegmentTimeline"`
}

type segmentTimeline struct {
	Entries []timelineEntry `xml:"S"`
}

type timelineEntry struct {
	Time     int64 `xml:"t,attr"`
	Duration int64 `xml:"d,attr"`
	Repeat   int64 `xml:"r,attr"`
}

type trackPlan struct {
	kind        string
	initURL     string
	segmentURLs []string
}

type plan struct {
	video *trackPlan
	audio *trackPlan
}

// Download saves a public static DASH presentation. It supports only direct
// SegmentTemplate addressing with one video representation and an optional
// audio representation. ContentProtection, dynamic MPDs, separate periods,
// SegmentBase, SegmentList, and signed/authenticated transport are rejected.
func Download(ctx context.Context, rawURL string, options Options) error {
	if options.Client == nil {
		options.Client = &http.Client{}
	}
	options.Workers = workerCount(options.Workers)
	if options.Output == "" {
		return fmt.Errorf("an output file path is required")
	}
	if err := ensureNewOutput(options.Output); err != nil {
		return err
	}

	manifestURL, err := safety.ValidatePublicURL(rawURL)
	if err != nil {
		return err
	}
	data, err := fetchManifest(ctx, options.Client, manifestURL.String())
	if err != nil {
		return err
	}
	parsed, err := parseManifest(data)
	if err != nil {
		return err
	}
	job, err := buildPlan(manifestURL, parsed)
	if err != nil {
		return err
	}
	if job.video == nil {
		return fmt.Errorf("DASH presentation contains no supported video representation")
	}

	if err := os.MkdirAll(filepath.Dir(options.Output), 0o755); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(filepath.Dir(options.Output), ".open-stream-saver-dash-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	videoConcat, err := downloadTrack(ctx, tempDir, options, job.video)
	if err != nil {
		return err
	}
	audioConcat := ""
	if job.audio != nil {
		audioConcat, err = downloadTrack(ctx, tempDir, options, job.audio)
		if err != nil {
			return err
		}
	}
	if err := ffmpeg.MergeDASH(ctx, videoConcat, audioConcat, options.Output); err != nil {
		return err
	}
	return integrity.Report(options.Writer, options.Output)
}

func fetchManifest(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	var result []byte
	err := retry.Do(ctx, 0, func() (error, bool) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
		if err != nil {
			return err, false
		}
		response, err := client.Do(request)
		if err != nil {
			return err, true
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("MPD server returned HTTP %d", response.StatusCode), retryableStatus(response.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, maxManifestBytes+1))
		if err != nil {
			return err, true
		}
		if len(body) > maxManifestBytes {
			return fmt.Errorf("MPD exceeds the local size limit"), false
		}
		result = body
		return nil, false
	})
	return result, err
}

func parseManifest(data []byte) (*mpd, error) {
	if len(data) > maxManifestBytes {
		return nil, fmt.Errorf("MPD exceeds the local size limit")
	}
	var result mpd
	if err := xml.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("could not parse MPD: %w", err)
	}
	if result.XMLName.Local != "MPD" {
		return nil, fmt.Errorf("document is not an MPEG-DASH MPD")
	}
	if strings.EqualFold(result.Type, "dynamic") {
		return nil, fmt.Errorf("live or dynamic DASH presentations are not supported")
	}
	if len(result.ContentProtection) > 0 {
		return nil, fmt.Errorf("DRM-protected DASH content is not supported")
	}
	if len(result.Periods) != 1 {
		return nil, fmt.Errorf("only one-period static DASH presentations are supported")
	}
	return &result, nil
}

func buildPlan(manifestURL *url.URL, document *mpd) (*plan, error) {
	period := document.Periods[0]
	if len(period.ContentProtection) > 0 {
		return nil, fmt.Errorf("DRM-protected DASH content is not supported")
	}
	duration, err := presentationDuration(document.MediaPresentation, period.Duration)
	if err != nil {
		return nil, err
	}
	rootBase, err := resolveBase(manifestURL, document.BaseURLs)
	if err != nil {
		return nil, err
	}
	periodBase, err := resolveBase(rootBase, period.BaseURLs)
	if err != nil {
		return nil, err
	}

	result := &plan{}
	for _, set := range period.AdaptationSets {
		if len(set.ContentProtection) > 0 {
			return nil, fmt.Errorf("DRM-protected DASH content is not supported")
		}
		kind := mediaKind(set.ContentType, set.MimeType)
		if kind == "" || (kind == "video" && result.video != nil) || (kind == "audio" && result.audio != nil) {
			continue
		}
		track, err := selectTrack(periodBase, set, kind, duration)
		if err != nil {
			return nil, err
		}
		if kind == "video" {
			result.video = track
		} else {
			result.audio = track
		}
	}
	return result, nil
}

func selectTrack(parent *url.URL, set adaptationSet, kind string, duration float64) (*trackPlan, error) {
	setBase, err := resolveBase(parent, set.BaseURLs)
	if err != nil {
		return nil, err
	}
	var selected *representation
	for index := range set.Representations {
		candidate := &set.Representations[index]
		if len(candidate.ContentProtection) > 0 {
			return nil, fmt.Errorf("DRM-protected DASH content is not supported")
		}
		if selected == nil || candidate.Bandwidth > selected.Bandwidth {
			selected = candidate
		}
	}
	if selected == nil || selected.ID == "" {
		return nil, fmt.Errorf("%s adaptation set has no usable representation", kind)
	}
	representationBase, err := resolveBase(setBase, selected.BaseURLs)
	if err != nil {
		return nil, err
	}
	template := inheritedTemplate(set.SegmentTemplate, selected.SegmentTemplate)
	if template == nil || template.Media == "" || template.Initialization == "" {
		return nil, fmt.Errorf("%s representation requires unsupported DASH addressing", kind)
	}
	if template.Timeline != nil {
		return nil, fmt.Errorf("SegmentTimeline DASH addressing is not supported yet")
	}
	if template.Duration <= 0 || duration <= 0 {
		return nil, fmt.Errorf("%s representation requires a static SegmentTemplate duration", kind)
	}
	if template.Timescale <= 0 {
		template.Timescale = 1
	}
	startNumber := template.StartNumber
	if startNumber <= 0 {
		startNumber = 1
	}
	count := int64(math.Ceil(duration * float64(template.Timescale) / float64(template.Duration)))
	if count < 1 || count > maxSegments {
		return nil, fmt.Errorf("%s representation has an unsupported segment count", kind)
	}
	initURL, err := resolveTemplateURL(representationBase, template.Initialization, selected.ID, startNumber, 0)
	if err != nil {
		return nil, err
	}
	segments := make([]string, 0, count)
	for number := startNumber; number < startNumber+count; number++ {
		item, err := resolveTemplateURL(representationBase, template.Media, selected.ID, number, 0)
		if err != nil {
			return nil, err
		}
		segments = append(segments, item)
	}
	return &trackPlan{kind: kind, initURL: initURL, segmentURLs: segments}, nil
}

func inheritedTemplate(parent, child *segmentTemplate) *segmentTemplate {
	if parent == nil && child == nil {
		return nil
	}
	result := segmentTemplate{}
	if parent != nil {
		result = *parent
	}
	if child == nil {
		return &result
	}
	if child.Timescale != 0 {
		result.Timescale = child.Timescale
	}
	if child.Duration != 0 {
		result.Duration = child.Duration
	}
	if child.StartNumber != 0 {
		result.StartNumber = child.StartNumber
	}
	if child.Media != "" {
		result.Media = child.Media
	}
	if child.Initialization != "" {
		result.Initialization = child.Initialization
	}
	if child.Timeline != nil {
		result.Timeline = child.Timeline
	}
	return &result
}

func downloadTrack(ctx context.Context, directory string, options Options, track *trackPlan) (string, error) {
	trackDirectory := filepath.Join(directory, track.kind)
	if err := os.MkdirAll(trackDirectory, 0o700); err != nil {
		return "", err
	}
	entries := make([]downloadItem, 0, len(track.segmentURLs)+1)
	entries = append(entries, downloadItem{index: 0, url: track.initURL, path: filepath.Join(trackDirectory, "00000000.init")})
	for index, rawURL := range track.segmentURLs {
		entries = append(entries, downloadItem{index: index + 1, url: rawURL, path: filepath.Join(trackDirectory, fmt.Sprintf("%08d.m4s", index+1))})
	}
	tracker := progress.New(int64(len(entries)), strings.Title(track.kind)+" segments", options.Writer)
	defer tracker.Complete()
	jobs := make(chan downloadItem)
	group, groupCtx := errgroup.WithContext(ctx)
	for range options.Workers {
		group.Go(func() error {
			for item := range jobs {
				if err := downloadItemFile(groupCtx, options.Client, item); err != nil {
					return err
				}
				tracker.Add(1)
			}
			return nil
		})
	}
	group.Go(func() error {
		defer close(jobs)
		for _, item := range entries {
			select {
			case jobs <- item:
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
		return nil
	})
	if err := group.Wait(); err != nil {
		return "", err
	}
	paths := make([]string, 0, len(entries))
	for _, item := range entries {
		paths = append(paths, item.path)
	}
	sort.Strings(paths)
	return writeConcatFile(trackDirectory, paths)
}

type downloadItem struct {
	index     int
	url, path string
}

func downloadItemFile(ctx context.Context, client *http.Client, item downloadItem) error {
	return retry.Do(ctx, 0, func() (error, bool) {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, item.url, nil)
		if err != nil {
			return err, false
		}
		response, err := client.Do(request)
		if err != nil {
			return err, true
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK {
			return fmt.Errorf("DASH segment server returned HTTP %d", response.StatusCode), retryableStatus(response.StatusCode)
		}
		file, err := os.Create(item.path)
		if err != nil {
			return err, false
		}
		_, copyErr := io.Copy(file, io.LimitReader(response.Body, 256*1024*1024))
		closeErr := file.Close()
		if copyErr != nil {
			_ = os.Remove(item.path)
			return copyErr, true
		}
		if closeErr != nil {
			_ = os.Remove(item.path)
			return closeErr, true
		}
		return nil, false
	})
}

func writeConcatFile(directory string, paths []string) (string, error) {
	path := filepath.Join(directory, "segments.txt")
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	for _, item := range paths {
		if strings.Contains(item, "'") {
			_ = file.Close()
			return "", fmt.Errorf("unexpected quote in temporary path")
		}
		if _, err := fmt.Fprintf(file, "file '%s'\n", item); err != nil {
			_ = file.Close()
			return "", err
		}
	}
	if err := file.Close(); err != nil {
		return "", err
	}
	return path, nil
}

func resolveBase(parent *url.URL, values []string) (*url.URL, error) {
	if len(values) == 0 {
		return parent, nil
	}
	if len(values) != 1 || strings.TrimSpace(values[0]) == "" {
		return nil, fmt.Errorf("multiple or empty DASH BaseURL values are not supported")
	}
	resolved, err := parent.Parse(strings.TrimSpace(values[0]))
	if err != nil {
		return nil, err
	}
	validated, err := safety.ValidatePublicURL(resolved.String())
	if err != nil {
		return nil, fmt.Errorf("DASH BaseURL is not public: %w", err)
	}
	return validated, nil
}

func resolveTemplateURL(base *url.URL, pattern, representationID string, number, timeValue int64) (string, error) {
	if strings.Contains(pattern, "$") && (strings.Contains(pattern, "%") || strings.Contains(pattern, "Time")) {
		return "", fmt.Errorf("unsupported DASH template token in %q", pattern)
	}
	resolvedPattern := strings.ReplaceAll(pattern, "$RepresentationID$", representationID)
	resolvedPattern = strings.ReplaceAll(resolvedPattern, "$Number$", strconv.FormatInt(number, 10))
	resolvedPattern = strings.ReplaceAll(resolvedPattern, "$Time$", strconv.FormatInt(timeValue, 10))
	if strings.Contains(resolvedPattern, "$") {
		return "", fmt.Errorf("unresolved DASH template token in %q", pattern)
	}
	resolved, err := base.Parse(resolvedPattern)
	if err != nil {
		return "", err
	}
	validated, err := safety.ValidatePublicURL(resolved.String())
	if err != nil {
		return "", fmt.Errorf("DASH segment URL is not public: %w", err)
	}
	return validated.String(), nil
}

func mediaKind(contentType, mimeType string) string {
	value := strings.ToLower(strings.TrimSpace(contentType))
	if value == "video" || value == "audio" {
		return value
	}
	mime := strings.ToLower(strings.TrimSpace(mimeType))
	if strings.HasPrefix(mime, "video/") {
		return "video"
	}
	if strings.HasPrefix(mime, "audio/") {
		return "audio"
	}
	return ""
}

var isoDuration = regexp.MustCompile(`^PT(?:(\d+(?:\.\d+)?)H)?(?:(\d+(?:\.\d+)?)M)?(?:(\d+(?:\.\d+)?)S)?$`)

func presentationDuration(mpdDuration, periodDuration string) (float64, error) {
	value := periodDuration
	if value == "" {
		value = mpdDuration
	}
	matches := isoDuration.FindStringSubmatch(value)
	if matches == nil {
		return 0, fmt.Errorf("DASH presentation requires a valid static ISO-8601 duration")
	}
	units := []float64{3600, 60, 1}
	var total float64
	for index, item := range matches[1:] {
		if item == "" {
			continue
		}
		parsed, err := strconv.ParseFloat(item, 64)
		if err != nil || parsed < 0 {
			return 0, fmt.Errorf("invalid DASH duration")
		}
		total += parsed * units[index]
	}
	if total <= 0 {
		return 0, fmt.Errorf("DASH duration must be positive")
	}
	return total, nil
}

func retryableStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooEarly || status == http.StatusTooManyRequests || status >= 500
}
func workerCount(workers int) int {
	if workers < 1 {
		return 1
	}
	if workers > maxWorkers {
		return maxWorkers
	}
	return workers
}
func ensureNewOutput(output string) error {
	if _, err := os.Lstat(output); err == nil {
		return fmt.Errorf("refusing to overwrite existing output: %s", output)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}
