package promise

import (
	"context"
	"fmt"
	"regexp"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_invalidConcurrency_Panic(t *testing.T) {
	fns := make(chan func() Promise)

	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()

	NewPool(0, fns)
}

func TestPool(t *testing.T) {
	fns := make(chan func() Promise)

	var fulfilled int64

	pool := NewPool(10, fns)

	go func() {
		defer close(fns)

		for i := 0; i < 10; i++ {
			fns <- func() Promise {
				return Resolve(nil).Then(func(val Value) Value {
					atomic.AddInt64(&fulfilled, 1)
					return val
				})
			}
		}
	}()

	p := pool.Run(context.Background())

	_, err := awaitWithTimeout(t, p, 2*time.Second)
	if err != nil {
		t.Fatalf("expected nil err but got: %v", err)
	}

	if fulfilled != 10 {
		t.Fatalf("expected 10 promises to be fulfilled but got %d", fulfilled)
	}
}

func TestPool_RunTwicePanic(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("expected panic")
		}
	}()
	fns := make(chan func() Promise)

	pool := NewPool(10, fns)
	pool.Run(context.Background())
	pool.Run(context.Background())
}

func TestPool_Error(t *testing.T) {
	fns := make(chan func() Promise)

	pool := NewPool(10, fns)

	go func() {
		defer close(fns)

		for i := 0; i < 10; i++ {
			fns <- func(val int) func() Promise {
				return func() Promise {
					if val < 5 {
						return Resolve(nil)
					}

					return Reject(fmt.Errorf("error in %d", val))
				}
			}(i)
		}
	}()

	p := pool.Run(context.Background())

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
	fns := make(chan func() Promise)

	pool := NewPool(10, fns)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		defer close(fns)
		fns <- func() Promise {
			return New(func(resolve ResolveFunc, reject RejectFunc) {
				time.Sleep(500 * time.Millisecond)
				resolve("done")
			})
		}
	}()

	p := pool.Run(ctx)

	go func() {
		<-time.After(50 * time.Millisecond)
		cancel()
	}()

	_, err := awaitWithTimeout(t, p, 2*time.Second)
	if err == nil {
		t.Fatal("expected error but got nil")
	}
}

func TestPool_AddEventListener(t *testing.T) {
	fns := make(chan func() Promise)

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

	pool := NewPool(10, fns)

	// double add on purpose
	pool.AddEventListener(listener)
	pool.AddEventListener(listener)

	go func() {
		defer close(fns)

		for i := 0; i < 10; i++ {
			fns <- func(val int) func() Promise {
				return func() Promise {
					if val < 5 {
						return Resolve(nil)
					}

					return Reject(fmt.Errorf("error in %d", val))
				}
			}(i)
		}
	}()

	p := pool.Run(context.Background())

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

func TestPool_RemoveEventListener(t *testing.T) {
	fns := make(chan func() Promise)

	listener := &PoolEventListener{
		OnFulfilled: func(val Value) {
			t.Fatalf("unexpected call to OnFulfilled with value: %v", val)
		},
		OnRejected: func(err error) {
			t.Fatalf("unexpected call to OnRejected with value: %v", err)
		},
	}

	pool := NewPool(10, fns)
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
