package promise

type indexedValue struct {
	index int
	value interface{}
}

func Race(promises ...*Promise) *Promise {
	if len(promises) == 0 {
		return Resolve(nil)
	}

	return New(func(resolve ResolveFunc, reject RejectFunc) {
		valChan := make(chan Value, len(promises))
		errChan := make(chan error, len(promises))

		for _, promise := range promises {
			promise.Then(func(val Value) Value {
				valChan <- val
				return val
			}).Catch(func(err error) Value {
				errChan <- err
				return err
			})
		}

		select {
		case val := <-valChan:
			resolve(val)
		case err := <-errChan:
			reject(err)
		}
	})
}

func All(promises ...*Promise) *Promise {
	if len(promises) == 0 {
		return Resolve([]Value{})
	}

	return New(func(resolve ResolveFunc, reject RejectFunc) {
		resChan := make(chan indexedValue, len(promises))
		errChan := make(chan error, len(promises))

		for i, promise := range promises {
			idx := i

			promise.Then(func(val Value) Value {
				resChan <- indexedValue{idx, val}
				return val
			}).Catch(func(err error) Value {
				errChan <- err
				return err
			})
		}

		results := make([]Value, len(promises))

		for i := 0; i < len(promises); i++ {
			select {
			case res := <-resChan:
				results[res.index] = res.value
			case err := <-errChan:
				reject(err)
				return
			}
		}

		resolve(results)
	})
}

type Result struct {
	Value Value
	Err   error
}

func AllSettled(promises ...*Promise) *Promise {
	if len(promises) == 0 {
		return Resolve([]Result{})
	}

	return New(func(resolve ResolveFunc, _ RejectFunc) {
		resChan := make(chan indexedValue, len(promises))

		for i, promise := range promises {
			idx := i

			promise.Then(func(val Value) Value {
				resChan <- indexedValue{idx, Result{Value: val}}
				return val
			}).Catch(func(err error) Value {
				resChan <- indexedValue{idx, Result{Err: err}}
				return err
			})
		}

		results := make([]Result, len(promises))

		for i := 0; i < len(promises); i++ {
			res := <-resChan
			results[res.index] = res.value.(Result)
		}

		resolve(results)
	})
}
