// Package instrumented is a drop in replacement for the promise package to
// instrument promises for debugging, tracing and logging of invocations of the
// promise resolution handlers.
package instrumented

import (
	"runtime"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/martinohmann/promise"
)

// InstrumentationHandlerFunc is the signature of a func that can be used as a promise
// invocation handler. It is called with an invocation info every time an
// onFulfilled, onRejected, onFinally handler or Await is called on a promise.
type InstrumentationHandlerFunc func(invocation *Invocation)

// Invocation is a container for information relevant to a given promise
// handler invocation.
type Invocation struct {
	// UUID is a unique string that is automatically generated for every
	// promise that is instrumented to keep track of it.
	UUID string

	// Promise is a pointer to the original promise that is wrapped by the
	// instrumentation. It is strongly advised against manipulating the promise
	// (e.g. calling Then or Await) inside an invocation handler as this may
	// cause weird side effects, panics or even deadlocks. This is only exposed
	// to be able to inspect the promise for debugging or tracing.
	Promise Promise

	// Subject is the subject of the invocation, that is: a function name (e.g.
	// Await) or a handler type (e.g. onFulfilled).
	Subject string

	// SubjectInfo contains information about the subject of the invocation.
	// This is usually a promise resolution handler like onFulfilled or the
	// Await method of a promise.
	SubjectInfo SubjectInfo

	// CallerInfo contains info about the callsite of Subject. In case of
	// onFulfilled, onRejected and onFinally handlers this contains the file,
	// line and func where the handler was passed to Then, Catch or Finally and
	// not the direct caller as this would point to internals of the promise
	// implementation which is generally not useful to the user.
	CallerInfo CallerInfo

	// StartTime holds the time Subject was called at.
	StartTime time.Time

	// EndTime holds the time at which the execution of Subject was finished.
	EndTime time.Time
}

// SubjectInfo contains information about the subject on which an
// instrumentation handler is invoked.
type SubjectInfo struct {
	// Subject is the subject of the invocation, that is: a function name (e.g.
	// Await) or a handler type (e.g. onFulfilled).
	Subject string

	// Arguments hold the argument values that Subject was called with.
	Arguments interface{}

	// ReturnValues hold the values returned by Subject.
	ReturnValues interface{}
}

// CallerInfo contains information about a call site.
type CallerInfo struct {
	// File in which the call happened.
	File string

	// Func contains the name of the the func surrounding the call site.
	Func string

	// Line number of the call site.
	Line int
}

func getCallerInfo(skipFrames int) CallerInfo {
	pc, file, line, _ := runtime.Caller(skipFrames)

	return CallerInfo{
		File: file,
		Func: runtime.FuncForPC(pc).Name(),
		Line: line,
	}
}

var defaultInstrumentation = NewInstrumentation()

// Instrumentation is a factory type for instrumented promises. It holds
// references to instrumentation handlers that should be attached to every
// instrumented promise created by the factory. The factory is useful as a
// drop-in replacement for calls to promise.New, promise.Resolve and
// promise.Reject.
type Instrumentation struct {
	sync.RWMutex
	handlers []InstrumentationHandlerFunc
}

// NewInstrumentation creates a new instrumented promise factory with given
// handler funcs. If no handler funcs are provided, calling any of the factory
// methods returns promises without instrumenting them.
func NewInstrumentation(handlers ...InstrumentationHandlerFunc) *Instrumentation {
	return &Instrumentation{
		handlers: handlers,
	}
}

// AddHandlers adds handler funcs to the instrumentation. Newly added handlers
// are also attached to promises previously created by this instrumentation.
func (i *Instrumentation) AddHandlers(handlers ...InstrumentationHandlerFunc) {
	i.Lock()
	defer i.Unlock()

	i.handlers = append(i.handlers, handlers...)
}

// RemoveHandlers removes all handlers from the instrumentation. After calling
// this function, all promises newly created by this package will not be
// instrumented. This can be used to conditionally disable instrumentation
// without altering the code using the promises.
func (i *Instrumentation) RemoveHandlers() {
	i.Lock()
	defer i.Unlock()

	i.handlers = nil
}

