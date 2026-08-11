package testing

import (
	"fmt"
	"strconv"
	"strings"
)

// SplitAddr splits a "host:port" service address into its parts.
// Test services are addressed by their compose DNS name, so the host part
// is not necessarily localhost.
func SplitAddr(addr string) (string, int, error) {
	host, portStr, found := strings.Cut(addr, ":")
	if !found || host == "" || portStr == "" {
		return "", 0, fmt.Errorf("invalid address %q, expected host:port", addr)
	}

	port, err := strconv.Atoi(portStr)
	if err != nil {
		return "", 0, fmt.Errorf("invalid port in address %q: %w", addr, err)
	}

	return host, port, nil
}
