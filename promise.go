package promise

import (
	"fmt"
	"sync"
)

type state uint8

const (
	pending state = iota
	fulfilled
	rejected
)

// Value describes the value of a fulfilled promise. This is an interface type
// to allow arbitrary types.
type Value interface{}

// OnFulfilledFunc is used in promise fulfillment handlers.
type OnFulfilledFunc func(val Value) Value

// OnRejectedFunc is used in promise rejection handlers.
type OnRejectedFunc func(err error) Value

// ResolveFunc is passed as the first argument to a ResolutionFunc and may be
// called by the user to trigger the promise fulfillment handler chain with the
// provided value.
type ResolveFunc func(val Value)

// RejectFunc is passed as the second argument to a ResolutionFunc and may be
// called by the user to trigger the promise rejection handler chain with the
// provided error value.
type RejectFunc func(val error)

// ResolutionFunc is passed to a promise in order to expose ResolveFunc and
// RejectFunc to the application logic that decides about fulfillment or
// rejection of a promise. At least one of `resolve` or `reject` must be called
// in order to trigger the resolution of a given promise. Subsequent calls to
// `resolve` or `reject` are ignored. Not calling any of the two leaves the
// promise in a pending state. A panic in the ResolutionFunc will be recovered
// and causes the promise to be reject with the panic message.
type ResolutionFunc func(resolve ResolveFunc, reject RejectFunc)

// A Promise represents the eventual completion (or failure) of an asynchronous
// operation, and its resulting value.
type Promise struct {
	sync.Mutex

	done chan struct{}

	state state
	value Value
	err   error

	handlers []*handler
}

type handler struct {
	onFulfilled OnFulfilledFunc
	onRejected  OnRejectedFunc
}

// New creates a new promise with the resolution func fn. Within fn either the
// passed `resolve` or `reject` func must be called exactly once with a value
// or error to fulfill or reject the promise. Neither calling `resolve` nor
// `reject` will cause the promise to be in a pending state.
func New(fn ResolutionFunc) *Promise {
	if fn == nil {
		panic("resolution func must be non-nil")
	}

	p := &Promise{
		handlers: make([]*handler, 0),
		done:     make(chan struct{}),
	}

	go func() {
		defer close(p.done)
		defer handlePanic(p)
		fn(p.resolve, p.reject)
	}()

	return p
}

func (p *Promise) popHandler() *handler {
	h := p.handlers[0]
	p.handlers = p.handlers[1:]

	return h
}

func (p *Promise) resolve(val Value) {
	p.Lock()
	defer p.Unlock()

	p.resolveLocked(val)
}

// resolveLocked resolves the promise. The lock must be held when calling this
// method. This is a performance optimization to avoid releasing the lock when
// val causes the promise to be rejected, e.g. because it is an err value or a
// rejected promise itself. This is necessary to be able to reject a promise in
// a then-handler.
func (p *Promise) resolveLocked(val Value) {
	if p.state != pending {
		return
	}

	switch v := val.(type) {
	case error:
		p.rejectLocked(v)
		return
	case *Promise:
		val, err := v.Await()
		if err != nil {
			p.rejectLocked(err)
			return
		}

		p.value = val
	default:
		p.value = v
	}

	p.err = nil

	for len(p.handlers) > 0 {
		h := p.popHandler()
		if h.onFulfilled == nil {
			continue
		}

		res := h.onFulfilled(p.value)

		switch v := res.(type) {
		case error:
			p.rejectLocked(v)
			return
		case *Promise:
			val, err := v.Await()
			if err != nil {
				p.rejectLocked(err)
				return
			}

			p.value = val
		default:
			p.value = v
		}
	}

	p.state = fulfilled
	p.handlers = nil
}

func (p *Promise) reject(err error) {
	p.Lock()
	defer p.Unlock()

	p.rejectLocked(err)
}

