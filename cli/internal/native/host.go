// Package native implements Chrome Native Messaging framing for the local host.
// The host deliberately accepts only an explicit public-media download request.
// It does not receive or process cookies, headers, tokens, browser storage, or
// arbitrary filesystem paths.
package native

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/w1977-0/open-stream-saver/cli/internal/app"
)

const (
	maxMessageBytes = 1 << 20
	defaultWorkers  = 4
)

type Request struct {
	Action            string `json:"action"`
	URL               string `json:"url"`
	FileName          string `json:"fileName,omitempty"`
	Workers           int    `json:"workers,omitempty"`
	Variant           int    `json:"variant,omitempty"`
	AcknowledgeRights bool   `json:"acknowledgeRights"`
}

type Response struct {
	OK     bool   `json:"ok"`
	Output string `json:"output,omitempty"`
	Error  string `json:"error,omitempty"`
}

// Serve reads framed JSON requests from Chrome and writes one framed response
// per request. stdout is reserved exclusively for the Native Messaging protocol;
// callers should direct human-readable diagnostics to stderr.
func Serve(ctx context.Context, input io.Reader, output io.Writer, diagnostics io.Writer) error {
	for {
		payload, err := readFrame(input)
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		request, err := decodeRequest(payload)
		if err != nil {
			if writeErr := writeFrame(output, Response{Error: err.Error()}); writeErr != nil {
				return writeErr
			}
			continue
		}
		response := execute(ctx, request, diagnostics)
		if err := writeFrame(output, response); err != nil {
			return err
		}
	}
}

func execute(ctx context.Context, request Request, diagnostics io.Writer) Response {
	if request.Action != "download" {
		return Response{Error: "unsupported native action"}
	}
	if !request.AcknowledgeRights {
		return Response{Error: "rights acknowledgement is required"}
	}
	fileName, err := safeFileName(request.FileName)
	if err != nil {
		return Response{Error: err.Error()}
	}
	workers := request.Workers
	if workers == 0 {
		workers = defaultWorkers
	}
	output, err := defaultOutputPath(fileName)
	if err != nil {
		return Response{Error: err.Error()}
	}
	if err := app.DownloadAuthorized(ctx, request.URL, output, workers, request.Variant, 30*time.Minute, diagnostics); err != nil {
		return Response{Error: err.Error()}
	}
	return Response{OK: true, Output: output}
}

func decodeRequest(payload []byte) (Request, error) {
	decoder := json.NewDecoder(strings.NewReader(string(payload)))
	decoder.DisallowUnknownFields()
	var request Request
	if err := decoder.Decode(&request); err != nil {
		return Request{}, fmt.Errorf("invalid native request: %w", err)
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return Request{}, fmt.Errorf("invalid native request: trailing JSON values")
	}
	return request, nil
}

func safeFileName(value string) (string, error) {
	if value == "" {
		return fmt.Sprintf("authorized-download-%s.mp4", time.Now().UTC().Format("20060102-150405")), nil
	}
	if len(value) > 180 || filepath.Base(value) != value || strings.Contains(value, "/") || strings.Contains(value, "\\\\") || value == "." || value == ".." {
		return "", fmt.Errorf("fileName must be a short base filename")
	}
	if filepath.Ext(value) == "" {
		value += ".mp4"
	}
	return value, nil
}

func defaultOutputPath(fileName string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not resolve your home directory: %w", err)
	}
	directory := filepath.Join(home, "Downloads")
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(directory, fileName), nil
}

func readFrame(input io.Reader) ([]byte, error) {
	var lengthBytes [4]byte
	if _, err := io.ReadFull(input, lengthBytes[:]); err != nil {
		return nil, err
	}
	length := binary.LittleEndian.Uint32(lengthBytes[:])
	if length == 0 || length > maxMessageBytes {
		return nil, fmt.Errorf("native message size is invalid")
	}
	payload := make([]byte, length)
	if _, err := io.ReadFull(input, payload); err != nil {
		return nil, err
	}
	return payload, nil
}

func writeFrame(output io.Writer, response Response) error {
	payload, err := json.Marshal(response)
	if err != nil {
		return err
	}
	if len(payload) > maxMessageBytes {
		return fmt.Errorf("native response is too large")
	}
	var lengthBytes [4]byte
	binary.LittleEndian.PutUint32(lengthBytes[:], uint32(len(payload)))
	if _, err := output.Write(lengthBytes[:]); err != nil {
		return err
	}
	_, err = output.Write(payload)
	return err
}
