package main

import (
	"fmt"

	"github.com/martinohmann/promise"
)

type Foo struct {
	Value string
}

// Then implements promise.Thenable.
func (f *Foo) Then(resolve promise.ResolveFunc, reject promise.RejectFunc) {
	resolve(f.Value)
}

func main() {
	p := promise.Resolve(&Foo{Value: "bar"})

	val, err := p.Await()
	if err != nil {
		fmt.Printf("Promise rejected with error: %v\n", err)
	} else {
		fmt.Printf("Promise fulfilled with value: %v\n", val)
	}
}
