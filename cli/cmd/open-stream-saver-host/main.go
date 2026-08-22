package main

import (
	"context"
	"fmt"
	"os"

	"github.com/w1977-0/media-archiver/cli/internal/native"
)

func main() {
	if err := native.Serve(context.Background(), os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "open-stream-saver native host:", err)
		os.Exit(1)
	}
}
