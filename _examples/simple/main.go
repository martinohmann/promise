package main

import (
	"errors"
	"fmt"
	"math/rand"
	"time"

	"github.com/martinohmann/promise"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

func main() {
	p := promise.New(func(resolve promise.ResolveFunc, reject promise.RejectFunc) {
		// simulate some computation
		sleepDuration := time.Duration(rand.Int63n(2000)) * time.Millisecond
		time.Sleep(sleepDuration)

		fmt.Printf("computation took %s\n", sleepDuration)

		// inject some random errors
		if rand.Int63n(2) == 0 {
			reject(errors.New("computation failed"))
			return
		}

		// simulate computation result
		resolve(rand.Int63())
	})

	// Wait for the promise resolution to be complete, that is: either fulfillment or rejection.
	val, err := p.Await()
	if err != nil {
		fmt.Printf("Promise rejected with error: %v\n", err)
	} else {
		fmt.Printf("Promise fulfilled with value: %d\n", val.(int64))
	}
}
