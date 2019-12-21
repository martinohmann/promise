package promise

import (
	"fmt"
	"strings"
)

type indexedValue struct {
	index int
	value interface{}
}

// Race returns a promise that fulfills or rejects as soon as one of the
// promises in the passed slice fulfills or rejects, with the value or reason
// from that promise.
func Race(promises ...Promise) Promise {
	if len(promises) == 0 {
		return Resolve(nil)
	}

	return New(func(resolve ResolveFunc, reject RejectFunc) {
		valChan := make(chan Value, len(promises))
		errChan := make(chan error, len(promises))

		for _, promise := range promises {
			promise.Then(func(val Value) Value {
				valChan <- val
				return val
			}).Catch(func(err error) Value {
				errChan <- err
				return err
			})
		}

		select {
		case val := <-valChan:
			resolve(val)
		case err := <-errChan:
			reject(err)
		}
	})
}

// All method returns a single promise that fulfills when all of the promises
// passed as a slice have been fulfilled or when the slice contains no
// promises. It rejects with the reason of the first promise that rejects.
//
// It is typically used after having started multiple asynchronous tasks to run
// concurrently and having created promises for their results, so that one can
// wait for all the tasks being finished.
func All(promises ...Promise) Promise {
	if len(promises) == 0 {
		return Resolve([]Value{})
	}

	return New(func(resolve ResolveFunc, reject RejectFunc) {
		resChan := make(chan indexedValue, len(promises))
		errChan := make(chan error, len(promises))

		for i, promise := range promises {
			idx := i

			promise.Then(func(val Value) Value {
				resChan <- indexedValue{idx, val}
				return val
			}).Catch(func(err error) Value {
				errChan <- err
				return err
			})
		}

		results := make([]Value, len(promises))

		for i := 0; i < len(promises); i++ {
			select {
			case res := <-resChan:
				results[res.index] = res.value
			case err := <-errChan:
				reject(err)
				return
			}
		}

		resolve(results)
	})
}

// AggregateError is a collection of errors that are aggregated in a single
// error.
type AggregateError []error

// Error implements the error interface. It aggregates the messages of multiple
// errors into a single error string.
func (e AggregateError) Error() string {
	if len(e) == 1 {
		return e[0].Error()
	}

	errStrings := make([]string, len(e))
	for i, err := range e {
		errStrings[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d promises rejected due to errors:\n%s",
		len(e), strings.Join(errStrings, "\n"))
}

// Any takes a slice of promises and, as soon as one of the promises in the
// slice fulfills, returns a single promise that resolves with the value from
// that promise. If no promises in the slice fulfill (if all of the given
// promises are rejected), then the returned promise is rejected with an
// AggregateError, containing all rejection reasons of individual promises.
// Essentially, this func does the opposite of All.
func Any(promises ...Promise) Promise {
	if len(promises) == 0 {
		return Resolve(nil)
	}

	return New(func(resolve ResolveFunc, reject RejectFunc) {
		valChan := make(chan Value, len(promises))
		errChan := make(chan indexedValue, len(promises))

		for i, promise := range promises {
			idx := i

			promise.Then(func(val Value) Value {
				valChan <- val
				return val
			}).Catch(func(err error) Value {
				errChan <- indexedValue{idx, err}
				return err
			})
		}

		errors := make(AggregateError, len(promises))

		for i := 0; i < len(promises); i++ {
			select {
			case val := <-valChan:
				resolve(val)
				return
			case err := <-errChan:
				errors[err.index] = err.value.(error)
			}
		}

		reject(errors)
	})
}

// Result holds a value in case of a fulfilled promise, or a non-nil error if
// the promise was rejected.
type Result struct {
	Value Value
	Err   error
}

// AllSettled returns a promise that resolves after all of the given promises
// have either resolved or rejected, with a slice of Result values that each
// describe the outcome of each promise.
func AllSettled(promises ...Promise) Promise {
	if len(promises) == 0 {
		return Resolve([]Result{})
	}

	return New(func(resolve ResolveFunc, _ RejectFunc) {
		resChan := make(chan indexedValue, len(promises))

		for i, promise := range promises {
			idx := i

			promise.Then(func(val Value) Value {
				resChan <- indexedValue{idx, Result{Value: val}}
				return val
			}).Catch(func(err error) Value {
				resChan <- indexedValue{idx, Result{Err: err}}
				return err
			})
		}

		results := make([]Result, len(promises))

		for i := 0; i < len(promises); i++ {
			res := <-resChan
			results[res.index] = res.value.(Result)
		}

		resolve(results)
	})
}
