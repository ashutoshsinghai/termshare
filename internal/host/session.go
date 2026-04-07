package host

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"

	"github.com/ashutoshsinghai/termshare/internal/protocol"
	"github.com/creack/pty"
)

// Start listens for incoming connections and handles each client in a goroutine.
func Start(ctx context.Context, port int, code string) error {
	ln, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		return fmt.Errorf("failed to listen on port %d: %w", port, err)
	}
	defer ln.Close()

	go func() {
		<-ctx.Done()
		ln.Close()
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			select {
			case <-ctx.Done():
				return nil
			default:
				return fmt.Errorf("accept error: %w", err)
			}
		}
		go handleClient(conn, code)
	}
}

func handleClient(conn net.Conn, code string) {
	defer conn.Close()

	// Expect auth message first
	msgType, payload, err := protocol.ReadMessage(conn)
	if err != nil || msgType != protocol.MsgAuth || string(payload) != code {
		protocol.WriteMessage(conn, protocol.MsgAuthFail, []byte("invalid join code"))
		return
	}
	if err := protocol.WriteMessage(conn, protocol.MsgAuthOK, nil); err != nil {
		return
	}

	// Start host shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	cmd := exec.Command(shell)
	ptmx, err := pty.Start(cmd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to start pty: %v\n", err)
		return
	}
	defer ptmx.Close()
	defer cmd.Process.Kill()

	var wg sync.WaitGroup

	// Stream PTY output → client
	wg.Add(1)
	go func() {
		defer wg.Done()
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				if writeErr := protocol.WriteMessage(conn, protocol.MsgOutput, buf[:n]); writeErr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Stream client messages → PTY
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			msgType, payload, err := protocol.ReadMessage(conn)
			if err != nil {
				return
			}
			switch msgType {
			case protocol.MsgInput:
				ptmx.Write(payload)
			case protocol.MsgResize:
				if len(payload) == 4 {
					resize := protocol.DecodeResize(payload)
					pty.Setsize(ptmx, &pty.Winsize{
						Rows: resize.Rows,
						Cols: resize.Cols,
					})
				}
			}
		}
	}()

	wg.Wait()
}
