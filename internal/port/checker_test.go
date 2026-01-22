package port

import (
	"net"
	"testing"
)

func TestTCPChecker(t *testing.T) {
	checker := TCPChecker{}

	t.Run("implements Checker interface", func(t *testing.T) {
		var _ Checker = checker
	})

	t.Run("returns true for unused port", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to get free port: %v", err)
		}
		port := listener.Addr().(*net.TCPAddr).Port
		listener.Close()

		if !checker.IsFree(port) {
			t.Errorf("TCPChecker.IsFree(%d) = false, want true", port)
		}
	})

	t.Run("returns false for port in use", func(t *testing.T) {
		listener, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("failed to bind port: %v", err)
		}
		defer listener.Close()

		port := listener.Addr().(*net.TCPAddr).Port

		if checker.IsFree(port) {
			t.Errorf("TCPChecker.IsFree(%d) = true, want false", port)
		}
	})
}
