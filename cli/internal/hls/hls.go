package hls

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/Eyevinn/hls-m3u8/m3u8"
	"github.com/w1977-0/open-stream-saver/cli/internal/ffmpeg"
	"github.com/w1977-0/open-stream-saver/cli/internal/integrity"
	"github.com/w1977-0/open-stream-saver/cli/internal/progress"
	"github.com/w1977-0/open-stream-saver/cli/internal/retry"
	"github.com/w1977-0/open-stream-saver/cli/internal/safety"
	"golang.org/x/sync/errgroup"
)

const (
	maxPlaylistBytes = 10 * 1024 * 1024
	maxWorkers       = 32
)

type Options struct {
	Client  *http.Client
	Workers int
	Variant int // -1 selects the highest advertised bandwidth from a master playlist.
	Output  string
	Writer  io.Writer
}

type Variant struct {
	Index            int
	URI              string
	Resolution       string
	Bandwidth        uint32
	AverageBandwidth uint32
	Codecs           string
}

type segment struct {
	index int
	url   string
}

// Inspect returns the safe media variants advertised by one public HLS master
// playlist. It rejects encryption and separate renditions rather than trying to
// reconstruct a protected or multi-track presentation.
func Inspect(ctx context.Context, rawURL string, client *http.Client) ([]Variant, error) {
	if client == nil {
		client = &http.Client{}
	}
	playlistURL, err := safety.ValidatePublicURL(rawURL)
	if err != nil {
		return nil, err
	}
	playlist, err := fetchPlaylist(ctx, client, playlistURL.String())
	if err != nil {
		return nil, err
	}
	decoded, kind, err := decodePlaylist(playlist)
	if err != nil {
		return nil, err
	}
	if kind != m3u8.MASTER {
		return nil, fmt.Errorf("URL is a media playlist, not a master playlist")
	}
	master, ok := decoded.(*m3u8.MasterPlaylist)
	if !ok {
		return nil, fmt.Errorf("could not read HLS master playlist")
	}
	return variantsFromMaster(playlistURL, master)
}

