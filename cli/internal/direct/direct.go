package direct

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"

	"github.com/w1977-0/open-stream-saver/cli/internal/progress"
	"golang.org/x/sync/errgroup"
)

const chunkSize int64 = 8 * 1024 * 1024

type Options struct {
	Client  *http.Client
	Workers int
	Output  string
	Writer  io.Writer
}

type byteRange struct{ start, end int64 }

// Download saves one public direct file. Range concurrency is only used after
// the server explicitly confirms byte-range support and a stable content length.
func Download(ctx context.Context, rawURL string, options Options) error {
	if options.Client == nil {
		options.Client = &http.Client{}
	}
	if options.Workers < 1 {
		options.Workers = 1
	}
	if options.Output == "" {
		return fmt.Errorf("an output file path is required")
	}

	head, err := request(ctx, options.Client, http.MethodHead, rawURL, nil)
	if err != nil || head.StatusCode >= 400 {
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

func sequential(ctx context.Context, rawURL string, options Options) error {
	response, err := request(ctx, options.Client, http.MethodGet, rawURL, nil)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK && response.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server returned HTTP %d", response.StatusCode)
	}
	if err := os.MkdirAll(filepath.Dir(options.Output), 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(options.Output), ".open-stream-saver-*")
	if err != nil {
		return err
	}
	temporaryName := temporary.Name()
	defer os.Remove(temporaryName)

	tracker := progress.New(response.ContentLength, "Downloading", options.Writer)
	writer := &countingWriter{writer: temporary, add: tracker.Add}
	_, copyErr := io.Copy(writer, response.Body)
	closeErr := temporary.Close()
	tracker.Complete()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(temporaryName, options.Output)
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
	defer temporary.Close()
	if err := temporary.Truncate(total); err != nil {
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
		return err
	}
	if written.Load() != total {
		return fmt.Errorf("downloaded %d bytes, expected %d", written.Load(), total)
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	return os.Rename(temporaryName, options.Output)
}

func downloadRange(ctx context.Context, client *http.Client, rawURL string, file *os.File, part byteRange, written *atomic.Int64, tracker *progress.Tracker) error {
	headers := http.Header{}
	headers.Set("Range", fmt.Sprintf("bytes=%d-%d", part.start, part.end))
	response, err := request(ctx, client, http.MethodGet, rawURL, headers)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusPartialContent {
		return fmt.Errorf("server did not honor byte-range request")
	}

	remaining := part.end - part.start + 1
	offset := part.start
	buffer := make([]byte, 64*1024)
	for remaining > 0 {
		read, readErr := response.Body.Read(buffer)
		if read > 0 {
			if read > int(remaining) {
				return fmt.Errorf("server returned more data than requested")
			}
			count, writeErr := file.WriteAt(buffer[:read], offset)
			if writeErr != nil {
				return writeErr
			}
			if count != read {
				return io.ErrShortWrite
			}
			offset += int64(read)
			remaining -= int64(read)
			written.Add(int64(read))
			tracker.Add(read)
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}
	if remaining != 0 {
		return fmt.Errorf("incomplete byte range")
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
