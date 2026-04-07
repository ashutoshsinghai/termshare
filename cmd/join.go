package cmd

import (
	"fmt"

	"github.com/ashutoshsinghai/termshare/internal/client"
	"github.com/spf13/cobra"
)

var joinCmd = &cobra.Command{
	Use:   "join <host:port>",
	Short: "Join a terminal session",
	Args:  cobra.ExactArgs(1),
	RunE:  runJoin,
}

func init() {
	joinCmd.Flags().StringP("code", "c", "", "Join code provided by the host (required)")
	joinCmd.MarkFlagRequired("code")
}

func runJoin(cmd *cobra.Command, args []string) error {
	addr := args[0]
	code, _ := cmd.Flags().GetString("code")

	fmt.Printf("Connecting to %s...\n", addr)
	return client.Connect(addr, code)
}
