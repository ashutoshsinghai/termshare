package cmd

import (
	"fmt"
	"time"

	"github.com/ashutoshsinghai/termshare/internal/discovery"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List available termshare sessions on the LAN",
	RunE:  runList,
}

func runList(cmd *cobra.Command, args []string) error {
	fmt.Println("Scanning for sessions (3s)...")
	sessions, err := discovery.Discover(3 * time.Second)
	if err != nil {
		return err
	}
	if len(sessions) == 0 {
		fmt.Println("No sessions found.")
		return nil
	}
	fmt.Println("\nAvailable sessions:")
	for _, s := range sessions {
		fmt.Printf("  %s  →  %s:%d\n", s.Hostname, s.Addr, s.Port)
	}
	return nil
}