// rejectLocked rejects the promise. The lock must be held when calling this
// method. This is a performance optimization to avoid releasing the lock when
// val is a promise that resolves. This is necessary to be able to recover from
// a rejected promise in a catch-handler.
func (p *Promise) rejectLocked(err error) {
	if p.state != pending {
		return
	}

	p.value = nil
	p.err = err

	for len(p.handlers) > 0 {
		h := p.popHandler()
		if h.onRejected == nil {
			continue
		}

		res := h.onRejected(p.err)

		switch v := res.(type) {
		case *Promise:
			val, err := v.Await()
			if err == nil {
				p.resolveLocked(val)
				return
			}

			p.err = err
		case error:
			p.err = v
		default:
			p.resolveLocked(v)
			return
		}
	}

	p.state = rejected
	p.handlers = nil
}

func handlePanic(promise *Promise) {
	if err := recover(); err != nil {
		promise.reject(fmt.Errorf("panic during promise resolution: %v", err))
	}
}

// Await blocks until the promise is settled, i.e. either fulfilled or
// rejected. It returns the promise value and an error. In the case of a
// rejected promise, the error will be non-nil and contains the rejection
// reason. The returned value contains the result of a fulfulled promise.
func (p *Promise) Await() (Value, error) {
	<-p.done

	return p.value, p.err
}

// Then adds a handler to handle promise fulfillment (onFulfilled). Optionally
// it also accepts a second handler func to handle promise rejection cases
// (onRejected). Passing a nil func for either of the two results in no
// handlers to be created for these cases. Returning non-nil errors, rejected
// promises or panics in the onFulfilled handler will reject the promise. Any
// other value will be passed to the next onFulfilled handler in the chain,
// eventually resolving the promise if there are now more handlers left.
// Similarly, non-nil errors and rejected promises returned by or panics during
// the execution of the onRejected handler will result in the promise to be
// rejected, whereas any value different from these will recover from the
// rejection and trigger the next onFulfilled handler. Returns a promise.
func (p *Promise) Then(onFulfilled OnFulfilledFunc, onRejected ...OnRejectedFunc) *Promise {
	p.Lock()
	defer p.Unlock()

	switch p.state {
	case pending:
		if onFulfilled != nil {
			p.handlers = append(p.handlers, &handler{onFulfilled: onFulfilled})
		}

		if len(onRejected) > 0 && onRejected[0] != nil {
			p.handlers = append(p.handlers, &handler{onRejected: onRejected[0]})
		}
	case fulfilled:
		if onFulfilled != nil {
			return promiseValue(onFulfilled(p.value))
		}
	case rejected:
		if len(onRejected) > 0 && onRejected[0] != nil {
			return promiseValue(onRejected[0](p.err))
		}
	}

	return p
}

func promiseValue(val Value) *Promise {
	switch v := val.(type) {
	case *Promise:
		return v
	case error:
		return Reject(v)
	default:
		return Resolve(v)
	}
}

// Catch adds a handler to handle promise rejections. It behaves the same as
// calling Promise.Then(nil, onRejected) instead. A passed nil func will be
// ignored and does not cause any errors. See the documentation of Then for
// details on how return values of the onRejected handler affect the promise
// resolution. Returns a promise.
func (p *Promise) Catch(onRejected OnRejectedFunc) *Promise {
	return p.Then(nil, onRejected)
}

// Finally executes fn when the promise is settled, i.e. either fulfilled or
// rejected. This provides a way to run code regardless of the outcome of the
// promise resolution process. Returns a promise.
func (p *Promise) Finally(fn func()) *Promise {
	return p.Then(func(val Value) Value {
		defer fn()
		return val
	}).Catch(func(err error) Value {
		defer fn()
		return err
	})
}

// Resolve returns a promise that is resolved with given value. If val is a
// non-nil error or a rejected promise, the promise will be resolved to a
// rejected promise instead.
func Resolve(val Value) *Promise {
	return New(func(resolve ResolveFunc, _ RejectFunc) {
		resolve(val)
	})
}

// Reject returns a promise that is rejected with given error reason.
func Reject(err error) *Promise {
	return New(func(_ ResolveFunc, reject RejectFunc) {
		reject(err)
	})
}
