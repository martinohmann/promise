package main

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/martinohmann/promise"
	"github.com/martinohmann/promise/_examples/instrumentation"
	"github.com/martinohmann/promise/instrumented"
)

func main() {
	uuidCollector := instrumentation.NewUUIDCollector()

	// Create an instrumented promise factory.
	instrumentedPromise := instrumented.NewInstrumentation(instrumentation.LoggingHandler, uuidCollector.CollectUUIDs)

	p := promise.Resolve(42).Then(func(val promise.Value) promise.Value {
		time.Sleep(100 * time.Millisecond)
		return errors.New("ooops")
	}).Catch(func(err error) promise.Value {
		time.Sleep(350 * time.Millisecond)

		// Selectively instrument promises.
		return instrumentedPromise.Resolve(50).Then(func(val promise.Value) promise.Value {
			time.Sleep(10 * time.Millisecond)
			return val
		})
	}).Finally(func() {
		time.Sleep(50 * time.Millisecond)
	})

	p.Await()

	uuids := uuidCollector.GetUUIDs()
	log.Printf("collected %d UUIDs: %s", len(uuids), strings.Join(uuids, ", "))
}
