// Package integrity computes local output digests. A digest is an integrity
// report for the resulting local file; it is not a claim about media rights or
// the authenticity of a remote source.
package integrity

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
)

// SHA256File returns the lowercase SHA-256 hex digest of one local file.
func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// Report writes a stable, shell-friendly digest line when a writer is supplied.
func Report(writer io.Writer, path string) error {
	if writer == nil {
		return nil
	}
	digest, err := SHA256File(path)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(writer, "SHA256 %s  %s\n", digest, path)
	return err
}
