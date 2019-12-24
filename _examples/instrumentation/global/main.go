package main

import (
	"errors"
	"log"
	"strings"
	"time"

	"github.com/martinohmann/promise/_examples/instrumentation"

	// Drop in replacement for github.com/martinohmann/promise. Exports the
	// same symbols as the promise package.
	promise "github.com/martinohmann/promise/instrumented"
)

func main() {
	uuidCollector := instrumentation.NewUUIDCollector()

	// add global instrumentation handlers to all promises created via the
	// github.com/martinohmann/promise/instrumented package.
	promise.AddInstrumentationHandlers(instrumentation.LoggingHandler, uuidCollector.CollectUUIDs)
	// Handlers added via promise.AddInstrumentationHandlers can be cleaned up like this:
	// defer promise.RemoveInstrumentationHandlers()

	p := promise.Resolve(42).Then(func(val promise.Value) promise.Value {
		time.Sleep(100 * time.Millisecond)
		return errors.New("ooops")
	}).Catch(func(err error) promise.Value {
		time.Sleep(350 * time.Millisecond)
		return promise.Resolve(50).Then(func(val promise.Value) promise.Value {
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
