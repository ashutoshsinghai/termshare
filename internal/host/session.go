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

	// Accept loop — validates join code then queues approval request
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
				// Tell client to wait
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

	// Approval loop — runs in main goroutine, reads stdin
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
				fmt.Printf("[+] Connected: %s\n", req.from)
				go runSession(req.conn, req.from)
			} else {
				req.approve <- false
				fmt.Printf("[-] Rejected: %s\n", req.from)
			}
		}
	}
}

func runSession(conn net.Conn, from string) {
	defer func() {
		conn.Close()
		fmt.Printf("[-] Disconnected: %s\n", from)
	}()

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
	defer ptmx.Close()
	defer cmd.Process.Kill()

	var wg sync.WaitGroup

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
