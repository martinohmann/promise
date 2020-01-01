package promise

import (
	"context"
	"sync"
)

// PoolEventListener can be attached to a promise pool to listen for
// fulfillment and rejection events of the promises created and tracked by the
// pool. This can be used for logging or collecting values.
type PoolEventListener struct {
	// OnFulfilled is called on each promise fulfillment.
	OnFulfilled func(val Value)

	// OnRejected is called on each promise rejection.
	OnRejected func(err error)
}

// A Pool creates promises from a stream of promise factory funcs and
// supervises their resolution. It ensures that only a configurable number of
// promises will be resolved concurrently.
type Pool struct {
	mu        sync.Mutex
	sem       chan struct{}
	done      chan struct{}
	result    chan Result
	fns       <-chan func() Promise
	listeners []*PoolEventListener
	promise   Promise
}

// NewPool creates a new promise pool with given concurrency and channel which
// provides promise factory funcs. Negative concurrency values will cause a
// panic. Nil funcs or nil promises returned by the funcs from the channel will
// also cause panics when Run is called on the pool.
func NewPool(concurrency int64, fns <-chan func() Promise) *Pool {
	if concurrency <= 0 {
		panic("concurrency must be greater than 0")
	}

	return &Pool{
		fns:    fns,
		sem:    make(chan struct{}, concurrency),
		done:   make(chan struct{}),
		result: make(chan Result),
	}
}

// Run starts the pool. This will consume the funcs from the channel provided
// to NewPool with the configured concurrency. It returns a promise which
// fulfills once the channel providing the promise factory funcs is closed. The
// promise rejects upon the first error encountered or if ctx is cancelled. Run
// must only be called once. Subsequent calls to it will panic.
func (p *Pool) Run(ctx context.Context) Promise {
	if p.promise != nil {
		panic("promise pool cannot be started twice")
	}

	p.promise = New(func(resolve ResolveFunc, reject RejectFunc) {
		defer func() {
			p.mu.Lock()
			p.listeners = nil
			close(p.done)
			p.mu.Unlock()
		}()

		select {
		case res := <-p.result:
			if res.Err != nil {
				reject(res.Err)
				return
			}

			resolve(res.Value)
		case <-ctx.Done():
			reject(ctx.Err())
		}
	})

	go p.run(ctx)

	return p.promise
}

func (p *Pool) run(ctx context.Context) {
	for {
		select {
		case fn, ok := <-p.fns:
			if !ok {
				// Fns channel was closed, we need to stop. By consuming all
				// semaphores we make sure that all promises that are currently
				// in flight resolved before we send the final result.
				for i := 0; i < cap(p.sem); i++ {
					p.sem <- struct{}{}
				}

				p.result <- Result{}
				return
			}

			// Wait for a semaphore before executing the promise factory func.
			select {
			case p.sem <- struct{}{}:
				p.execute(fn)
			case <-p.done:
				// One of the promises that are currently in flight rejected or
				// ctx was cancelled which in turn caused the pool promise to
				// reject while waiting for sem. We exit here as there is no
				// point in continuing.
				return
			}
		}
	}
}

func (p *Pool) execute(fn func() Promise) {
	fn().Then(func(val Value) Value {
		p.dispatchFulfillment(val)
		<-p.sem

		return val
	}).Catch(func(err error) Value {
		p.dispatchRejection(err)

		// Use a select with default to prevent blocking in the case where a
		// result was already sent by another catch handler. We would discard
		// it anyways as the first error by any promise immediately rejects the
		// pool promise. Also, we avoid leaking a goroutine by that.
		select {
		case p.result <- Result{Err: err}:
		default:
		}

		<-p.sem

		return err
	})
}

func (p *Pool) dispatchFulfillment(val Value) {
	p.mu.Lock()
	listeners := p.listeners
	p.mu.Unlock()

	for _, l := range listeners {
		if l.OnFulfilled != nil {
			l.OnFulfilled(val)
		}
	}
}

func (p *Pool) dispatchRejection(err error) {
	p.mu.Lock()
	listeners := p.listeners
	p.mu.Unlock()

	for _, l := range listeners {
		if l.OnRejected != nil {
			l.OnRejected(err)
		}
	}
}

// AddEventListener adds listener to the pool. Will not add it again if
// listener is already present.
func (p *Pool) AddEventListener(listener *PoolEventListener) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, l := range p.listeners {
		if l == listener {
			return
		}
	}

	p.listeners = append(p.listeners, listener)
}

// RemoveEventListener removes listener from the pool if it was present.
func (p *Pool) RemoveEventListener(listener *PoolEventListener) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, l := range p.listeners {
		if l == listener {
			p.listeners = append(p.listeners[:i], p.listeners[i+1:]...)
			return
		}
	}
}
