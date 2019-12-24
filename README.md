Promise
=======

[![Build Status](https://travis-ci.org/martinohmann/promise.svg?branch=master)](https://travis-ci.org/martinohmann/promise)
[![codecov](https://codecov.io/gh/martinohmann/promise/branch/master/graph/badge.svg)](https://codecov.io/gh/martinohmann/promise)
[![GoDoc](https://godoc.org/github.com/martinohmann/promise?status.svg)](https://godoc.org/github.com/martinohmann/promise)
[![Go Report Card](https://goreportcard.com/badge/github.com/martinohmann/promise)](https://goreportcard.com/report/github.com/martinohmann/promise)

Initially this started out as an experiment to better understand the inner workings of
JavaScript promises and how this can be implemented from scratch in go.

The result is a promise implementation which follows the [Promises/A+
Standard](https://promisesaplus.com/) and comes with the following additional
features:

* Waiting for promise resolution.
* `Race`, `All`, `Any` and `AllSettled` extensions to handle the parallel
  resolution of multiple promises.
* Promise instrumentation for tracing, logging and debugging. See the
  [`instrumented` package
  documentation](https://godoc.org/github.com/martinohmann/promise/instrumented)
  for more information.

Head over to the [`promise`
godoc](https://godoc.org/github.com/martinohmann/promise) for the API
documentation.

Installation
------------

```
go get -u github.com/martinohmann/promise
```

Usage
-----

Check out the examples in the [_examples/](_examples/) directory to see promises in action.

### Simple example

```go
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
	}).Then(func(val promise.Value) promise.Value {
		fmt.Printf("computation result: %d\n", val.(int64))
		return val
	}).Catch(func(err error) promise.Value {
		fmt.Printf("error during computation: %v\n", err)
		return err
	})

	// Wait for the promise resolution to be complete, that is: either fulfillment or rejection.
	val, err := p.Await()
	if err != nil {
		fmt.Printf("Promise rejected with error: %v\n", err)
	} else {
		fmt.Printf("Promise fulfilled with value: %d\n", val.(int64))
	}
}
```

License
-------

The source code of promise is released under the MIT License. See the bundled
LICENSE file for details.
