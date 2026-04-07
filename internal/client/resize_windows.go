//go:build windows

package client

import "net"

func watchResize(conn net.Conn, fd int, done <-chan struct{}) {
	// SIGWINCH is not available on Windows; resize events are not supported
	<-done
}
