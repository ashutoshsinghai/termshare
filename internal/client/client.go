package client

import (
	"fmt"
	"net"
	"os"

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

	// Wait for auth response (may get MsgPending first)
	for {
		msgType, payload, err := protocol.ReadMessage(conn)
		if err != nil {
			return fmt.Errorf("auth response error: %w", err)
		}
		if msgType == protocol.MsgPending {
			fmt.Println("Waiting for host approval...")
			continue
		}
		if msgType == protocol.MsgAuthFail {
			return fmt.Errorf("connection rejected: %s", string(payload))
		}
		break // MsgAuthOK
	}

	fmt.Println("Connected. Press Ctrl+\\ to disconnect.")

	// Put local terminal in raw mode
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		return fmt.Errorf("failed to set raw mode: %w", err)
	}
	defer term.Restore(fd, oldState)

	sendResize(conn, fd)

	done := make(chan struct{})

	go watchResize(conn, fd, done)

	// stdin → server (Ctrl+\ disconnects locally)
	go func() {
		defer close(done)
		buf := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				// 0x1c is Ctrl+\, intercept as local disconnect
				if n == 1 && buf[0] == 0x1c {
					conn.Close()
					return
				}
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
