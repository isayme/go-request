package request

import (
	"errors"
	"net"
	"strings"
	"syscall"
)

func shouldRetry(err error) bool {
	// reset by peer
	if errors.Is(err, syscall.ECONNABORTED) {
		return true
	}

	// pipe broken
	if errors.Is(err, syscall.EPIPE) {
		return true
	}

	switch v := err.(type) {
	case net.Error:
		if v.Temporary() {
			return true
		}
	}

	if strings.LastIndex(err.Error(), "broken pipe") >= 0 {
		return true
	}

	if strings.LastIndex(err.Error(), "connection reset by peer") >= 0 {
		return true
	}

	return false
}
