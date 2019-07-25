package metrics

import (
	"strings"
)

func TrimPort(addr string) string {
	if n := strings.IndexByte(addr, ':'); n > 0 {
		addr = addr[:n]
	}

	return addr
}
