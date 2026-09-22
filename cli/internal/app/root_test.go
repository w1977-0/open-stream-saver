package app

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"
)

// These tests stay on the safe side of the network boundary: every case
// either exercises a pure function or fails a check that runs before any
// request is built. No test here resolves a remote host.

func TestDefaultOutputKeepsAUsableFilename(t *testing.T) {
	parsed, err := url.Parse("https://example.org/pub/clip.mp4")
	if err != nil {
		t.Fatalf("url.Parse: %v", err)
	}
	if got := defaultOutput(parsed); got != "clip.mp4" {
		t.Fatalf("defaultOutput = %q, want %q", got, "clip.mp4")
	}
}

func TestDefaultOutputFallsBackForPlaylistsAndBarePaths(t *testing.T) {
	cases := map[string]string{
		"https://example.org/master.m3u8":  "authorized-download.mp4",
		"https://example.org/MASTER.M3U8":  "authorized-download.mp4", // case-insensitive
		"https://example.org/manifest.mpd": "authorized-download.mp4",
		"https://example.org/":             "authorized-download.mp4",
		"https://example.org":              "authorized-download.mp4",
	}
	for raw, want := range cases {
		parsed, err := url.Parse(raw)
		if err != nil {
			t.Fatalf("url.Parse(%q): %v", raw, err)
		}
		if got := defaultOutput(parsed); got != want {
			t.Errorf("defaultOutput(%q) = %q, want %q", raw, got, want)
		}
	}
}

func TestNewClientRefusesToFollowRedirects(t *testing.T) {
	client := newClient(30 * time.Second)
	if client.Timeout != 30*time.Second {
		t.Fatalf("Timeout = %v, want 30s", client.Timeout)
	}
	if client.CheckRedirect == nil {
		t.Fatal("CheckRedirect is nil: a redirect would be followed")
	}
	if err := client.CheckRedirect(nil, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("CheckRedirect = %v, want http.ErrUseLastResponse", err)
	}
}

func TestDownloadAuthorizedRejectsBadOptionsBeforeAnyRequest(t *testing.T) {
	cases := []struct {
		name    string
		workers int
		timeout time.Duration
	}{
		{"workers below one", 0, time.Minute},
		{"workers above the cap", maxWorkers + 1, time.Minute},
		{"non-positive timeout", 4, 0},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			err := DownloadAuthorized(
				context.Background(), "https://example.org/clip.mp4", "",
				item.workers, -1, item.timeout, io.Discard,
			)
			if err == nil {
				t.Fatal("expected an error before any network work")
			}
		})
	}
}

func TestDownloadAuthorizedRejectsLoopbackTarget(t *testing.T) {
	// 127.0.0.1 resolves without asking a name server, so this stays local
	// while proving the private-target gate runs before the request.
	err := DownloadAuthorized(
		context.Background(), "http://127.0.0.1/clip.mp4", "",
		4, -1, time.Minute, io.Discard,
	)
	if err == nil {
		t.Fatal("loopback target must be rejected before any request")
	}
}

func TestDownloadCommandDefaults(t *testing.T) {
	command := newDownloadCommand()
	if command.Use != "download" {
		t.Fatalf("Use = %q, want %q", command.Use, "download")
	}
	workers, err := command.Flags().GetInt("workers")
	if err != nil {
		t.Fatalf("workers flag: %v", err)
	}
	if workers != 4 {
		t.Errorf("default workers = %d, want 4", workers)
	}
	variant, err := command.Flags().GetInt("variant")
	if err != nil {
		t.Fatalf("variant flag: %v", err)
	}
	if variant != -1 {
		t.Errorf("default variant = %d, want -1 (highest advertised)", variant)
	}
	acknowledged, err := command.Flags().GetBool("acknowledge-rights")
	if err != nil {
		t.Fatalf("acknowledge-rights flag: %v", err)
	}
	if acknowledged {
		t.Error("acknowledge-rights must default to false")
	}
}

func TestDownloadRefusesWithoutAcknowledgement(t *testing.T) {
	command := newDownloadCommand()
	command.SetArgs([]string{"--url", "https://example.org/clip.mp4"})
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	err := command.Execute()
	if err == nil {
		t.Fatal("expected the rights gate to refuse")
	}
	if !strings.Contains(err.Error(), "acknowledge") {
		t.Fatalf("error = %v, want it to name the acknowledgement flag", err)
	}
}

func TestDownloadRequiresURL(t *testing.T) {
	command := newDownloadCommand()
	command.SetArgs([]string{})
	command.SetOut(io.Discard)
	command.SetErr(io.Discard)
	err := command.Execute()
	if err == nil {
		t.Fatal("expected the required-url check to fail")
	}
	if !strings.Contains(err.Error(), "url") {
		t.Fatalf("error = %v, want it to name the missing url flag", err)
	}
}

func TestInspectHLSCommandDefaults(t *testing.T) {
	command := newInspectHLSCommand()
	if command.Use != "inspect-hls" {
		t.Fatalf("Use = %q, want %q", command.Use, "inspect-hls")
	}
	timeout, err := command.Flags().GetDuration("timeout")
	if err != nil {
		t.Fatalf("timeout flag: %v", err)
	}
	if timeout != 30*time.Second {
		t.Errorf("default timeout = %v, want 30s", timeout)
	}
}
