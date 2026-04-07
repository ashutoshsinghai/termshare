package discovery

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/grandcat/zeroconf"
)

const serviceType = "_termshare._tcp"
const domain = "local."

type Session struct {
	Hostname string
	Addr     string
	Port     int
}

// Advertise registers the host session via mDNS. Blocks until ctx is cancelled.
func Advertise(ctx context.Context, port int) error {
	hostname, _ := os.Hostname()
	server, err := zeroconf.Register(hostname, serviceType, domain, port, []string{"termshare"}, nil)
	if err != nil {
		return fmt.Errorf("mDNS registration failed: %w", err)
	}
	<-ctx.Done()
	server.Shutdown()
	return nil
}

// Discover browses for active termshare sessions on the LAN.
func Discover(timeout time.Duration) ([]Session, error) {
	resolver, err := zeroconf.NewResolver(nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create resolver: %w", err)
	}

	entries := make(chan *zeroconf.ServiceEntry)
	var sessions []Session

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	go func() {
		for entry := range entries {
			var addr string
			if len(entry.AddrIPv4) > 0 {
				addr = entry.AddrIPv4[0].String()
			} else if len(entry.AddrIPv6) > 0 {
				addr = fmt.Sprintf("[%s]", entry.AddrIPv6[0].String())
			}
			if addr != "" {
				sessions = append(sessions, Session{
					Hostname: entry.HostName,
					Addr:     addr,
					Port:     entry.Port,
				})
			}
		}
	}()

	if err := resolver.Browse(ctx, serviceType, domain, entries); err != nil {
		return nil, fmt.Errorf("browse failed: %w", err)
	}

	<-ctx.Done()
	return sessions, nil
}
