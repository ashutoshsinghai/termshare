package host

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/ashutoshsinghai/termshare/internal/protocol"
)

type approvalRequest struct {
	conn    net.Conn
	from    string
	approve chan bool
}

// Start listens for incoming connections and handles approvals interactively.
func Start(ctx context.Context, port int, code string, readOnly bool) error {
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
				runSession(req.conn, req.from, readOnly)
				fmt.Printf("[+] Waiting for next connection... (Ctrl+C to stop)\n")
			} else {
				req.approve <- false
				fmt.Printf("[-] Rejected: %s\n", req.from)
			}
		}
	}
}
