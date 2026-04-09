//go:build windows

package host

import (
	"fmt"
	"net"
	"os"
	"sync"

	gopty "github.com/aymanbagabas/go-pty"
	"github.com/ashutoshsinghai/termshare/internal/protocol"
	"golang.org/x/term"
)

func runSession(conn net.Conn, from string, readOnly bool) {
	defer conn.Close()

	if readOnly {
		if err := protocol.WriteMessage(conn, protocol.MsgReadOnly, nil); err != nil {
			return
		}
	}
	if err := protocol.WriteMessage(conn, protocol.MsgAuthOK, nil); err != nil {
		return
	}

	shell := os.Getenv("COMSPEC")
	if shell == "" {
		shell = "cmd.exe"
	}

	fd := int(os.Stdin.Fd())
	cols, rows, err := term.GetSize(fd)
	if err != nil {
		cols, rows = 80, 24
	}

	ptmx, err := gopty.New()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create pty: %v\n", err)
		return
	}
	defer ptmx.Close()

	if err := ptmx.Resize(cols, rows); err != nil {
		fmt.Fprintf(os.Stderr, "failed to resize pty: %v\n", err)
		return
	}

	cmd := ptmx.Command(shell)
	cmd.Env = append(os.Environ(), "TERM=xterm-256color")
	if err := cmd.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start pty: %v\n", err)
		return
	}
	defer cmd.Process.Kill()

	oldState, err := term.MakeRaw(fd)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to set raw mode: %v\n", err)
		return
	}
	defer term.Restore(fd, oldState)

	done := make(chan struct{})
	var once sync.Once
	finish := func() { once.Do(func() { close(done) }) }

	// PTY output → host stdout + client
	go func() {
		defer finish()
		buf := make([]byte, 4096)
		for {
			n, err := ptmx.Read(buf)
			if n > 0 {
				os.Stdout.Write(buf[:n])
				protocol.WriteMessage(conn, protocol.MsgOutput, buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()

	// Host stdin → PTY
	go func() {
		buf := make([]byte, 256)
		for {
			n, err := os.Stdin.Read(buf)
			if n > 0 {
				if _, werr := ptmx.Write(buf[:n]); werr != nil {
					return
				}
			}
			if err != nil {
				return
			}
		}
	}()

	// Client messages → PTY (input blocked in read-only mode)
	go func() {
		defer finish()
		for {
			msgType, payload, err := protocol.ReadMessage(conn)
			if err != nil {
				return
			}
			switch msgType {
			case protocol.MsgInput:
				if !readOnly {
					ptmx.Write(payload)
				}
			case protocol.MsgResize:
				if len(payload) == 4 {
					resize := protocol.DecodeResize(payload)
					ptmx.Resize(int(resize.Cols), int(resize.Rows))
				}
			}
		}
	}()

	<-done
	fmt.Printf("\n[-] Disconnected: %s\n", from)
}
