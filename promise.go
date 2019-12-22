package promise

import (
	"errors"
	"fmt"
	"sync"
)

// state is the type of the promise state. Is one of: pending, fulfilled or
// rejected.
type state uint8

const (
	pending state = iota
	fulfilled
	rejected
)

// Value describes the value of a fulfilled promise. This is an interface type
// to allow arbitrary types.
type Value = interface{}

// OnFulfilledFunc is used in promise fulfillment handlers.
type OnFulfilledFunc = func(val Value) Value

// OnRejectedFunc is used in promise rejection handlers.
type OnRejectedFunc = func(err error) Value

// ResolveFunc is passed as the first argument to a ResolutionFunc and may be
// called by the user to trigger the promise fulfillment handler chain with the
// provided value.
type ResolveFunc = func(val Value)

// RejectFunc is passed as the second argument to a ResolutionFunc and may be
// called by the user to trigger the promise rejection handler chain with the
// provided error value.
type RejectFunc = func(val error)

// ResolutionFunc is passed to a promise in order to expose ResolveFunc and
// RejectFunc to the application logic that decides about fulfillment or
// rejection of a promise. At least one of `resolve` or `reject` must be called
// in order to trigger the resolution of a given promise. Subsequent calls to
// `resolve` or `reject` are ignored. Not calling any of the two leaves the
// promise in a pending state. A panic in the ResolutionFunc will be recovered
// and causes the promise to be reject with the panic message. ResolutionFunc
// itself implements Thenable.
type ResolutionFunc func(resolve ResolveFunc, reject RejectFunc)

// Then implements Thenable.
func (f ResolutionFunc) Then(resolve ResolveFunc, reject RejectFunc) {
	f(resolve, reject)
}

// A Promise represents the eventual completion (or failure) of an asynchronous
// operation, and its resulting value.
type Promise interface {
	// Then adds a handler to handle promise fulfillment (onFulfilled). Optionally
	// it also accepts a second handler func to handle promise rejection cases
	// (onRejected). Passing a nil func for either of the two results in no
	// handlers to be created for these cases. Returning non-nil errors, rejected
	// promises or panics in the onFulfilled handler will reject the promise. Any
	// other value will be passed to the next onFulfilled handler in the chain,
	// eventually resolving the promise if there are no more handlers left.
	// Similarly, non-nil errors and rejected promises returned by or panics during
	// the execution of the onRejected handler will result in the promise to be
	// rejected, whereas any value different from these will recover from the
	// rejection and trigger the next onFulfilled handler. Returns a promise.
	Then(onFulfilled OnFulfilledFunc, onRejected ...OnRejectedFunc) Promise

	// Catch adds a handler to handle promise rejections. It behaves the same as
	// calling Promise.Then(nil, onRejected) instead. A passed nil func will be
	// ignored and does not cause any errors. See the documentation of Then for
	// details on how return values of the onRejected handler affect the promise
	// resolution. Returns a promise.
	Catch(onRejected OnRejectedFunc) Promise

	// Finally executes fn when the promise is settled, i.e. either fulfilled or
	// rejected. This provides a way to run code regardless of the outcome of the
	// promise resolution process. Returns a promise.
	Finally(fn func()) Promise

	// Await blocks until the promise is settled, i.e. either fulfilled or
	// rejected. It returns the promise value and an error. In the case of a
	// rejected promise, the error will be non-nil and contains the rejection
	// reason. The returned value contains the result of a fulfulled promise.
	Await() (Value, error)
}

// A Thenable is a special kind of handler that can be returned during promise
// resolution/rejection.
type Thenable interface {
	Then(resolve ResolveFunc, reject RejectFunc)
}

type promise struct {
	sync.Mutex

	done chan struct{}

	state state
	value Value
	err   error

	handlers []*handler
}

// handler is a holder for deferred resolution or rejection handlers. The
// implementation assumes that only one of onFulfilled and onRejected is a
// non-nil function, but not both.
type handler struct {
	onFulfilled OnFulfilledFunc
	onRejected  OnRejectedFunc
}

