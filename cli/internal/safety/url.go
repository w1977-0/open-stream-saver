package safety

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"strings"
)

const maxURLLength = 4096

// ValidatePublicURL accepts only public HTTP(S) targets. It deliberately does not
// accept credentials, local hostnames, or private/reserved IP addresses.
func ValidatePublicURL(raw string) (*url.URL, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || len(raw) > maxURLLength {
		return nil, fmt.Errorf("URL must be between 1 and %d characters", maxURLLength)
	}

	parsed, err := url.ParseRequestURI(raw)
	if err != nil || parsed.Scheme == "" || parsed.Hostname() == "" {
		return nil, fmt.Errorf("provide a complete public http(s) URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("only http and https URLs are supported")
	}
	if parsed.User != nil {
		return nil, fmt.Errorf("URLs with embedded credentials are not accepted")
	}

	host := strings.TrimSuffix(strings.ToLower(parsed.Hostname()), ".")
	if host == "localhost" || strings.HasSuffix(host, ".local") {
		return nil, fmt.Errorf("local and private network targets are not accepted")
	}

	addresses, err := net.LookupIP(host)
	if err != nil || len(addresses) == 0 {
		return nil, fmt.Errorf("could not resolve the target host")
	}
	for _, address := range addresses {
		parsedAddress, ok := netip.AddrFromSlice(address)
		if !ok || !isPublic(parsedAddress) {
			return nil, fmt.Errorf("local and private network targets are not accepted")
		}
	}
	return parsed, nil
}

func isPublic(address netip.Addr) bool {
	return address.IsValid() && !address.IsPrivate() && !address.IsLoopback() && !address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() && !address.IsMulticast() && !address.IsUnspecified()
}
