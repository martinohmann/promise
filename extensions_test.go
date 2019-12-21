package promise

import (
	"errors"
	"reflect"
	"testing"
	"time"
)

func TestRace_Empty(t *testing.T) {
	val, err := awaitWithTimeout(t, Race(), 500*time.Millisecond)
	if err != nil {
		t.Fatalf("expected nil error, got: %#v", err)
	}

	if val != nil {
		t.Fatalf("expected nil value, got %#v", val)
	}
}

func TestRace_Resolve(t *testing.T) {
	promiseA := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(1 * time.Second)
		resolve("foo")
	})

	promiseB := New(func(resolve ResolveFunc, reject RejectFunc) {
		resolve("bar")
	})

	promiseC := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(500 * time.Millisecond)
		reject(errors.New("baz"))
	})

	p := Race(promiseA, promiseB, promiseC)

	val, err := awaitWithTimeout(t, p, 2*time.Second)
	if err != nil {
		t.Fatalf("expected nil error, got: %#v", err)
	}

	expected := "bar"

	if val != expected {
		t.Fatalf("expected value %#v, got %#v", expected, val)
	}
}

func TestRace_Reject(t *testing.T) {
	promiseA := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(1 * time.Second)
		resolve("foo")
	})

	promiseB := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(500 * time.Millisecond)
		reject(errors.New("baz"))
	})

	promiseC := New(func(resolve ResolveFunc, reject RejectFunc) {
		reject(errors.New("bar"))
	})

	p := Race(promiseA, promiseB, promiseC)

	val, err := awaitWithTimeout(t, p, 2*time.Second)

	expectedErr := errors.New("bar")
	if !reflect.DeepEqual(expectedErr, err) {
		t.Fatalf("expected error %#v, got: %#v", expectedErr, err)
	}

	if val != nil {
		t.Fatalf("expected nil value, got %#v", val)
	}
}

func TestAll_Empty(t *testing.T) {
	val, err := awaitWithTimeout(t, All(), 500*time.Millisecond)
	if err != nil {
		t.Fatalf("expected nil error, got: %#v", err)
	}

	expected := []Value{}

	if !reflect.DeepEqual(expected, val) {
		t.Fatalf("expected value %#v, got %#v", expected, val)
	}
}

func TestAll_Resolve(t *testing.T) {
	promiseA := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(50 * time.Millisecond)
		resolve("foo")
	})

	promiseB := New(func(resolve ResolveFunc, reject RejectFunc) {
		resolve("bar")
	})

	promiseC := New(func(resolve ResolveFunc, reject RejectFunc) {
		resolve("baz")
	})

	p := All(promiseC, promiseA, promiseB)

	val, err := awaitWithTimeout(t, p, 2*time.Second)
	if err != nil {
		t.Fatalf("expected nil error, got: %#v", err)
	}

	expected := []Value{"baz", "foo", "bar"}

	if !reflect.DeepEqual(expected, val) {
		t.Fatalf("expected value %#v, got %#v", expected, val)
	}
}

func TestAll_Reject(t *testing.T) {
	promiseA := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(50 * time.Millisecond)
		resolve("foo")
	})

	promiseB := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(10 * time.Millisecond)
		reject(errors.New("bar"))
	})

	promiseC := New(func(resolve ResolveFunc, reject RejectFunc) {
		resolve("baz")
	})

	p := All(promiseC, promiseA, promiseB)

	val, err := awaitWithTimeout(t, p, 2*time.Second)

	expectedErr := errors.New("bar")
	if !reflect.DeepEqual(expectedErr, err) {
		t.Fatalf("expected error %#v, got: %#v", expectedErr, err)
	}

	if val != nil {
		t.Fatalf("expected nil value, got %#v", val)
	}
}

func TestAny_Empty(t *testing.T) {
	val, err := awaitWithTimeout(t, Any(), 500*time.Millisecond)
	if err != nil {
		t.Fatalf("expected nil error, got: %#v", err)
	}

	if val != nil {
		t.Fatalf("expected nil value, got %#v", val)
	}
}

func TestAny_Resolve(t *testing.T) {
	promiseA := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(50 * time.Millisecond)
		resolve("foo")
	})

	promiseB := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(10 * time.Millisecond)
		resolve("bar")
	})

	promiseC := New(func(resolve ResolveFunc, reject RejectFunc) {
		reject(errors.New("baz"))
	})

	p := Any(promiseC, promiseA, promiseB)

	val, err := awaitWithTimeout(t, p, 2*time.Second)
	if err != nil {
		t.Fatalf("expected nil error, got: %#v", err)
	}

	expected := "bar"

	if !reflect.DeepEqual(expected, val) {
		t.Fatalf("expected value %#v, got %#v", expected, val)
	}
}

func TestAny_Reject(t *testing.T) {
	promiseA := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(50 * time.Millisecond)
		reject(errors.New("foo"))
	})

	promiseB := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(10 * time.Millisecond)
		reject(errors.New("bar"))
	})

	promiseC := New(func(resolve ResolveFunc, reject RejectFunc) {
		reject(errors.New("baz"))
	})

	p := Any(promiseC, promiseA, promiseB)

	val, err := awaitWithTimeout(t, p, 2*time.Second)

	expectedErr := AggregateError{
		errors.New("baz"),
		errors.New("foo"),
		errors.New("bar"),
	}
	if !reflect.DeepEqual(expectedErr, err) {
		t.Fatalf("expected error %#v, got: %#v", expectedErr, err)
	}

	if val != nil {
		t.Fatalf("expected nil value, got %#v", val)
	}
}

func TestAllSettled_Empty(t *testing.T) {
	val, err := awaitWithTimeout(t, AllSettled(), 500*time.Millisecond)
	if err != nil {
		t.Fatalf("expected nil error, got: %#v", err)
	}

	expected := []Result{}

	if !reflect.DeepEqual(expected, val) {
		t.Fatalf("expected value %#v, got %#v", expected, val)
	}
}

func TestAllSettled(t *testing.T) {
	promiseA := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(50 * time.Millisecond)
		resolve("foo")
	})

	promiseB := New(func(resolve ResolveFunc, reject RejectFunc) {
		resolve("bar")
	})

	promiseC := New(func(resolve ResolveFunc, reject RejectFunc) {
		time.Sleep(60 * time.Millisecond)
		reject(errors.New("baz"))
	})

	promiseD := New(func(resolve ResolveFunc, reject RejectFunc) {
		reject(errors.New("qux"))
	})

	p := AllSettled(promiseC, promiseD, promiseB, promiseA)

	val, err := awaitWithTimeout(t, p, 2*time.Second)
	if err != nil {
		t.Fatalf("expected nil error, got: %#v", err)
	}

	expected := []Result{
		{Err: errors.New("baz")},
		{Err: errors.New("qux")},
		{Value: "bar"},
		{Value: "foo"},
	}

	if !reflect.DeepEqual(expected, val) {
		t.Fatalf("expected value %#v, got %#v", expected, val)
	}
}

func TestAggregateError(t *testing.T) {
	one := AggregateError{errors.New("foo")}

	expected := "foo"
	if one.Error() != expected {
		t.Fatalf("expected %q, got: %q", expected, one.Error())
	}

	multi := AggregateError{errors.New("foo"), errors.New("bar"), errors.New("baz")}

	expected = `3 promises rejected due to errors:
* foo
* bar
* baz`
	if multi.Error() != expected {
		t.Fatalf("expected %q, got: %q", expected, multi.Error())
	}
}
