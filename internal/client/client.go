package client

import (
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/ashutoshsinghai/termshare/internal/protocol"
	"golang.org/x/term"
)

// Connect dials the host and starts an interactive terminal session.
func Connect(addr string, code string) error {
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer conn.Close()

	// Send join code
	if err := protocol.WriteMessage(conn, protocol.MsgAuth, []byte(code)); err != nil {
		return fmt.Errorf("failed to send auth: %w", err)
	}

	// Wait for auth response
	msgType, payload, err := protocol.ReadMessage(conn)
	if err != nil {
		return fmt.Errorf("auth response error: %w", err)
	}
	if msgType == protocol.MsgAuthFail {
		return fmt.Errorf("authentication failed: %s", string(payload))
	}

	fmt.Println("Connected. Press Ctrl+C to exit.")

	// Put local terminal in raw mode
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	// Watch for terminal resize signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGWINCH)
	sendResize(conn, fd)

	done := make(chan struct{})

	// Relay resize events
	go func() {
		for {
			select {
			case <-sigCh:
				sendResize(conn, fd)
			case <-done:
				return
			}
		}
	}()

	// stdin → server
	go func() {
		defer close(done)
		buf := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				if writeErr := protocol.WriteMessage(conn, protocol.MsgInput, buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// server → stdout
	for {
		msgType, payload, err := protocol.ReadMessage(conn)
		if err != nil {
			select {
			case <-done:
			default:
			}
			return nil
		}
		if msgType == protocol.MsgOutput {
			os.Stdout.Write(payload)
		}
	}
}

func sendResize(conn net.Conn, fd int) {
	cols, rows, err := term.GetSize(fd)
	if err != nil {
		return
	}
	payload := protocol.EncodeResize(uint16(rows), uint16(cols))
	protocol.WriteMessage(conn, protocol.MsgResize, payload)
}
