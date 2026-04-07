package cmd

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"os/signal"
	"syscall"

	"github.com/ashutoshsinghai/termshare/internal/discovery"
	"github.com/ashutoshsinghai/termshare/internal/host"
	"github.com/spf13/cobra"
)

const defaultPort = 4321

var hostCmd = &cobra.Command{
	Use:   "host",
	Short: "Host a terminal session",
	RunE:  runHost,
}

func runHost(cmd *cobra.Command, args []string) error {
	code, err := generateCode()
	if err != nil {
		return fmt.Errorf("failed to generate join code: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		fmt.Println("\nShutting down...")
		cancel()
	}()

	fmt.Println("termshare — hosting session")
	fmt.Printf("Join code : %s\n", code)
	fmt.Printf("Port      : %d\n\n", defaultPort)
	fmt.Println("Waiting for a connection... (Ctrl+C to stop)")

	go func() {
		if err := discovery.Advertise(ctx, defaultPort); err != nil {
			fmt.Fprintf(os.Stderr, "mDNS warning: %v\n", err)
		}
	}()

	return host.Start(ctx, defaultPort, code)
}

func generateCode() (string, error) {
	const chars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	code := make([]byte, 6)
	for i := range code {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		code[i] = chars[n.Int64()]
	}
	return string(code), nil
}
