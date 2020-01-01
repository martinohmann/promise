package dns

import (
	"context"
	"net"
	"strings"
	"time"

	"github.com/martinohmann/promise"
)

type LookupResult struct {
	Host    string
	IPAddrs []net.IPAddr
}

func (r LookupResult) String() string {
	var sb strings.Builder

	sb.WriteString(r.Host)
	sb.WriteString(":\n")

	for _, ip := range r.IPAddrs {
		if ip.IP.To4() != nil {
			sb.WriteString("  A    ")
		} else {
			sb.WriteString("  AAAA ")
		}

		sb.WriteString(ip.IP.String())
		sb.WriteByte('\n')
	}

	return sb.String()
}

type Resolver struct {
	*net.Resolver
}

func NewResolver(addr string) *Resolver {
	if addr == "" {
		return &Resolver{net.DefaultResolver}
	}

	if !strings.Contains(addr, ":") {
		addr += ":53"
	}

	return &Resolver{
		&net.Resolver{
			PreferGo: true,
			Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
				d := net.Dialer{
					Timeout: time.Millisecond * 5000,
				}
				return d.DialContext(ctx, "udp", addr)
			},
		},
	}
}

func (r *Resolver) Lookup(host string) promise.Promise {
	return promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		ips, err := r.Resolver.LookupIPAddr(context.Background(), host)
		if err != nil {
			reject(err)
			return
		}

		resolve(LookupResult{Host: host, IPAddrs: ips})
	})
}
