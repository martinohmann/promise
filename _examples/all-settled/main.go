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
		resolve(42)
	})

	p3 := promise.Resolve("yay")

	p4 := promise.Reject(errors.New("nope, just nope"))

	p := promise.AllSettled(p1, p2, p3, p4)

	results, _ := p.Await()

	for i, result := range results.([]promise.Result) {
		if result.Err != nil {
			fmt.Printf("Promise %d was rejected with err: %v\n", i+1, result.Err)
		} else {
			fmt.Printf("Promise %d was fulfilled with value: %v\n", i+1, result.Value)
		}
	}
}
