package main

import (
	"context"
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/martinohmann/promise"
)

type LookupResult struct {
	Host    string
	IPAddrs []net.IPAddr
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

var resolverAddr = flag.String("resolver", "", "Address of the DNS resolver to use. If omitted, the default resolver is used. Format: <ipv4-address>[:<port>]")

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s [flags] <host> [<host>...]\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	hosts := flag.Args()

	if len(hosts) < 1 {
		flag.Usage()
		os.Exit(1)
	}

	promises := make([]promise.Promise, len(hosts))

	resolver := NewResolver(*resolverAddr)

	for i, host := range hosts {
		promises[i] = resolver.Lookup(host).Then(func(val promise.Value) promise.Value {
			var sb strings.Builder

			result := val.(LookupResult)

			sb.WriteString(result.Host)
			sb.WriteString(":\n")

			for _, ip := range result.IPAddrs {
				if ip.IP.To4() != nil {
					sb.WriteString("  A    ")
				} else {
					sb.WriteString("  AAAA ")
				}

				sb.WriteString(ip.IP.String())
				sb.WriteByte('\n')
			}

			return sb.String()
		})
	}

	results, _ := promise.AllSettled(promises...).Await()

	var ret int

	for _, result := range results.([]promise.Result) {
		if result.Err != nil {
			ret = 1
			fmt.Fprintf(os.Stderr, "%v\n\n", result.Err)
		} else {
			fmt.Fprintln(os.Stdout, result.Value)
		}
	}

	os.Exit(ret)
}
