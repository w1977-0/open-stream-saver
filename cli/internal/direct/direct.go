package direct

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/w1977-0/open-stream-saver/cli/internal/integrity"
	"github.com/w1977-0/open-stream-saver/cli/internal/progress"
	"github.com/w1977-0/open-stream-saver/cli/internal/retry"
	"golang.org/x/sync/errgroup"
)

const (
	chunkSize  int64 = 8 * 1024 * 1024
	maxWorkers       = 32
)

type Options struct {
	Client  *http.Client
	Workers int
	Output  string
	Writer  io.Writer
}

type byteRange struct{ start, end int64 }

// Download saves one public direct file. Range concurrency is used only after
// the server explicitly confirms byte-range support and a stable content length.
// Each completed output is atomically moved into place and reported with SHA-256.
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

	head, err := request(ctx, options.Client, http.MethodHead, rawURL, nil)
	if err != nil || head == nil || head.StatusCode >= http.StatusBadRequest {
		if head != nil {
			_ = head.Body.Close()
		}
		return sequential(ctx, rawURL, options)
	}
	_ = head.Body.Close()

	total := head.ContentLength
	ranges := strings.Contains(strings.ToLower(head.Header.Get("Accept-Ranges")), "bytes")
	if total <= 0 || !ranges || options.Workers == 1 {
		return sequential(ctx, rawURL, options)
	}
	return parallel(ctx, rawURL, total, options)
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

func sequential(ctx context.Context, rawURL string, options Options) error {
	if err := os.MkdirAll(filepath.Dir(options.Output), 0o755); err != nil {
		return err
	}

	var tracker *progress.Tracker
	err := retry.Do(ctx, 0, func() (error, bool) {
		response, err := request(ctx, options.Client, http.MethodGet, rawURL, nil)
		if err != nil {
			return err, true
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
			return fmt.Errorf("server returned HTTP %d", response.StatusCode), retryableStatus(response.StatusCode)
		}

		temporary, err := os.CreateTemp(filepath.Dir(options.Output), ".open-stream-saver-*")
		if err != nil {
			return err, false
		}
		temporaryName := temporary.Name()
		defer os.Remove(temporaryName)

		if tracker == nil {
			tracker = progress.New(response.ContentLength, "Downloading", options.Writer)
		}
		writer := &countingWriter{writer: temporary, add: tracker.Add}
		copied, copyErr := io.Copy(writer, response.Body)
		closeErr := temporary.Close()
		if copyErr != nil {
			return copyErr, true
		}
		if closeErr != nil {
			return closeErr, true
		}
		if response.ContentLength >= 0 && copied != response.ContentLength {
			return fmt.Errorf("downloaded %d bytes, expected %d", copied, response.ContentLength), true
		}
		if err := os.Rename(temporaryName, options.Output); err != nil {
			return err, false
		}
		return nil, false
	})
	if tracker != nil {
		tracker.Complete()
	}
	if err != nil {
		return err
	}
	return integrity.Report(options.Writer, options.Output)
}

func parallel(ctx context.Context, rawURL string, total int64, options Options) error {
	if err := os.MkdirAll(filepath.Dir(options.Output), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(options.Output), ".open-stream-saver-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)
	if err := temporary.Truncate(total); err != nil {
		_ = temporary.Close()
		return err
	}

	jobs := make(chan byteRange)
	group, groupCtx := errgroup.WithContext(ctx)
	tracker := progress.New(total, "Downloading", options.Writer)
	defer tracker.Complete()
	var written atomic.Int64

	for range options.Workers {
		group.Go(func() error {
			for part := range jobs {
				if err := downloadRange(groupCtx, options.Client, rawURL, temporary, part, &written, tracker); err != nil {
					return err
				}
			}
			return nil
		})
	}
	group.Go(func() error {
		defer close(jobs)
		for start := int64(0); start < total; start += chunkSize {
			end := min(start+chunkSize-1, total-1)
			select {
			case jobs <- byteRange{start: start, end: end}:
			case <-groupCtx.Done():
				return groupCtx.Err()
			}
		}
		return nil
	})

	if err := group.Wait(); err != nil {
		_ = temporary.Close()
		return err
	}
	if written.Load() != total {
		_ = temporary.Close()
		return fmt.Errorf("downloaded %d bytes, expected %d", written.Load(), total)
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryName, options.Output); err != nil {
		return err
	}
	return integrity.Report(options.Writer, options.Output)
}

func downloadRange(ctx context.Context, client *http.Client, rawURL string, file *os.File, part byteRange, written *atomic.Int64, tracker *progress.Tracker) error {
	expected := part.end - part.start + 1
	return retry.Do(ctx, 0, func() (error, bool) {
		headers := http.Header{}
		headers.Set("Range", fmt.Sprintf("bytes=%d-%d", part.start, part.end))
		response, err := request(ctx, client, http.MethodGet, rawURL, headers)
		if err != nil {
			return err, true
		}
		defer response.Body.Close()
		if response.StatusCode != http.StatusPartialContent {
			return fmt.Errorf("server did not honor byte-range request"), retryableStatus(response.StatusCode)
		}
		if !validContentRange(response.Header.Get("Content-Range"), part) {
			return fmt.Errorf("server returned an unexpected content range"), false
		}

		buffer := make([]byte, 64*1024)
		remaining := expected
		offset := part.start
		for remaining > 0 {
			read, readErr := response.Body.Read(buffer)
			if read > 0 {
				if int64(read) > remaining {
					return fmt.Errorf("server returned more data than requested"), false
				}
				count, writeErr := file.WriteAt(buffer[:read], offset)
				if writeErr != nil {
					return writeErr, false
				}
				if count != read {
					return io.ErrShortWrite, false
				}
				offset += int64(read)
				remaining -= int64(read)
			}
			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				return readErr, true
			}
		}
		if remaining != 0 {
			return fmt.Errorf("incomplete byte range"), true
		}
		written.Add(expected)
		tracker.Add(int(expected))
		return nil, false
	})
}

func validContentRange(value string, part byteRange) bool {
	prefix := fmt.Sprintf("bytes %d-%d/", part.start, part.end)
	return strings.HasPrefix(strings.ToLower(value), strings.ToLower(prefix)) && len(value) > len(prefix)
}

func retryableStatus(status int) bool {
	return status == http.StatusRequestTimeout || status == http.StatusTooEarly || status == http.StatusTooManyRequests || status >= 500
}

func ensureNewOutput(output string) error {
	if _, err := os.Lstat(output); err == nil {
		return fmt.Errorf("refusing to overwrite existing output: %s", output)
	} else if !os.IsNotExist(err) {
		return err
	}
	return nil
}

func request(ctx context.Context, client *http.Client, method, rawURL string, headers http.Header) (*http.Response, error) {
	request, err := http.NewRequestWithContext(ctx, method, rawURL, nil)
	if err != nil {
		return nil, err
	}
	for key, values := range headers {
		for _, value := range values {
			request.Header.Add(key, value)
		}
	}
	return client.Do(request)
}

type countingWriter struct {
	writer io.Writer
	add    func(int)
}

func (writer *countingWriter) Write(data []byte) (int, error) {
	count, err := writer.writer.Write(data)
	writer.add(count)
	return count, err
}

// contentLength is kept visible to tests that exercise malformed responses.
func contentLength(header http.Header) int64 {
	value, err := strconv.ParseInt(header.Get("Content-Length"), 10, 64)
	if err != nil {
		return -1
	}
	return value
}
