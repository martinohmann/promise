package main

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/martinohmann/promise"
)

type timingPromise struct {
	promise.Promise
}

func logExecution(start time.Time, format string, args ...interface{}) {
	format = fmt.Sprintf("[%12s] %s", time.Now().Sub(start), format)
	log.Printf(format, args...)
}

func WithTiming(p promise.Promise) promise.Promise {
	return &timingPromise{Promise: p}
}

func (p *timingPromise) Then(onFulfilled promise.OnFulfilledFunc, onRejected ...promise.OnRejectedFunc) promise.Promise {
	instrumentedOnRejected := make([]promise.OnRejectedFunc, len(onRejected))
	for i, fn := range onRejected {
		instrumentedOnRejected[i] = func(err error) promise.Value {
			defer logExecution(time.Now(), "<p:%p> onRejected with err %#v", err)
			return fn(err)
		}
	}

	return WithTiming(p.Promise.Then(func(val promise.Value) promise.Value {
		defer logExecution(time.Now(), "<p:%p> onFulfilled with val %#v", p.Promise, val)
		return onFulfilled(val)
	}, instrumentedOnRejected...))
}

func (p *timingPromise) Catch(onRejected promise.OnRejectedFunc) promise.Promise {
	return WithTiming(p.Promise.Catch(func(err error) promise.Value {
		defer logExecution(time.Now(), "<p:%p> onRejected with err %#v", p.Promise, err)
		return onRejected(err)
	}))
}

func (p *timingPromise) Finally(fn func()) promise.Promise {
	return WithTiming(p.Promise.Finally(func() {
		defer logExecution(time.Now(), "<p:%p> onFinally", p.Promise)
		fn()
	}))
}

func (p *timingPromise) Await() (promise.Value, error) {
	defer logExecution(time.Now(), "<p:%p> Await", p.Promise)
	return p.Promise.Await()
}

func main() {
	p := WithTiming(
		promise.Resolve(42),
	).Then(func(val promise.Value) promise.Value {
		time.Sleep(100 * time.Millisecond)
		return errors.New("ooops")
	}).Catch(func(err error) promise.Value {
		time.Sleep(350 * time.Millisecond)
		return WithTiming(promise.Resolve(50)).Then(func(val promise.Value) promise.Value {
			time.Sleep(10 * time.Millisecond)
			return val
		})
	}).Finally(func() {
		time.Sleep(50 * time.Millisecond)
	})

	p.Await()
}
