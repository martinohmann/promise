package main

import (
	"errors"
	"flag"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"strconv"

	"github.com/martinohmann/promise"
)

func init() {
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "usage: %s <raw>\n", filepath.Base(os.Args[0]))
		flag.PrintDefaults()
	}
}

func parseBigInt(raw string) promise.Promise {
	return promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		n := new(big.Int)
		n, ok := n.SetString(raw, 10)
		if !ok {
			reject(errors.New("big.Int.SetString: error"))
			return
		}

		resolve(n)
	})
}

func parseFloat(raw string) promise.Promise {
	return promise.Resolve(raw).Then(func(val promise.Value) promise.Value {
		num, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return err
		}

		return num
	})
}

// This example is somewhat convoluted and not very helpful in production code.
// It's just here to demonstrate promise nesting. It tries to parse a raw cli
// arg into uint64, int64, big.Int, bool or float64, whatever succeeds without
// error first. It also demonstrates how promises can be created in different
// ways and recovered.
func main() {
	flag.Parse()

	args := flag.Args()

	if len(args) != 1 {
		flag.Usage()
		os.Exit(1)
	}

	raw := args[0]

	p := promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		num, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			reject(err)
			return
		}

		resolve(num)
	}).Catch(func(err error) promise.Value {
		return promise.Reject(err).Catch(func(err error) promise.Value {
			num, err := strconv.ParseInt(raw, 10, 64)
			if err != nil {
				return err
			}
			return num
		}).Catch(func(err error) promise.Value {
			return parseBigInt(raw).Catch(func(err error) promise.Value {
				b, err := strconv.ParseBool(raw)
				if err != nil {
					return err
				}

				return b
			})
		})
	}).Catch(func(err error) promise.Value {
		return parseFloat(raw)
	}).Catch(func(err error) promise.Value {
		return promise.Resolve(raw)
	}).Then(func(val promise.Value) promise.Value {
		return fmt.Sprintf("%T: %v", val, val)
	})

	val, err := p.Await()

	fmt.Printf("Promise value: %v, error: %v\n", val, err)
}
