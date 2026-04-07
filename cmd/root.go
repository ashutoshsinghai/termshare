package cmd

import "github.com/spf13/cobra"

var rootCmd = &cobra.Command{
	Use:   "termshare",
	Short: "Share terminal sessions over LAN instantly",
	Long:  "termshare lets you host and join terminal sessions across devices on the same network — no config required.",
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.AddCommand(hostCmd)
	rootCmd.AddCommand(listCmd)
	rootCmd.AddCommand(joinCmd)
}
