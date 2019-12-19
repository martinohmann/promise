package promise

import (
	"fmt"
	"sync"
)

type State int

const (
	Pending = iota
	Fulfilled
	Rejected
	Adopt
)

type Value interface{}

type OnFulfilledFunc func(val Value) Value
type OnRejectedFunc func(err error) error

type ResolveFunc func(val Value)
type RejectFunc func(err error)

type ExecutorFunc func(resolve ResolveFunc, reject RejectFunc)

type Promise struct {
	sync.Mutex

	execute ExecutorFunc

	value Value
	err   error

	fulfillCallbacks []OnFulfilledFunc
	rejectCallbacks  []OnRejectedFunc

	state State

	wg *sync.WaitGroup
}

func New(executor ExecutorFunc) *Promise {
	p := &Promise{
		wg:               &sync.WaitGroup{},
		execute:          executor,
		fulfillCallbacks: make([]OnFulfilledFunc, 0),
		rejectCallbacks:  make([]OnRejectedFunc, 0),
	}

	if executor != nil {
		p.wg.Add(1)
		go func() {
			defer p.wg.Done()
			defer handlePanic(p)
			p.execute(p.resolve, p.reject)
		}()
	}

	return p
}

func (p *Promise) resolve(val Value) {
	p.Lock()

	if p.state != Pending {
		p.Unlock()
		return
	}

	p.value = val

	for len(p.fulfillCallbacks) > 0 {
		cb := p.fulfillCallbacks[0]
		p.fulfillCallbacks = p.fulfillCallbacks[1:]
		p.Unlock()

		val := cb(p.value)
		if err, ok := val.(error); ok {
			p.reject(err)
			return
		}

		p.Lock()
		p.value = val
	}

	p.state = Fulfilled

	p.Unlock()
}

func (p *Promise) reject(err error) {
	p.Lock()

	if p.state != Pending {
		p.Unlock()
		return
	}

	p.err = err

	for len(p.rejectCallbacks) > 0 {
		cb := p.rejectCallbacks[0]
		p.rejectCallbacks = p.rejectCallbacks[1:]
		p.Unlock()

		p.err = cb(p.err)

		p.Lock()
	}

	p.state = Rejected

	p.Unlock()
}

func handlePanic(promise *Promise) {
	err := recover()
	if err != nil {
		promise.reject(fmt.Errorf("panic while resolving promise: %v", err))
	}
}

func (p *Promise) Await() (Value, error) {
	p.wg.Wait()

	return p.value, p.err
}

func (p *Promise) Then(onFulfilled OnFulfilledFunc, onRejected ...OnRejectedFunc) *Promise {
	p.Lock()
	defer p.Unlock()

	switch p.state {
	case Pending:
		if onFulfilled != nil {
			p.fulfillCallbacks = append(p.fulfillCallbacks, onFulfilled)
		}

		if len(onRejected) > 0 && onRejected[0] != nil {
			p.rejectCallbacks = append(p.rejectCallbacks, onRejected[0])
		}
	case Fulfilled:
		if onFulfilled != nil {
			return Resolve(onFulfilled(p.value))
		}
	case Rejected:
		if len(onRejected) > 0 && onRejected[0] != nil {
			return Reject(onRejected[0](p.err))
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

	return &Promise{
		state: Fulfilled,
		value: val,
	}
}

func Reject(err error) *Promise {
	return New(func(_ ResolveFunc, reject RejectFunc) {
		reject(err)
	})
}
