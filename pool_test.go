package promise

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"
)

func makePool(concurrency int, options ...PoolOptions) (*Pool, chan func() Promise) {
	fns := make(chan func() Promise)
	pool := NewPool(concurrency, fns, options...)
	return pool, fns
}

func TestNewPool_Panic(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	NewPool(0, make(chan func() Promise))
}

func TestPool(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	var fulfilled int64

	pool, fns := makePool(10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer close(fns)

		for i := 0; i < 10; i++ {
			select {
			case fns <- func() Promise {
				return Resolve(nil).Then(func(val Value) Value {
					atomic.AddInt64(&fulfilled, 1)
					return val
				})
			}:
			case <-ctx.Done():
				return
			}
		}
	}()

	p := pool.Run(ctx)

	_, err := awaitWithTimeout(t, p, 2*time.Second)
	if err != nil {
		t.Fatalf("expected nil err but got: %v", err)
	}

	if fulfilled != 10 {
		t.Fatalf("expected 10 promises to be fulfilled but got %d", fulfilled)
	}
}

func TestPool_RunTwicePanic(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	pool, _ := makePool(10)
	pool.Run(ctx)
	pool.Run(ctx)
}

func TestPool_Error(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	pool, fns := makePool(10)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer close(fns)

		for i := 0; i < 10; i++ {
			select {
			case fns <- func(val int) func() Promise {
				return func() Promise {
					if val < 5 {
						return Resolve(nil)
					}

					return Reject(fmt.Errorf("error in %d", val))
				}
			}(i):
			case <-ctx.Done():
				return
			}
		}
	}()

	p := pool.Run(ctx)

	errPattern := `^error in [5-9]$`

	_, err := awaitWithTimeout(t, p, 2*time.Second)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	if matched, _ := regexp.MatchString(errPattern, err.Error()); !matched {
		t.Fatalf("expected err to match pattern %q, but got %q", errPattern, err.Error())
	}
}

func TestPool_ContextCancel(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	pool, fns := makePool(10)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	done := make(chan struct{})
	defer close(done)

	go func() {
		defer close(fns)
		fns <- func() Promise {
			return New(func(resolve ResolveFunc, reject RejectFunc) {
				<-done
				resolve("done")
			})
		}
	}()

	p := pool.Run(ctx)

	_, err := awaitWithTimeout(t, p, 2*time.Second)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected error %v got %v", context.DeadlineExceeded, err)
	}
}

func TestPool_ContinueOnError(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	var fulfilled int64
	var rejected int64

	pool, fns := makePool(10, PoolOptions{ContinueOnError: true})

	pool.AddEventListener(&PoolEventListener{
		OnFulfilled: func(val Value) {
			atomic.AddInt64(&fulfilled, 1)
		},
		OnRejected: func(err error) {
			atomic.AddInt64(&rejected, 1)
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer close(fns)

		for i := 0; i < 10; i++ {
			select {
			case fns <- func(val int) func() Promise {
				return func() Promise {
					if val%2 == 0 {
						return Resolve(nil)
					}

					return Reject(fmt.Errorf("error in %d", val))
				}
			}(i):
			case <-ctx.Done():
				return
			}
		}
	}()

	p := pool.Run(ctx)

	_, err := awaitWithTimeout(t, p, 2*time.Second)
	if err != nil {
		t.Fatalf("expected nil error but got: %v", err)
	}

	n := atomic.LoadInt64(&fulfilled)
	if n != 5 {
		t.Fatalf("expected 5 promises to be fulfilled but got %d", n)
	}

	n = atomic.LoadInt64(&rejected)
	if n != 5 {
		t.Fatalf("expected 5 promises to be rejected but got %d", n)
	}
}

func TestPool_AddEventListener(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	var fulfilled int64
	var rejected int64

	listener := &PoolEventListener{
		OnFulfilled: func(val Value) {
			atomic.AddInt64(&fulfilled, 1)
		},
		OnRejected: func(err error) {
			atomic.AddInt64(&rejected, 1)
		},
	}

	pool, fns := makePool(1)

	// double add on purpose
	pool.AddEventListener(listener)
	pool.AddEventListener(listener)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		defer close(fns)

		for i := 0; i < 10; i++ {
			select {
			case fns <- func(val int) func() Promise {
				return func() Promise {
					if val < 5 {
						return Resolve(nil)
					}

					return Reject(fmt.Errorf("error in %d", val))
				}
			}(i):
			case <-ctx.Done():
				return
			}
		}
	}()

	p := pool.Run(ctx)

	_, err := awaitWithTimeout(t, p, 2*time.Second)
	if err == nil {
		t.Fatal("expected error but got nil")
	}

	n := atomic.LoadInt64(&fulfilled)
	if n != 5 {
		t.Fatalf("expected 5 promises to be fulfilled but got %d", n)
	}

	n = atomic.LoadInt64(&rejected)
	if n < 1 {
		t.Fatalf("expected 1 or more promises to be rejected but got %d", n)
	}
}

func TestPool_AddEventListener_nilListener(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	NewPool(1, make(chan func() Promise)).AddEventListener(nil)
}

func TestPool_RemoveEventListener(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	listener := &PoolEventListener{
		OnFulfilled: func(val Value) {
			t.Fatalf("unexpected call to OnFulfilled with value: %v", val)
		},
		OnRejected: func(err error) {
			t.Fatalf("unexpected call to OnRejected with value: %v", err)
		},
	}

	pool, fns := makePool(10)
	pool.AddEventListener(listener)
	pool.RemoveEventListener(listener)

	go func() {
		defer close(fns)
		fns <- func() Promise { return Resolve(nil) }
	}()

	p := pool.Run(context.Background())

	_, err := awaitWithTimeout(t, p, 2*time.Second)
	if err != nil {
		t.Fatalf("expected nil error but got: %v", err)
	}
}

func TestPool_ErrorWhileWaitingForSemaphore(t *testing.T) {
	defer goleak.VerifyNoLeaks(t)

	pool, fns := makePool(1)

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
	defer cancel()

	go func() {
		defer close(fns)

		for i := 0; i < 2; i++ {
			select {
			case fns <- func(val int) func() Promise {
				return func() Promise {
					if val%2 == 0 {
						return New(func(resolve ResolveFunc, reject RejectFunc) {
							// sleep a little so we are sure the next item read
							// from the channel is waiting for the semaphore
							// when the context is cancelled due to the timeout.
							time.Sleep(50 * time.Millisecond)
							reject(errors.New("error in long running operation"))
						})
					}

					return Resolve(nil)
				}
			}(i):
			case <-ctx.Done():
				return
			}
		}
	}()

	p := pool.Run(ctx)

	_, err := awaitWithTimeout(t, p, 2*time.Second)
	if err != context.DeadlineExceeded {
		t.Fatalf("expected error %v got %v", context.DeadlineExceeded, err)
	}
}
