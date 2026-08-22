package main

import (
	"fmt"
	"os"

	"github.com/w1977-0/media-archiver/cli/internal/app"
)

func main() {
	if err := app.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