// New creates a new promise with the resolution func fn. Within fn either the
// passed `resolve` or `reject` func must be called exactly once with a value
// or error to fulfill or reject the promise. Neither calling `resolve` nor
// `reject` will cause the promise to stay in a pending state.
func New(fn ResolutionFunc) Promise {
	if fn == nil {
		panic("resolution func must be non-nil")
	}

	p := &promise{
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

// popHandler removes the next handler from the handler chain and returns it.
// Will panic if called when the handlers slice is empty.
func (p *promise) popHandler() *handler {
	h := p.handlers[0]
	p.handlers = p.handlers[1:]

	return h
}

// resolve attempts to fulfill p with val. This internally calls resolveLocked
// after acquiring the lock.
func (p *promise) resolve(val Value) {
	p.Lock()
	defer p.Unlock()

	p.resolveLocked(val)
}

// ErrCircularResolutionChain is the error that a promise is rejected with if a
// circular resolution dependency is detected, that is: an attempt to resolve
// or reject an promise with itself at arbitrary depth it the chain.
var ErrCircularResolutionChain = errors.New("circular resolution chain: a promise cannot be resolved or rejected with itself")

// resolveLocked resolves the promise. The lock must be held when calling this
// method. This is a performance optimization to avoid releasing the lock when
// val causes the promise to be rejected, e.g. because it is an err value or a
// rejected promise itself. This is necessary to be able to reject a promise in
// a then-handler.
func (p *promise) resolveLocked(val Value) {
	if p.state != pending {
		return
	}

	val, err := p.resolveValue(val)
	if err != nil {
		p.rejectLocked(err)
		return
	}

	p.value = val
	p.err = nil

	for len(p.handlers) > 0 {
		h := p.popHandler()
		if h.onFulfilled == nil {
			// This is an onRejected handler which gets ignored since we are
			// trying to resolve the promise. We must discard it or we might
			// get weird behaviour later on.
			continue
		}

		res := h.onFulfilled(p.value)

		val, err := p.resolveValue(res)
		if err != nil {
			p.rejectLocked(err)
			return
		}

		p.value = val
	}

	p.state = fulfilled
	p.handlers = nil
}

// resolveValue tries to resolve all nested thenables and promises in val to
// the final promise value and return that as the first return value. As the
// second return value a non-nil error is returned if one of the following
// cases applies: a promise or thenable in the chain rejected, the value was a
// non-nil error, or a circular dependency in the resolution chain was
// detected.
func (p *promise) resolveValue(val Value) (Value, error) {
	switch v := val.(type) {
	case error:
		return nil, v
	case Thenable:
		return New(v.Then).Await()
	case Promise:
		if v == p {
			return nil, ErrCircularResolutionChain
		}

		return v.Await()
	default:
		return v, nil
	}
}

// reject attempts to reject p with err. This internally calls rejectLocked
// after acquiring the lock.
func (p *promise) reject(err error) {
	p.Lock()
	defer p.Unlock()

	p.rejectLocked(err)
}

// rejectLocked rejects the promise. The lock must be held when calling this
// method. This is a performance optimization to avoid releasing the lock when
// val is a promise that resolves. This is necessary to be able to recover from
// a rejected promise in a catch-handler.
func (p *promise) rejectLocked(err error) {
	if p.state != pending {
		return
	}

	p.value = nil
	p.err = err

	for len(p.handlers) > 0 {
		h := p.popHandler()
		if h.onRejected == nil {
			// This is an onFulfilled handler which gets ignored since we are
			// trying to reject the promise. We must discard it or we might get
			// weird behaviour later on.
			continue
		}

		res := h.onRejected(p.err)

		val, err := p.resolveValue(res)
		if err == nil {
			p.resolveLocked(val)
			return
		}

		p.err = err
	}

	p.state = rejected
	p.handlers = nil
}

// handlePanic handles a panic by rejecting the promise with the panic reason.
// Must be called as a deferred function at the beginning of the code section
// that may panic.
func handlePanic(promise *promise) {
	if err := recover(); err != nil {
		promise.reject(fmt.Errorf("panic during promise resolution: %v", err))
	}
}

// Await implements the Await method of the Promise interface.
func (p *promise) Await() (Value, error) {
	<-p.done

	return p.value, p.err
}

// Then implements the Then method of the Promise interface.
func (p *promise) Then(onFulfilled OnFulfilledFunc, onRejected ...OnRejectedFunc) Promise {
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
			return Resolve(p.value).Then(onFulfilled)
		}
	case rejected:
		if len(onRejected) > 0 && onRejected[0] != nil {
			return Reject(p.err).Then(nil, onRejected[0])
		}
	}

	return p
}

// Catch implements the Catch method of the Promise interface.
func (p *promise) Catch(onRejected OnRejectedFunc) Promise {
	return p.Then(nil, onRejected)
}

// Finally implements the Finally method of the Promise interface.
func (p *promise) Finally(fn func()) Promise {
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
func Resolve(val Value) Promise {
	return New(func(resolve ResolveFunc, _ RejectFunc) {
		resolve(val)
	})
}

// Reject returns a promise that is rejected with given error reason.
func Reject(err error) Promise {
	return New(func(_ ResolveFunc, reject RejectFunc) {
		reject(err)
	})
}
