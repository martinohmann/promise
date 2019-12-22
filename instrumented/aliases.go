package instrumented

import "github.com/martinohmann/promise"

// Alias exported promise package types to allow usage of the
// promise/instrumented package as drop in replacement.
type (
	// Value describes the value of a fulfilled promise.
	Value = promise.Value

	// OnFulfilledFunc is used in promise fulfillment handlers.
	OnFulfilledFunc = promise.OnFulfilledFunc

	// OnRejectedFunc is used in promise rejection handlers.
	OnRejectedFunc = promise.OnRejectedFunc

	// ResolveFunc is passed as the first argument to a ResolutionFunc and may be
	// called by the user to trigger the promise fulfillment handler chain with the
	// provided value.
	ResolveFunc = promise.ResolveFunc

	// RejectFunc is passed as the second argument to a ResolutionFunc and may be
	// called by the user to trigger the promise rejection handler chain with the
	// provided error value.
	RejectFunc = promise.RejectFunc

	// ResolutionFunc is passed to a promise in order to expose ResolveFunc and
	// RejectFunc to the application logic that decides about fulfillment or
	// rejection of a promise. At least one of `resolve` or `reject` must be called
	// in order to trigger the resolution of a given promise. Subsequent calls to
	// `resolve` or `reject` are ignored. Not calling any of the two leaves the
	// promise in a pending state.
	ResolutionFunc = promise.ResolutionFunc

	// A Promise represents the eventual completion (or failure) of an asynchronous
	// operation, and its resulting value.
	Promise = promise.Promise

	// A Thenable is a special kind of handler that can be returned during promise
	// resolution/rejection.
	Thenable = promise.Thenable

	// AggregateError is a collection of errors that are aggregated in a single
	// error.
	AggregateError = promise.AggregateError

	// Result holds a value in case of a fulfilled promise, or a non-nil error if
	// the promise was rejected.
	Result = promise.Result
)

// Alias exported promise package variables to allow usage of the
// promise/instrumented package as drop in replacement.
var (
	// ErrCircularResolutionChain is the error that a promise is rejected with if a
	// circular resolution dependency is detected, that is: an attempt to resolve
	// or reject an promise with itself at arbitrary depth it the chain.
	ErrCircularResolutionChain = promise.ErrCircularResolutionChain

	// Race returns a promise that fulfills or rejects as soon as one of the
	// promises in the passed slice fulfills or rejects, with the value or reason
	// from that promise.
	Race = promise.Race

	// All method returns a single promise that fulfills when all of the promises
	// passed as a slice have been fulfilled or when the slice contains no
	// promises. It rejects with the reason of the first promise that rejects.
	//
	// It is typically used after having started multiple asynchronous tasks to run
	// concurrently and having created promises for their results, so that one can
	// wait for all the tasks being finished.
	All = promise.All

	// Any takes a slice of promises and, as soon as one of the promises in the
	// slice fulfills, returns a single promise that resolves with the value from
	// that promise. If no promises in the slice fulfill (if all of the given
	// promises are rejected), then the returned promise is rejected with an
	// AggregateError, containing all rejection reasons of individual promises.
	// Essentially, this func does the opposite of All.
	Any = promise.Any

	// AllSettled returns a promise that resolves after all of the given promises
	// have either resolved or rejected, with a slice of Result values that each
	// describe the outcome of each promise.
	AllSettled = promise.AllSettled
)
