package promise

import (
	"errors"
	"fmt"
	"reflect"
	"testing"
)

func TestNew(t *testing.T) {
	p := New(nil)

	if p == nil {
		t.Fatalf("did not return promise")
	}
}

func TestResolve(t *testing.T) {
	p := Resolve(2)
	if p == nil {
		t.Fatalf("did not return promise")
	}

	q := Resolve(p)
	if p != q {
		t.Fatalf("did not flatten resolved promise")
	}
}

func TestPromise_ResolvedThen(t *testing.T) {
	p := Resolve(2)

	p.Await()

	p = p.Then(func(val Value) Value { return val.(int) + 1 })

	val, err := p.Await()
	if err != nil {
		t.Fatalf("Await returned unexpected error: %v", err)
	}

	if val.(int) != 3 {
		t.Fatalf("expected val of 3, but got %v", val)
	}
}

func TestPromise_RejectedThenCatchPromise(t *testing.T) {
	p := Reject(errors.New("foo"))

	p.Await()

	p = p.
		Then(func(val Value) Value { t.Fatal("then called although it should not"); return nil }).
		Catch(func(err error) Value { return Resolve(5) })

	val, err := p.Await()
	if err != nil {
		t.Fatalf("Await returned unexpected error: %v", err)
	}

	if val.(int) != 5 {
		t.Fatalf("expected val of 5, but got %v", val)
	}
}

func TestPromise_RejectedThenCatchStruct(t *testing.T) {
	p := Reject(errors.New("foo"))

	p.Await()

	p = p.
		Then(func(val Value) Value { t.Fatal("then called although it should not"); return nil }).
		Catch(func(err error) Value { return struct{ name string }{name: "bar"} })

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "{bar}"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestPromise_RejectedThenCatch(t *testing.T) {
	p := Reject(errors.New("foo"))

	p.Await()

	p = p.
		Then(func(val Value) Value { t.Fatal("then called although it should not"); return nil }).
		Catch(func(err error) Value { return fmt.Errorf("foo %v", err) })

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "foo foo"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestPromise_Then(t *testing.T) {
	calls := 0

	p := New(func(resolve ResolveFunc, _ RejectFunc) {
		resolve(2)
	}).Then(func(val Value) Value {
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
	calls := 0

	p := New(func(_ ResolveFunc, reject RejectFunc) {
		reject(errors.New("foo"))
	}).Then(func(val Value) Value {
		t.Fatalf("unexpected execution of Then callback with value: %v", val)

		return val
	}).Catch(func(err error) Value {
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
	calls := 0

	p := New(func(resolve ResolveFunc, _ RejectFunc) {
		panic("whoops")
	}).Catch(func(err error) Value {
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
	}).Then(func(val Value) Value {
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
	}).Then(func(val Value) Value {
		return errors.New("whoops")
	}).Catch(func(err error) Value {
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

func TestPromise_ThenPromise(t *testing.T) {
	p := New(func(resolve ResolveFunc, _ RejectFunc) {
		resolve("foo")
	}).Then(func(val Value) Value {
		return Resolve("bar").Then(func(val2 Value) Value {
			return val.(string) + val2.(string)
		})
	})

	val, err := p.Await()
	if err != nil {
		t.Fatalf("Await returned unexpected error: %#v", err)
	}

	expected := "foobar"

	if !reflect.DeepEqual(val, expected) {
		t.Fatalf("expected %q, but got %#v", expected, val)
	}
}

func TestPromise_ThenPromiseReject(t *testing.T) {
	p := New(func(resolve ResolveFunc, _ RejectFunc) {
		resolve("foo")
	}).Then(func(val Value) Value {
		return Reject(errors.New("bar"))
	})

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "bar"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestPromise_CatchPanic(t *testing.T) {
	p := New(func(_ ResolveFunc, reject RejectFunc) {
		reject(errors.New("foo"))
	}).Catch(func(err error) Value {
		panic("whoops")
	}).Catch(func(err error) Value {
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

func TestPromise_CatchString(t *testing.T) {
	p := New(func(_ ResolveFunc, reject RejectFunc) {
		reject(errors.New("foo"))
	}).Catch(func(err error) Value {
		return "barbaz"
	})

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "barbaz"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestPromise_CatchStruct(t *testing.T) {
	p := New(func(_ ResolveFunc, reject RejectFunc) {
		reject(errors.New("foo"))
	}).Catch(func(err error) Value {
		return struct{ one, two string }{one: "bar", two: "baz"}
	})

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "{bar baz}"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}

func TestPromise_CatchPromise(t *testing.T) {
	p := New(func(_ ResolveFunc, reject RejectFunc) {
		reject(errors.New("foo"))
	}).Catch(func(err error) Value {
		return Resolve("bar")
	})

	val, err := p.Await()
	if err != nil {
		t.Fatalf("Await returned unexpected error: %#v", err)
	}

	expected := "bar"

	if !reflect.DeepEqual(val, expected) {
		t.Fatalf("expected %q, but got %#v", expected, val)
	}
}

func TestPromise_CatchPromiseReject(t *testing.T) {
	p := New(func(_ ResolveFunc, reject RejectFunc) {
		reject(errors.New("foo"))
	}).Catch(func(err error) Value {
		return Reject(errors.New("bar"))
	})

	_, err := p.Await()
	if err == nil {
		t.Fatal("Await did not return expected error, got nil")
	}

	expectedErr := "bar"

	if err.Error() != expectedErr {
		t.Fatalf("expected error %q, got %q", expectedErr, err.Error())
	}
}
