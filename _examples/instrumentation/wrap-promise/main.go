package main

import (
	"log"

	"github.com/martinohmann/promise/_examples/instrumentation"

	"github.com/martinohmann/promise"
	"github.com/martinohmann/promise/instrumented"
)

func main() {
	instrumented.AddInstrumentationHandlers(instrumentation.LoggingHandler)
	defer instrumented.RemoveInstrumentationHandlers()

	p := promise.Resolve(42)

	wrapped := instrumented.Wrap(p).Then(func(val promise.Value) promise.Value {
		log.Printf("Promise fulfilled with value: %v", val)
		return val.(int) - 19
	})

	val, err := wrapped.Await()
	if err != nil {
		panic(err)
	}

	log.Printf("Promise fulfilled with value: %v", val)
}
