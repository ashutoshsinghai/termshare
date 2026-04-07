//go:build !windows

package client

import (
	"net"
	"os"
	"os/signal"
	"syscall"
)

func watchResize(conn net.Conn, fd int, done <-chan struct{}) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	for {
		select {
		case <-sigCh:
			sendResize(conn, fd)
		case <-done:
			return
		}
	}
}
