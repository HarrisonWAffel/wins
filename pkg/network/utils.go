package network

import (
	"k8s.io/client-go/rest"
	"net"
	"time"
)

// EnsureHostIsReachable makes a number of network requests against
// a host to ensure it is reachable.
func EnsureHostIsReachable(cfg *rest.Config, retryLimit int) bool {

	timeout := 3 * time.Second
	for i := 0; i < retryLimit; i++ {
		_, err := net.DialTimeout("tcp", cfg.Host, timeout)
		if err == nil {
			return true
		}
	}

	return false
}
