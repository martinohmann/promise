package promise

import (
	"errors"
	"fmt"
	"testing"
)

func TestNew(t *testing.T) {
	p := New(nil)

	if p == nil {
		t.Fatalf("did not return promise")
	}
}

func TestPromise_Then(t *testing.T) {
	p := New(func(resolve ResolveFunc, _ RejectFunc) {
		resolve(2)
	})

	calls := 0

	p.Then(func(val Value) Value {
		calls++
		if val.(int) != 2 {
			t.Fatalf("expected 2, but got %v", val)
		}

		return val.(int) + 1
	}).Then(func(val Value) Value {
		calls++
		return val
	})

	val, err := p.Await()
	if err != nil {
		t.Fatalf("Await returned unexpected error: %v", err)
	}

	if val.(int) != 3 {
		t.Fatalf("expected val of 3, but got %v", val)
	}

	if calls != 2 {
		t.Fatalf("expected 2 calls of Then callbacks, but got %d", calls)
	}
}

func TestPromise_Catch(t *testing.T) {
	p := New(func(_ ResolveFunc, reject RejectFunc) {
		reject(errors.New("foo"))
	})

	calls := 0

	p.Then(func(val Value) Value {
		t.Fatalf("unexpected execution of Then callback with value: %v", val)

		return val
	}).Catch(func(err error) error {
		calls++
		return fmt.Errorf("bar: %v", err)
	})

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "bar: foo"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

	if calls != 1 {
		t.Fatalf("expected 1 call of Catch callbacks, but got %d", calls)
	}
}

func TestPromise_Panic(t *testing.T) {
	p := New(func(resolve ResolveFunc, _ RejectFunc) {
		panic("whoops")
	})

	calls := 0

	p.Catch(func(err error) error {
		calls++
		return fmt.Errorf("recovered: %v", err)
	})

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "recovered: panic while resolving promise: whoops"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}

	if calls != 1 {
		t.Fatalf("expected 1 call of Catch callbacks, but got %d", calls)
	}
}

func TestPromise_ThenPanic(t *testing.T) {
	p := New(func(resolve ResolveFunc, _ RejectFunc) {
		resolve("foo")
	})

	p.Then(func(val Value) Value {
		panic("whoops")
	})

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "panic while resolving promise: whoops"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestPromise_ThenError(t *testing.T) {
	p := New(func(resolve ResolveFunc, _ RejectFunc) {
		resolve("foo")
	})

	p.Then(func(val Value) Value {
		return errors.New("whoops")
	}).Catch(func(err error) error {
		return fmt.Errorf("bar: %v", err)
	})

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "bar: whoops"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestPromise_CatchPanic(t *testing.T) {
	p := New(func(_ ResolveFunc, reject RejectFunc) {
		reject(errors.New("foo"))
	})

	p.Catch(func(err error) error {
		panic("whoops")
	}).Catch(func(err error) error {
		return fmt.Errorf("recovered: %v", err)
	})

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "recovered: panic while resolving promise: whoops"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}
