package port

import (
	"fmt"
	"net"
)

// Checker checks if a port is free for binding.
type Checker interface {
	IsFree(port int) bool
}

// TCPChecker checks port availability by attempting to bind.
type TCPChecker struct{}

func (TCPChecker) IsFree(port int) bool {
	listener, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = listener.Close()
	return true
}
