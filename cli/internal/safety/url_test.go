package safety

import (
	"net/netip"
	"testing"
)

func TestIsPublicRejectsLocalAndPrivateAddresses(t *testing.T) {
	for _, raw := range []string{"127.0.0.1", "10.0.0.1", "192.168.1.1", "::1"} {
		address := netip.MustParseAddr(raw)
		if isPublic(address) {
			t.Fatalf("expected %s to be rejected", raw)
		}
	}
}

func TestValidatePublicURLRejectsUnsafeInputs(t *testing.T) {
	for _, raw := range []string{
		"http://localhost/video.mp4",
		"http://user:password@example.com/video.mp4",
		"file:///tmp/video.mp4",
	} {
		if _, err := ValidatePublicURL(raw); err == nil {
			t.Fatalf("expected %q to be rejected", raw)
		}
	}
}
