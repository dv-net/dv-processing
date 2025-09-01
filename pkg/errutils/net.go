package errutils

import (
	"errors"
	"net"
	"strings"
	"syscall"
)

// IsNetworkError checks if an error is a network-related error that should trigger a retry
func IsNetworkError(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, net.ErrClosed) {
		return true
	}

	if errors.Is(err, syscall.ECONNRESET) {
		return true
	}

	if errors.Is(err, syscall.ECONNREFUSED) {
		return true
	}

	if errors.Is(err, syscall.ECONNABORTED) {
		return true
	}

	if errors.Is(err, syscall.ETIMEDOUT) {
		return true
	}

	if errors.Is(err, syscall.EHOSTUNREACH) {
		return true
	}

	if errors.Is(err, syscall.ENETUNREACH) {
		return true
	}

	if errors.Is(err, syscall.EPIPE) {
		return true
	}

	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}

	if strings.Contains(err.Error(), "EOF") {
		return true
	}

	return false
}
