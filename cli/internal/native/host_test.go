package native

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"strings"
	"testing"
)

func TestReadAndWriteFrame(t *testing.T) {
	var encoded bytes.Buffer
	if err := writeFrame(&encoded, Response{OK: true, Output: "/tmp/example.mp4"}); err != nil {
		t.Fatalf("writeFrame returned error: %v", err)
	}
	payload, err := readFrame(&encoded)
	if err != nil {
		t.Fatalf("readFrame returned error: %v", err)
	}
	var response Response
	if err := json.Unmarshal(payload, &response); err != nil {
		t.Fatal(err)
	}
	if !response.OK || response.Output != "/tmp/example.mp4" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestReadFrameRejectsOversizedMessage(t *testing.T) {
	var encoded bytes.Buffer
	var size [4]byte
	binary.LittleEndian.PutUint32(size[:], maxMessageBytes+1)
	encoded.Write(size[:])
	if _, err := readFrame(&encoded); err == nil {
		t.Fatal("expected oversized message rejection")
	}
}

func TestDecodeRequestRejectsUnknownFields(t *testing.T) {
	_, err := decodeRequest([]byte(`{"action":"download","url":"https://example.com/video.mp4","headers":{"Authorization":"no"}}`))
	if err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("expected unknown field rejection, got %v", err)
	}
}

func TestSafeFileNameRejectsPath(t *testing.T) {
	if _, err := safeFileName("../escape.mp4"); err == nil {
		t.Fatal("expected path rejection")
	}
	name, err := safeFileName("clip")
	if err != nil {
		t.Fatal(err)
	}
	if name != "clip.mp4" {
		t.Fatalf("name = %q, want clip.mp4", name)
	}
}
