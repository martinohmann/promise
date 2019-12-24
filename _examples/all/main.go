package main

import (
	"fmt"

	"github.com/martinohmann/promise"
)

func main() {
	p1 := promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		resolve("today is a good day")
	})

	p2 := promise.Resolve("i'm fine")

	p3 := promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		resolve(42)
	})

	p4 := promise.Resolve("yay")

	p := promise.All(p1, p2, p3, p4)

	values, err := p.Await()
	if err != nil {
		panic(err)
	}

	for i, value := range values.([]promise.Value) {
		fmt.Printf("Promise %d was fulfilled with value: %v\n", i+1, value)
	}
}
