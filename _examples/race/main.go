package main

import (
	"fmt"
	"time"

	"github.com/martinohmann/promise"
)

func main() {
	p1 := promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		time.Sleep(10 * time.Millisecond)
		resolve("Promise 1")
	})

	p2 := promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		resolve("Promise 2")
	})

	p3 := promise.Resolve("Promise 3")

	p := promise.Race(p1, p2, p3)

	val, err := p.Await()
	if err != nil {
		panic(err)
	}

	fmt.Printf("%s finished first\n", val)
}
