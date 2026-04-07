package host

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/ashutoshsinghai/termshare/internal/protocol"
	"github.com/creack/pty"
	"golang.org/x/term"
)

type approvalRequest struct {
	conn    net.Conn
	from    string
	approve chan bool
}

// Start listens for incoming connections and handles approvals interactively.
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

	approvalCh := make(chan approvalRequest, 1)

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
				default:
					fmt.Fprintf(os.Stderr, "accept error: %v\n", err)
				}
				return
			}
			go func(c net.Conn) {
				from := c.RemoteAddr().String()
				msgType, payload, err := protocol.ReadMessage(c)
				if err != nil || msgType != protocol.MsgAuth || string(payload) != code {
					protocol.WriteMessage(c, protocol.MsgAuthFail, []byte("invalid join code"))
					c.Close()
					return
				}
				protocol.WriteMessage(c, protocol.MsgPending, nil)
				approve := make(chan bool, 1)
				approvalCh <- approvalRequest{conn: c, from: from, approve: approve}
				if !<-approve {
					protocol.WriteMessage(c, protocol.MsgAuthFail, []byte("host rejected the connection"))
					c.Close()
				}
			}(conn)
		}
	}()

	stdin := bufio.NewReader(os.Stdin)
	for {
		select {
		case <-ctx.Done():
			return nil
		case req := <-approvalCh:
			fmt.Printf("\n[?] Connection request from %s — approve? [y/N]: ", req.from)
			line, _ := stdin.ReadString('\n')
			line = strings.TrimSpace(strings.ToLower(line))
			if line == "y" {
				req.approve <- true
				fmt.Printf("[+] Session started with %s\n", req.from)
				runSession(req.conn, req.from)
				fmt.Printf("[+] Waiting for next connection... (Ctrl+C to stop)\n")
			} else {
				req.approve <- false
				fmt.Printf("[-] Rejected: %s\n", req.from)
			}
		}
	}
}

func runSession(conn net.Conn, from string) {
	defer conn.Close()

	if err := protocol.WriteMessage(conn, protocol.MsgAuthOK, nil); err != nil {
		return
	}

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
	defer cmd.Process.Kill()
	defer ptmx.Close()

	// Set initial PTY size from host terminal
	fd := int(os.Stdin.Fd())
	if cols, rows, err := term.GetSize(fd); err == nil {
		pty.Setsize(ptmx, &pty.Winsize{Rows: uint16(rows), Cols: uint16(cols)})
	}

	// Put host terminal in raw mode for the duration of the session
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

	// Client messages → PTY
	go func() {
		defer finish()
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

	<-done
	fmt.Printf("\n[-] Disconnected: %s\n", from)
}
