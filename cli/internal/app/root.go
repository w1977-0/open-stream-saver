package app

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/w1977-0/open-stream-saver/cli/internal/dash"
	"github.com/w1977-0/open-stream-saver/cli/internal/direct"
	"github.com/w1977-0/open-stream-saver/cli/internal/hls"
	"github.com/w1977-0/open-stream-saver/cli/internal/safety"
)

var Version = "dev"

const maxWorkers = 32

type downloadFlags struct {
	url         string
	output      string
	workers     int
	variant     int
	timeout     time.Duration
	acknowledge bool
}

func Execute() error {
	root := &cobra.Command{
		Use:           "open-stream-saver",
		Short:         "Save authorized public media to your local computer",
		Long:          "Open Stream Saver downloads one authorized public direct file, completed unencrypted HLS playlist, or static unencrypted DASH presentation. It never accepts cookies, credentials, tokens, DRM keys, or custom authentication headers.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.Version = Version
	root.AddCommand(newDownloadCommand(), newInspectHLSCommand())
	return root.Execute()
}

func newDownloadCommand() *cobra.Command {
	flags := downloadFlags{variant: -1}
	command := &cobra.Command{
		Use:   "download",
		Short: "Save one authorized public URL",
		RunE: func(command *cobra.Command, _ []string) error {
			if !flags.acknowledge {
				return fmt.Errorf("refusing to start: pass --acknowledge-rights only when you have permission to save this content")
			}
			return DownloadAuthorized(command.Context(), flags.url, flags.output, flags.workers, flags.variant, flags.timeout, os.Stderr)
		},
	}
	command.Flags().StringVarP(&flags.url, "url", "u", "", "Public HTTP(S) direct-file, .m3u8, or .mpd URL")
	command.Flags().StringVarP(&flags.output, "output", "o", "", "New local output file path (refuses to overwrite)")
	command.Flags().IntVarP(&flags.workers, "workers", "w", 4, "Concurrent direct-file ranges or media segments (1-32)")
	command.Flags().IntVar(&flags.variant, "variant", -1, "HLS master variant index; -1 selects the highest advertised bandwidth")
	command.Flags().DurationVar(&flags.timeout, "timeout", 30*time.Minute, "Overall network operation timeout")
	command.Flags().BoolVar(&flags.acknowledge, "acknowledge-rights", false, "Confirm that you have the right to save the content")
	_ = command.MarkFlagRequired("url")
	return command
}

func newInspectHLSCommand() *cobra.Command {
	var rawURL string
	var timeout time.Duration
	command := &cobra.Command{
		Use:   "inspect-hls",
		Short: "List safe variants from one public HLS master playlist",
		RunE: func(command *cobra.Command, _ []string) error {
			parsed, err := safety.ValidatePublicURL(rawURL)
			if err != nil {
				return err
			}
			ctx, cancel := context.WithTimeout(command.Context(), timeout)
			defer cancel()
			client := newClient(timeout)
			variants, err := hls.Inspect(ctx, parsed.String(), client)
			if err != nil {
				return err
			}
			for _, item := range variants {
				_, err := fmt.Fprintf(command.OutOrStdout(), "%d\t%s\t%d\t%s\n", item.Index, item.Resolution, item.Bandwidth, item.Codecs)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	command.Flags().StringVarP(&rawURL, "url", "u", "", "Public HTTP(S) HLS master playlist URL")
	command.Flags().DurationVar(&timeout, "timeout", 30*time.Second, "Manifest request timeout")
	_ = command.MarkFlagRequired("url")
	return command
}

// DownloadAuthorized is shared by the interactive CLI and the local native
// host. Its caller must obtain an explicit acknowledgement from the user; it
// accepts only a URL and local output options, never browser identity data.
func DownloadAuthorized(parent context.Context, rawURL, output string, workers, variant int, timeout time.Duration, writer io.Writer) error {
	if workers < 1 || workers > maxWorkers {
		return fmt.Errorf("workers must be between 1 and %d", maxWorkers)
	}
	if timeout <= 0 {
		return fmt.Errorf("timeout must be positive")
	}
	parsed, err := safety.ValidatePublicURL(rawURL)
	if err != nil {
		return err
	}
	if output == "" {
		output = defaultOutput(parsed)
	}
	absoluteOutput, err := filepath.Abs(output)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	defer cancel()
	client := newClient(timeout)

	switch strings.ToLower(path.Ext(parsed.Path)) {
	case ".m3u8":
		if filepath.Ext(absoluteOutput) == "" {
			absoluteOutput += ".mp4"
		}
		return hls.DownloadMediaPlaylist(ctx, parsed.String(), hls.Options{
			Client: client, Workers: workers, Variant: variant, Output: absoluteOutput, Writer: writer,
		})
	case ".mpd":
		if filepath.Ext(absoluteOutput) == "" {
			absoluteOutput += ".mp4"
		}
		return dash.Download(ctx, parsed.String(), dash.Options{
			Client: client, Workers: workers, Output: absoluteOutput, Writer: writer,
		})
	default:
		return direct.Download(ctx, parsed.String(), direct.Options{
			Client: client, Workers: workers, Output: absoluteOutput, Writer: writer,
		})
	}
}

func newClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func defaultOutput(parsed *url.URL) string {
	name := path.Base(parsed.Path)
	if name == "." || name == "/" || name == "" || strings.HasSuffix(strings.ToLower(name), ".m3u8") || strings.HasSuffix(strings.ToLower(name), ".mpd") {
		return "authorized-download.mp4"
	}
	name = strings.Map(func(r rune) rune {
		if r == '/' || r == '\\' || r == 0 {
			return '_'
		}
		return r
	}, name)
	return name
}