// DownloadMediaPlaylist saves one completed, unencrypted HLS media playlist.
// When rawURL points to a master playlist, Variant selects a listed muxed media
// variant. No cookies, credentials, tokens, custom headers, keys, redirects, or
// DRM mechanisms are accepted or forwarded.
func DownloadMediaPlaylist(ctx context.Context, rawURL string, options Options) error {
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

	playlistURL, err := safety.ValidatePublicURL(rawURL)
	if err != nil {
		return err
	}
	playlist, err := fetchPlaylist(ctx, options.Client, playlistURL.String())
	if err != nil {
		return err
	}
	mediaURL, mediaData, err := resolveMediaPlaylist(ctx, playlistURL, playlist, options.Client, options.Variant)
	if err != nil {
		return err
	}
	segments, err := parseSegments(mediaURL, mediaData)
	if err != nil {
		return err
	}
	if len(segments) == 0 {
		return fmt.Errorf("media playlist contains no downloadable segments")
	}

	if err := os.MkdirAll(filepath.Dir(options.Output), 0o755); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp(filepath.Dir(options.Output), ".open-stream-saver-hls-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	tracker := progress.New(int64(len(segments)), "Segments", options.Writer)
	defer tracker.Complete()
	group, groupCtx := errgroup.WithContext(ctx)
	jobs := make(chan segment)
	var pathsMu sync.Mutex
	paths := make([]string, 0, len(segments))

	for range options.Workers {
		group.Go(func() error {
			for item := range jobs {
				path, err := downloadSegment(groupCtx, options.Client, tempDir, item)
				if err != nil {
					return err
				}
				pathsMu.Lock()
				paths = append(paths, path)
				pathsMu.Unlock()
				tracker.Add(1)
			}
			return nil
		})
	}
	group.Go(func() error {
		defer close(jobs)
		for _, item := range segments {
			select {
			case jobs <- item:
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		return err
	}
	sort.Strings(paths)
	concatPath, err := writeConcatFile(tempDir, paths)
	if err != nil {
		return err
	}
	if err := ffmpeg.MergeTS(ctx, concatPath, options.Output); err != nil {
		return err
	}
	return integrity.Report(options.Writer, options.Output)
}

func resolveMediaPlaylist(ctx context.Context, playlistURL *url.URL, data []byte, client *http.Client, requestedVariant int) (*url.URL, []byte, error) {
	decoded, kind, err := decodePlaylist(data)
	if err != nil {
		return nil, nil, err
	}
	if kind == m3u8.MEDIA {
		return playlistURL, data, nil
	}
	if kind != m3u8.MASTER {
		return nil, nil, fmt.Errorf("unsupported HLS playlist kind")
	}
	master, ok := decoded.(*m3u8.MasterPlaylist)
	if !ok {
		return nil, nil, fmt.Errorf("could not read HLS master playlist")
	}
	variants, err := variantsFromMaster(playlistURL, master)
	if err != nil {
		return nil, nil, err
	}
	selected, err := chooseVariant(variants, requestedVariant)
	if err != nil {
		return nil, nil, err
	}
	selectedURL, err := safety.ValidatePublicURL(selected.URI)
	if err != nil {
		return nil, nil, fmt.Errorf("selected variant is not public: %w", err)
	}
	media, err := fetchPlaylist(ctx, client, selectedURL.String())
	if err != nil {
		return nil, nil, err
	}
	decoded, kind, err = decodePlaylist(media)
	if err != nil {
		return nil, nil, err
	}
	if kind == m3u8.MASTER {
		return nil, nil, fmt.Errorf("nested master playlists are not supported")
	}
	if kind != m3u8.MEDIA {
		return nil, nil, fmt.Errorf("selected variant is not a media playlist")
	}
	return selectedURL, media, nil
}

func variantsFromMaster(playlistURL *url.URL, master *m3u8.MasterPlaylist) ([]Variant, error) {
	if len(master.SessionKeys) > 0 || master.ContentSteering != nil {
		return nil, fmt.Errorf("encrypted or content-steered HLS master playlists are not supported")
	}
	variants := make([]Variant, 0, len(master.Variants))
	for index, item := range master.Variants {
		if item == nil || item.URI == "" || item.Iframe {
			continue
		}
		if item.Audio != "" || item.Video != "" || item.Subtitles != "" || item.Captions != "" {
			return nil, fmt.Errorf("separate audio, video, subtitle, or caption renditions are not supported")
		}
		resolved, err := playlistURL.Parse(item.URI)
		if err != nil || (resolved.Scheme != "http" && resolved.Scheme != "https") {
			return nil, fmt.Errorf("master playlist contains a non-HTTP variant URL")
		}
		if _, err := safety.ValidatePublicURL(resolved.String()); err != nil {
			return nil, fmt.Errorf("master playlist contains a non-public variant URL")
		}
		variants = append(variants, Variant{
			Index: index, URI: resolved.String(), Resolution: item.Resolution,
			Bandwidth: item.Bandwidth, AverageBandwidth: item.AverageBandwidth, Codecs: item.Codecs,
		})
	}
	if len(variants) == 0 {
		return nil, fmt.Errorf("master playlist has no supported muxed media variants")
	}
	return variants, nil
}

func chooseVariant(variants []Variant, requested int) (Variant, error) {
	if requested >= 0 {
		for _, item := range variants {
			if item.Index == requested {
				return item, nil
			}
		}
		return Variant{}, fmt.Errorf("variant %d is not available", requested)
	}
	best := variants[0]
	for _, item := range variants[1:] {
		if item.Bandwidth > best.Bandwidth {
			best = item
		}
	}
	return best, nil
}

func decodePlaylist(data []byte) (m3u8.Playlist, m3u8.ListType, error) {
	if len(data) > maxPlaylistBytes {
		return nil, 0, fmt.Errorf("playlist exceeds the local size limit")
	}
	decoded, kind, err := m3u8.DecodeFrom(bytes.NewReader(data), true)
	if err != nil {
		return nil, 0, fmt.Errorf("could not parse HLS playlist: %w", err)
	}
	return decoded, kind, nil
}

func fetchPlaylist(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	var data []byte
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
			return fmt.Errorf("playlist server returned HTTP %d", response.StatusCode), retryableStatus(response.StatusCode)
		}
		body, err := io.ReadAll(io.LimitReader(response.Body, maxPlaylistBytes+1))
		if err != nil {
			return err, true
		}
		if len(body) > maxPlaylistBytes {
			return fmt.Errorf("playlist exceeds the local size limit"), false
		}
		data = body
		return nil, false
	})
	return data, err
}

func parseSegments(playlistURL *url.URL, data []byte) ([]segment, error) {
	decoded, kind, err := decodePlaylist(data)
	if err != nil {
		return nil, err
	}
	if kind == m3u8.MASTER {
		return nil, fmt.Errorf("master playlists must be resolved before parsing segments")
	}
	if kind != m3u8.MEDIA {
		return nil, fmt.Errorf("expected a completed, unencrypted media playlist")
	}
	media, ok := decoded.(*m3u8.MediaPlaylist)
	if !ok || !media.Closed {
		return nil, fmt.Errorf("live or incomplete media playlists are not supported")
	}
	if len(media.Keys) > 0 || media.Map != nil || len(media.PartialSegments) > 0 {
		return nil, fmt.Errorf("encrypted, initialized, or partial HLS streams are not supported")
	}

	segments := make([]segment, 0, len(media.Segments))
	for index, item := range media.Segments {
		if item == nil || item.URI == "" {
			continue
		}
		if len(item.Keys) > 0 || item.Map != nil || item.Limit > 0 || item.Gap {
			return nil, fmt.Errorf("protected, byte-range, or unavailable HLS segments are not supported")
		}
		resolved, err := playlistURL.Parse(item.URI)
		if err != nil || (resolved.Scheme != "http" && resolved.Scheme != "https") {
			return nil, fmt.Errorf("playlist contains a non-HTTP segment URL")
		}
		if _, err := safety.ValidatePublicURL(resolved.String()); err != nil {
			return nil, fmt.Errorf("playlist contains a non-public segment URL")
		}
		segments = append(segments, segment{index: index, url: resolved.String()})
	}
	return segments, nil
}

func downloadSegment(ctx context.Context, client *http.Client, directory string, item segment) (string, error) {
	path := filepath.Join(directory, fmt.Sprintf("%08d.ts", item.index))
	err := retry.Do(ctx, 0, func() (error, bool) {
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
			return fmt.Errorf("segment server returned HTTP %d", response.StatusCode), retryableStatus(response.StatusCode)
		}
		file, err := os.Create(path)
		if err != nil {
			return err, false
		}
		_, copyErr := io.Copy(file, io.LimitReader(response.Body, 256*1024*1024))
		closeErr := file.Close()
		if copyErr != nil {
			_ = os.Remove(path)
			return copyErr, true
		}
		if closeErr != nil {
			_ = os.Remove(path)
			return closeErr, true
		}
		return nil, false
	})
	if err != nil {
		_ = os.Remove(path)
		return "", fmt.Errorf("segment %d failed after bounded retries: %w", item.index, err)
	}
	return path, nil
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
