package promise

import (
	"fmt"
	"sync"
)

type State uint8

const (
	Pending = iota
	Fulfilled
	Rejected
)

type Value interface{}

type OnFulfilledFunc func(val Value) Value

type OnRejectedFunc func(err error) Value

type ResolveFunc func(val Value)

type RejectFunc func(val error)

type ResolutionFunc func(resolve ResolveFunc, reject RejectFunc)

type Promise struct {
	sync.Mutex

	done chan struct{}

	state State
	value Value
	err   error

	handlers []*handler
}

type handler struct {
	onFulfilled OnFulfilledFunc
	onRejected  OnRejectedFunc
}

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

func (p *Promise) resolve(val Value) {
	p.Lock()
	defer p.Unlock()

	p.resolveLocked(val)
}

// resolveLocked resolves the promise. The lock must be held when calling this
// method. This is a performance optimization to avoid releasing the lock when
// val causes the promise to be rejected, e.g. because it is an err value or a
// rejected promise itself. This is necessary to be able to reject a promise in
// a then-callback.
func (p *Promise) resolveLocked(val Value) {
	if p.state != Pending {
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
		h := p.handlers[0]
		p.handlers = p.handlers[1:]
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

	p.state = Fulfilled
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
// a rejected promise in a catch-callback.
func (p *Promise) rejectLocked(err error) {
	if p.state != Pending {
		return
	}

	p.value = nil
	p.err = err

	for len(p.handlers) > 0 {
		h := p.handlers[0]
		p.handlers = p.handlers[1:]
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

	p.state = Rejected
	p.handlers = nil
}

func handlePanic(promise *Promise) {
	if err := recover(); err != nil {
		promise.reject(fmt.Errorf("panic while resolving promise: %v", err))
	}
}

func (p *Promise) Await() (Value, error) {
	<-p.done

	return p.value, p.err
}

func (p *Promise) Then(onFulfilled OnFulfilledFunc, onRejected ...OnRejectedFunc) *Promise {
	p.Lock()
	defer p.Unlock()

	switch p.state {
	case Pending:
		if onFulfilled != nil {
			p.handlers = append(p.handlers, &handler{onFulfilled: onFulfilled})
		}

		if len(onRejected) > 0 && onRejected[0] != nil {
			p.handlers = append(p.handlers, &handler{onRejected: onRejected[0]})
		}
	case Fulfilled:
		if onFulfilled != nil {
			return Resolve(onFulfilled(p.value))
		}
	case Rejected:
		if len(onRejected) > 0 && onRejected[0] != nil {
			res := onRejected[0](p.err)

			switch v := res.(type) {
			case *Promise:
				return v
			case error:
				return Reject(v)
			default:
				return Resolve(v)
			}
		}
	}

	return p
}

func (p *Promise) Catch(onRejected OnRejectedFunc) *Promise {
	return p.Then(nil, onRejected)
}

func Resolve(val Value) *Promise {
	if p, ok := val.(*Promise); ok {
		return p
	}

	return New(func(resolve ResolveFunc, _ RejectFunc) {
		resolve(val)
	})
}

func Reject(err error) *Promise {
	return New(func(_ ResolveFunc, reject RejectFunc) {
		reject(err)
	})
}
