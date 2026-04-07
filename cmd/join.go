package cmd

import (
	"fmt"
	"time"

	"github.com/ashutoshsinghai/termshare/internal/client"
	"github.com/ashutoshsinghai/termshare/internal/discovery"
	"github.com/manifoldco/promptui"
	"github.com/spf13/cobra"
)

var joinCmd = &cobra.Command{
	Use:   "join [host:port]",
	Short: "Join a terminal session",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runJoin,
}

func init() {
	joinCmd.Flags().StringP("code", "c", "", "Join code provided by the host")
}

func runJoin(cmd *cobra.Command, args []string) error {
	var addr, code string

	if len(args) == 1 {
		// Direct connect: termshare join 192.168.1.5:4321 -c CODE
		addr = args[0]
		code, _ = cmd.Flags().GetString("code")
		if code == "" {
			code = promptCode()
		}
	} else {
		// Interactive mode: discover → dropdown → code prompt
		var err error
		addr, code, err = interactiveSelect()
		if err != nil {
			return err
		}
	}

	fmt.Printf("Connecting to %s...\n", addr)
	return client.Connect(addr, code)
}

func interactiveSelect() (addr, code string, err error) {
	fmt.Println("Scanning for sessions (3s)...")
	sessions, err := discovery.Discover(3 * time.Second)
	if err != nil {
		return "", "", fmt.Errorf("discovery failed: %w", err)
	}
	if len(sessions) == 0 {
		return "", "", fmt.Errorf("no sessions found on LAN — ask the host for their IP and use: termshare join <ip:port>")
	}

	// Build dropdown items
	items := make([]string, len(sessions))
	addrs := make([]string, len(sessions))
	for i, s := range sessions {
		items[i] = fmt.Sprintf("%s  (%s:%d)", s.Hostname, s.Addr, s.Port)
		addrs[i] = fmt.Sprintf("%s:%d", s.Addr, s.Port)
	}

	prompt := promptui.Select{
		Label: "Select a session",
		Items: items,
	}
	idx, _, err := prompt.Run()
	if err != nil {
		return "", "", fmt.Errorf("cancelled")
	}

	addr = addrs[idx]
	code = promptCode()
	return addr, code, nil
}

func promptCode() string {
	prompt := promptui.Prompt{
		Label: "Join code",
		Mask:  0,
	}
	code, _ := prompt.Run()
	return code
}
