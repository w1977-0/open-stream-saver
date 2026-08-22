package app

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/w1977-0/open-stream-saver/cli/internal/direct"
	"github.com/w1977-0/open-stream-saver/cli/internal/hls"
	"github.com/w1977-0/open-stream-saver/cli/internal/safety"
)

var Version = "dev"

type downloadFlags struct {
	url         string
	output      string
	workers     int
	timeout     time.Duration
	acknowledge bool
}

func Execute() error {
	root := &cobra.Command{
		Use:           "open-stream-saver",
		Short:         "Save authorized public media to your local computer",
		Long:          "Open Stream Saver downloads a single public direct file or completed, unencrypted HLS media playlist only after you explicitly acknowledge your rights to save it.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.Version = Version
	root.AddCommand(newDownloadCommand())
	return root.Execute()
}

func newDownloadCommand() *cobra.Command {
	flags := downloadFlags{}
	command := &cobra.Command{
		Use:   "download",
		Short: "Save one authorized public URL",
		RunE: func(command *cobra.Command, _ []string) error {
			if !flags.acknowledge {
				return fmt.Errorf("refusing to start: pass --acknowledge-rights only when you have permission to save this content")
			}
			if flags.workers < 1 || flags.workers > 16 {
				return fmt.Errorf("workers must be between 1 and 16")
			}
			parsed, err := safety.ValidatePublicURL(flags.url)
			if err != nil {
				return err
			}
			if flags.output == "" {
				flags.output = defaultOutput(parsed)
			}
			output, err := filepath.Abs(flags.output)
			if err != nil {
				return err
			}
			context, cancel := context.WithTimeout(command.Context(), flags.timeout)
			defer cancel()
			client := &http.Client{
				Timeout: flags.timeout,
				CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
					return http.ErrUseLastResponse
				},
			}

			if strings.HasSuffix(strings.ToLower(parsed.Path), ".m3u8") {
				if filepath.Ext(output) == "" {
					output += ".mp4"
				}
				return hls.DownloadMediaPlaylist(context, parsed.String(), hls.Options{
					Client: client, Workers: flags.workers, Output: output, Writer: os.Stderr,
				})
			}
			return direct.Download(context, parsed.String(), direct.Options{
				Client: client, Workers: flags.workers, Output: output, Writer: os.Stderr,
			})
		},
	}
	command.Flags().StringVarP(&flags.url, "url", "u", "", "Public HTTP(S) direct-file or .m3u8 media-playlist URL")
	command.Flags().StringVarP(&flags.output, "output", "o", "", "Local output file path")
	command.Flags().IntVarP(&flags.workers, "workers", "w", 4, "Concurrent direct-file ranges or HLS segments (1-16)")
	command.Flags().DurationVar(&flags.timeout, "timeout", 30*time.Minute, "Overall network operation timeout")
	command.Flags().BoolVar(&flags.acknowledge, "acknowledge-rights", false, "Confirm that you have the right to save the content")
	_ = command.MarkFlagRequired("url")
	return command
}

func defaultOutput(parsed *url.URL) string {
	name := path.Base(parsed.Path)
	if name == "." || name == "/" || name == "" || strings.HasSuffix(strings.ToLower(name), ".m3u8") {
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
