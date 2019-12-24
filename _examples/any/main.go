package main

import (
	"errors"
	"fmt"

	"github.com/martinohmann/promise"
)

func main() {
	p1 := promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		reject(errors.New("not in the mood today"))
	})

	p2 := promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		reject(errors.New("meeeh"))
	})

	p3 := promise.Reject(errors.New("nope, just nope"))

	p := promise.Any(p1, p2, p3)

	val, err := p.Await()
	if err != nil {
		panic(err)
	}

	fmt.Printf("A promise fulfilled with value %v\n", val)
}