// Handlers returns the handlers configured for i. This method is thread-safe.
func (i *Instrumentation) Handlers() []InstrumentationHandlerFunc {
	i.RLock()
	defer i.RUnlock()

	handlers := i.handlers
	return handlers
}

// New creates a new instrumented promise. It creates a new promise by calling
// promise.New(fn). If the instrumentation has no handlers configured, the new
// promise is returned without wrapping it with instrumentation.
func (i *Instrumentation) New(fn ResolutionFunc) Promise {
	return i.Wrap(promise.New(fn))
}

// Resolve returns a new instrumented promise that is fulfilled with given
// value. It creates a new fulfilled promise by calling promise.Resolve(val).
// If the instrumentation has no handlers configured, the fulfilled promise is
// returned without wrapping it with instrumentation.
func (i *Instrumentation) Resolve(val Value) Promise {
	return i.Wrap(promise.Resolve(val))
}

// Reject returns a new instrumented promise that is rejected with given error
// reason. It creates a new rejected promise by calling promise.Reject(err). If
// the instrumentation has no handlers configured, the rejected promise is
// returned without wrapping it with instrumentation.
func (i *Instrumentation) Reject(err error) Promise {
	return i.Wrap(promise.Reject(err))
}

// Wrap instruments an existing promise. If the instrumentation has no handlers
// configured, the original promise is returned without wrapping it with
// instrumentation. If the provided promise is already instrumented, the newly
// wrapped promise will adpot the UUID of the delegate.
func (i *Instrumentation) Wrap(delegate Promise) Promise {
	if len(i.Handlers()) == 0 {
		// If the instrumentation has no handlers there is no point in wrapping
		// the delegate promise as this would just introduce unnecessary
		// overhead. Return the uninstrumented delegate here also has the
		// benefit that we can always use the instrumented package in
		// production code even if we do not specify any handlers as it does
		// not add any overhead in this case.
		return delegate
	}

	// Handle already instrumented promises.
	if instrumented, ok := delegate.(*instrumentedPromise); ok {
		if instrumented.instrumentation == i {
			// If delegate is already instrumented with i we must not wrap it
			// again or we will end up calling the same instrumentation
			// handlers multiple times.
			return instrumented
		}

		// The delegate is instrumented with a different instrumentation. Wrap
		// it to keep the existing instrumentation in place but adopt its UUID
		// to keep track of it.
		return &instrumentedPromise{
			delegate:        instrumented,
			uuid:            instrumented.uuid,
			instrumentation: i,
		}
	}

	// Wrap it and assign a new UUID.
	return &instrumentedPromise{
		delegate:        delegate,
		uuid:            uuid.New().String(),
		instrumentation: i,
	}
}

// New creates a new promise using the default instrumentation.
func New(fn ResolutionFunc) Promise {
	return defaultInstrumentation.New(fn)
}

// Resolve returns a new promise that is fulfilled with given
// value using the default instrumentation.
func Resolve(val Value) Promise {
	return defaultInstrumentation.Resolve(val)
}

// Reject returns a new promise that is rejected with given error
// reason using the default instrumentation.
func Reject(err error) Promise {
	return defaultInstrumentation.Reject(err)
}

// Wrap instruments an existing promise using the default instrumentation.
func Wrap(p Promise) Promise {
	return defaultInstrumentation.Wrap(p)
}

// AddInstrumentationHandlers adds handlers to the default instrumentation.
func AddInstrumentationHandlers(handlers ...InstrumentationHandlerFunc) {
	defaultInstrumentation.AddHandlers(handlers...)
}

// RemoveInstrumentationHandlers removes all handlers from the default
// instrumentation. After calling this function, all promises newly created by
// this package will not be instrumented.
func RemoveInstrumentationHandlers() {
	defaultInstrumentation.RemoveHandlers()
}
