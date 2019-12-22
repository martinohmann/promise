package instrumented

import (
	"errors"
	"reflect"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/martinohmann/promise"
)

func noopHandler(_ *Invocation) {}

func TestInstrumentation_Wrap_noHandlers(t *testing.T) {
	p := promise.Resolve(nil)
	wrapped := Wrap(p)
	if wrapped != p {
		t.Fatal("expected Wrap to return original promise if there are no handlers defined")
	}
}

func TestInstrumentation_Wrap(t *testing.T) {
	instrumented := NewInstrumentation(noopHandler)

	p := promise.Resolve(nil)
	wrapped := instrumented.Wrap(p)
	if wrapped == p {
		t.Fatal("expected Wrap to return new instrumented promise")
	}

	if _, ok := wrapped.(*instrumentedPromise); !ok {
		t.Fatalf("expected Wrap to return %T, got %T", &instrumentedPromise{}, wrapped)
	}
}

func TestInstrumentation_Wrap_doNotDoubleWrapIfInstrumentationIsTheSame(t *testing.T) {
	instrumentedPromise := NewInstrumentation(noopHandler)

	p1 := instrumentedPromise.Resolve(nil)
	p2 := instrumentedPromise.Wrap(p1)

	if p1 != p2 {
		t.Fatalf("expected promises to be the same")
	}
}

func TestInstrumentation_Wrap_adpotUUIDIfInstrumentationDiffers(t *testing.T) {
	instrumentedPromise1 := NewInstrumentation(noopHandler)
	instrumentedPromise2 := NewInstrumentation(noopHandler)

	p1 := instrumentedPromise1.Resolve(nil)
	p2 := instrumentedPromise2.Wrap(p1)

	if p1 == p2 {
		t.Fatalf("expected promises to be different")
	}

	if p1.(*instrumentedPromise).uuid != p2.(*instrumentedPromise).uuid {
		t.Fatalf(
			"expected wrapped promise to have uuid %q, got %q",
			p1.(*instrumentedPromise).uuid,
			p2.(*instrumentedPromise).uuid,
		)
	}
}

type testHandler struct {
	sync.Mutex
	subjects []string
	uuidMap  map[string]bool
}

func newTestHandler() *testHandler {
	return &testHandler{
		subjects: make([]string, 0),
		uuidMap:  make(map[string]bool),
	}
}

func (h *testHandler) Log(invocation *Invocation) {
	h.Lock()
	defer h.Unlock()

	h.uuidMap[invocation.UUID] = true
	h.subjects = append(h.subjects, invocation.SubjectInfo.Subject)
}

func TestInstrumentation(t *testing.T) {
	handler := newTestHandler()
	AddInstrumentationHandlers(handler.Log)
	defer RemoveInstrumentationHandlers()

	p := New(func(resolve ResolveFunc, _ RejectFunc) {
		time.Sleep(10 * time.Millisecond)
		resolve(42)
	}).Then(func(val Value) Value {
		return val.(int) + 1
	}).Then(func(val Value) Value {
		if val.(int) != 42 {
			return errors.New("not 42")
		}

		return val
	}, func(err error) Value {
		val, err := strconv.Atoi(err.Error())
		if err != nil {
			return Reject(err)
		}

		return val
	}).Catch(func(err error) Value {
		return Resolve(43)
	}).Finally(func() {
		// noop
	})

	val, err := p.Await()
	if err != nil {
		t.Fatalf("expected no error but got %#v", err)
	}

	expected := 43
	if !reflect.DeepEqual(expected, val) {
		t.Fatalf("expected value %v, got %v", expected, val)
	}

	expectedSubjects := []string{"onFulfilled", "onFulfilled", "onRejected", "Await", "onRejected", "Await", "onFinally", "Await"}
	if !reflect.DeepEqual(expectedSubjects, handler.subjects) {
		t.Fatalf("expected handled subjects %v, got %v", expectedSubjects, handler.subjects)
	}

	expectedUUIDs := 3
	if len(handler.uuidMap) != expectedUUIDs {
		t.Fatalf("expected %d handled UUIDs, got %d", expectedUUIDs, len(handler.uuidMap))
	}
}

func TestInstrumentedPromise_WrapDelegate(t *testing.T) {
	p := promise.Resolve(42)

	instrumented := &instrumentedPromise{
		instrumentation: defaultInstrumentation,
		delegate:        p,
	}

	if instrumented.wrap(p) != instrumented {
		t.Fatalf("expected promises to be the same")
	}
}

func TestInstrumentedPromise_WrapOther(t *testing.T) {
	p := promise.Resolve(42)
	q := promise.Resolve(23)

	instrumented := &instrumentedPromise{
		instrumentation: defaultInstrumentation,
		delegate:        p,
	}

	if instrumented.wrap(q) == instrumented {
		t.Fatalf("expected promises to be the different")
	}
}
