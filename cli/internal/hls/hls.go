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
	"github.com/w1977-0/open-stream-saver/cli/internal/progress"
	"github.com/w1977-0/open-stream-saver/cli/internal/safety"
	"golang.org/x/sync/errgroup"
)

const maxPlaylistBytes = 10 * 1024 * 1024

type Options struct {
	Client  *http.Client
	Workers int
	Output  string
	Writer  io.Writer
}

type segment struct {
	index int
	url   string
}

// DownloadMediaPlaylist supports completed, unencrypted media playlists only.
// Master playlists, live playlists, EXT-X-KEY, EXT-X-MAP, partial segments and
// byte-range segments are intentionally rejected rather than approximated.
func DownloadMediaPlaylist(ctx context.Context, rawURL string, options Options) error {
	if options.Client == nil {
		options.Client = &http.Client{}
	}
	if options.Workers < 1 {
		options.Workers = 1
	}
	if options.Output == "" {
		return fmt.Errorf("an output file path is required")
	}

	playlistURL, err := safety.ValidatePublicURL(rawURL)
	if err != nil {
		return err
	}
	playlist, err := fetchPlaylist(ctx, options.Client, playlistURL.String())
	if err != nil {
		return err
	}
	segments, err := parseSegments(playlistURL, playlist)
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
	return ffmpeg.MergeTS(ctx, concatPath, options.Output)
}

func fetchPlaylist(ctx context.Context, client *http.Client, rawURL string) ([]byte, error) {
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("playlist server returned HTTP %d", response.StatusCode)
	}
	return io.ReadAll(io.LimitReader(response.Body, maxPlaylistBytes+1))
}

func parseSegments(playlistURL *url.URL, data []byte) ([]segment, error) {
	if len(data) > maxPlaylistBytes {
		return nil, fmt.Errorf("playlist exceeds the local size limit")
	}
	decoded, kind, err := m3u8.DecodeFrom(bytes.NewReader(data), true)
	if err != nil {
		return nil, fmt.Errorf("could not parse HLS playlist: %w", err)
	}
	if kind != m3u8.MEDIA {
		return nil, fmt.Errorf("master playlists are not supported; provide one completed, unencrypted media playlist")
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
		if err != nil || resolved.Scheme != "http" && resolved.Scheme != "https" {
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
	var lastErr error
	for attempt := 1; attempt <= 3; attempt++ {
		request, err := http.NewRequestWithContext(ctx, http.MethodGet, item.url, nil)
		if err != nil {
			return "", err
		}
		response, err := client.Do(request)
		if err == nil && response.StatusCode == http.StatusOK {
			file, createErr := os.Create(path)
			if createErr == nil {
				_, copyErr := io.Copy(file, io.LimitReader(response.Body, 256*1024*1024))
				closeErr := file.Close()
				response.Body.Close()
				if copyErr == nil && closeErr == nil {
					return path, nil
				}
				lastErr = firstError(copyErr, closeErr)
			} else {
				lastErr = createErr
				response.Body.Close()
			}
		} else {
			if response != nil {
				response.Body.Close()
			}
			if err != nil {
				lastErr = err
			} else {
				lastErr = fmt.Errorf("segment server returned HTTP %d", response.StatusCode)
			}
		}
		_ = os.Remove(path)
	}
	return "", fmt.Errorf("segment %d failed after 3 attempts: %w", item.index, lastErr)
}

func writeConcatFile(directory string, paths []string) (string, error) {
	path := filepath.Join(directory, "segments.txt")
	file, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	for _, item := range paths {
		if strings.Contains(item, "'") {
			return "", fmt.Errorf("unexpected quote in temporary path")
		}
		if _, err := fmt.Fprintf(file, "file '%s'\n", item); err != nil {
			return "", err
		}
	}
	return path, file.Close()
}

func firstError(errors ...error) error {
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}
