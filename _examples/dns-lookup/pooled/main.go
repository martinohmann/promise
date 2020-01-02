// This example uses promise.Pool to orchestrate concurrent DNS lookups.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/martinohmann/promise"
	dns "github.com/martinohmann/promise/_examples/dns-lookup"
)

var resolverAddr = flag.String("resolver", "", "Address of the DNS resolver to use. If omitted, the default resolver is used. Format: <ipv4-address>[:<port>]")
var concurrency = flag.Int("concurrency", 10, "Number of DNS lookups to perform concurrently")

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

	// Context cancellation will stop the promise pool and reject the promise
	// orchestrating it.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Make a channel to pass promise factory funcs to the promise pool.
	fns := make(chan func() promise.Promise)

	// Instead of aborting on the first lookup error we want to continue.
	// Errors will be logged to stderr using the event listener attached below.
	options := promise.PoolOptions{ContinueOnError: true}

	pool := promise.NewPool(*concurrency, fns, options)

	// We add an event listener to the pool so that we can print the values of
	// all fulfilled promises and all errors that occur along the way.
	pool.AddEventListener(&promise.PoolEventListener{
		OnFulfilled: func(val promise.Value) {
			fmt.Fprintln(os.Stdout, val)
		},
		OnRejected: func(err error) {
			fmt.Fprintln(os.Stderr, err)
		},
	})

	go func() {
		// The fns channel must be closed to signal the pool that we are done
		// sending more promise factory funcs. Otherwise waiting for the pool
		// promise to resolve will block forever.
		defer close(fns)

		resolver := dns.NewResolver(*resolverAddr)

		for _, host := range hosts {
			host := host

			fn := func() promise.Promise {
				return resolver.Lookup(host)
			}

			select {
			case <-ctx.Done():
				return
			case fns <- fn:
			}
		}
	}()

	go handleSignals(cancel)

	// Start consuming the promise factories from the fns channel. This returns
	// a promise which we can await. The promise resolves after fns is closed
	// and all in flight promises resolved. If ctx is cancelled, the promise is
	// rejected and consuming the fns channel is stopped. Without the
	// ContinueOnError PoolOption we set above, the rejection of any pooled
	// promise also causes the promise to be immediately be rejected.
	_, err := pool.Run(ctx).Await()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func handleSignals(cancel func()) {
	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGTERM, os.Interrupt)
	<-signals
	fmt.Println("received signal")
	cancel()
	<-signals
	fmt.Println("forced exit")
	os.Exit(1)
}
