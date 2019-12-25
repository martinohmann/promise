package instrumented

import "time"

type instrumentedPromise struct {
	delegate        Promise
	instrumentation *Instrumentation
	uuid            string
}

func (p *instrumentedPromise) handle(startTime time.Time, callerInfo CallerInfo, subjectInfo SubjectInfo) {
	invocation := &Invocation{
		StartTime:   startTime,
		EndTime:     time.Now(),
		UUID:        p.uuid,
		Promise:     p.delegate,
		SubjectInfo: subjectInfo,
		CallerInfo:  callerInfo,
	}

	for _, handler := range p.instrumentation.Handlers() {
		handler(invocation)
	}
}

func (p *instrumentedPromise) wrap(candidate Promise) Promise {
	if candidate == p.delegate {
		// If the candidate is the delegate of p we must not wrap it again to
		// avoid generating a new UUID for it which would make tracing
		// impossible. We can just go ahead and return the already instrumented
		// p.
		return p
	}

	// Wrap it and set the UUID of p on the wrapped promise.
	return p.instrumentation.wrap(candidate, func() string {
		return p.uuid
	})
}

func (p *instrumentedPromise) onRejectedFunc(onRejected OnRejectedFunc) OnRejectedFunc {
	callerInfo := getCallerInfo(3)

	return func(err error) (res Value) {
		defer func(startTime time.Time, callerInfo CallerInfo, err error) {
			p.handle(startTime, callerInfo, SubjectInfo{
				Subject:      "onRejected",
				Arguments:    err,
				ReturnValues: res,
			})
		}(time.Now(), callerInfo, err)
		res = onRejected(err)
		return
	}
}

// Then implements the Then method of the promise.Promise interface.
func (p *instrumentedPromise) Then(onFulfilled OnFulfilledFunc, onRejected ...OnRejectedFunc) Promise {
	instrumentedOnRejected := make([]OnRejectedFunc, len(onRejected))
	for i, fn := range onRejected {
		instrumentedOnRejected[i] = p.onRejectedFunc(fn)
	}

	callerInfo := getCallerInfo(2)

	return p.wrap(p.delegate.Then(func(val Value) (res Value) {
		defer func(startTime time.Time, callerInfo CallerInfo, val Value) {
			p.handle(startTime, callerInfo, SubjectInfo{
				Subject:      "onFulfilled",
				Arguments:    val,
				ReturnValues: res,
			})
		}(time.Now(), callerInfo, val)
		res = onFulfilled(val)
		return
	}, instrumentedOnRejected...))
}

// Catch implements the Catch method of the promise.Promise interface.
func (p *instrumentedPromise) Catch(onRejected OnRejectedFunc) Promise {
	return p.wrap(p.delegate.Catch(p.onRejectedFunc(onRejected)))
}

// Finally implements the Finally method of the promise.Promise interface.
func (p *instrumentedPromise) Finally(fn func()) Promise {
	callerInfo := getCallerInfo(2)

	return p.wrap(p.delegate.Finally(func() {
		defer func(startTime time.Time, callerInfo CallerInfo) {
			p.handle(startTime, callerInfo, SubjectInfo{
				Subject: "onFinally",
			})
		}(time.Now(), callerInfo)
		fn()
	}))
}

// Await implements the Await method of the promise.Promise interface.
func (p *instrumentedPromise) Await() (val Value, err error) {
	defer func(startTime time.Time, callerInfo CallerInfo) {
		p.handle(startTime, callerInfo, SubjectInfo{
			Subject:      "Await",
			ReturnValues: []interface{}{val, err},
		})
	}(time.Now(), getCallerInfo(2))
	val, err = p.delegate.Await()
	return
}
