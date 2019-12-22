// pi approximation algorithm taken from https://golang.org/doc/play/pi.go

package main

import (
	"fmt"
	"math"

	"github.com/martinohmann/promise"
)

// pi launches n goroutines to compute an
// approximation of pi.
func pi(n int) float64 {
	ch := make(chan float64)
	for k := 0; k <= n; k++ {
		go term(ch, float64(k))
	}
	f := 0.0
	for k := 0; k <= n; k++ {
		f += <-ch
	}
	return f
}

func term(ch chan float64, k float64) {
	ch <- 4 * math.Pow(-1, k) / (2*k + 1)
}

func piPromise(n int) promise.Promise {
	return promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		resolve(pi(n))
	}).Then(func(val promise.Value) promise.Value {
		fmt.Printf("approximation using %d goroutines finished: %f\n", n, val)
		return val
	})
}

func main() {
	p := promise.All(
		piPromise(1),
		piPromise(10),
		piPromise(100),
		piPromise(1000),
		piPromise(10000),
		piPromise(100000),
	)

	val, err := p.Await()
	if err != nil {
		panic(err)
	}

	fmt.Println("approximated pi values:")
	for _, v := range val.([]promise.Value) {
		fmt.Printf("* %f\n", v)
	}
}
