// This example demonstrates concurrent DNS lookups using promises. Check out
// the pooled example (../pooled/main.go) to see how to control the amount of
// concurrency. The code below might DDoS small home routers if provided with
// thousands of hosts at the same time. For cases like these a pooled approach
// is more robust.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/martinohmann/promise"
	dns "github.com/martinohmann/promise/_examples/dns-lookup"
)

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

	resolver := dns.NewResolver(*resolverAddr)

	for i, host := range hosts {
		promises[i] = resolver.Lookup(host)
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
