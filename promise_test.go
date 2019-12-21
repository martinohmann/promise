package promise

import (
	"errors"
	"fmt"
	"reflect"
	"strconv"
	"testing"
	"time"
)

func awaitWithTimeout(t *testing.T, p Promise, timeout time.Duration) (val Value, err error) {
	done := make(chan struct{})

	go func() {
		defer close(done)
		val, err = p.Await()
	}()

	select {
	case <-done:
		return
	case <-time.After(timeout):
		t.Fatal("timeout exceeded")
	}

	return
}

func TestNew(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("New did not panic on nil resolution func")
		}
	}()

	New(nil)
}

func TestPromise(t *testing.T) {
	tests := []struct {
		setup       func(t *testing.T) Promise
		name        string
		expectedErr error
		expected    Value
	}{
		{
			name:     "simple Then chain",
			expected: "3",
			setup: func(t *testing.T) Promise {
				return New(func(resolve ResolveFunc, _ RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					resolve(2)
				}).Then(func(val Value) Value {
					return val.(int) + 1
				}).Then(func(val Value) Value {
					return fmt.Sprintf("%d", val.(int))
				})
			},
		},
		{
			name:        "rejected promise does not trigger Then callbacks",
			expectedErr: errors.New("bar: foo"),
			setup: func(t *testing.T) Promise {
				return New(func(_ ResolveFunc, reject RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					reject(errors.New("foo"))
				}).Then(func(val Value) Value {
					t.Fatalf("unexpected execution of Then callback with value: %v", val)

					return val
				}).Catch(func(err error) Value {
					return fmt.Errorf("bar: %v", err)
				})
			},
		},
		{
			name:        "errors in Then callbacks are converted into a rejected promise",
			expectedErr: errors.New("whoops"),
			setup: func(t *testing.T) Promise {
				return New(func(resolve ResolveFunc, _ RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					resolve("foo")
				}).Then(func(val Value) Value {
					return errors.New("whoops")
				})
			},
		},
		{
			name:     "resolved promises returned by Then callbacks are evaluated",
			expected: "foobar",
			setup: func(t *testing.T) Promise {
				return New(func(resolve ResolveFunc, _ RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					resolve("foo")
				}).Then(func(val Value) Value {
					return Resolve("bar").Then(func(val2 Value) Value {
						return val.(string) + val2.(string)
					})
				})
			},
		},
		{
			name:        "rejected promises returned by Then callbacks are evaluated",
			expectedErr: errors.New("bar"),
			setup: func(t *testing.T) Promise {
				return New(func(resolve ResolveFunc, _ RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					resolve("foo")
				}).Then(func(val Value) Value {
					return Reject(errors.New("bar"))
				})
			},
		},
		{
			name:        "panics in resolution func are recovered and converted into a rejected promise",
			expectedErr: errors.New("panic during promise resolution: whoops"),
			setup: func(t *testing.T) Promise {
				return New(func(_ ResolveFunc, _ RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					panic("whoops")
				})
			},
		},
		{
			name:        "panics in Then callbacks are recovered and converted into a rejected promise",
			expectedErr: errors.New("panic during promise resolution: oops"),
			setup: func(t *testing.T) Promise {
				return New(func(resolve ResolveFunc, _ RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					resolve("myvalue")
				}).Then(func(val Value) Value {
					panic("oops")
				})
			},
		},
		{
			name:        "panics in Then callbacks of a resolved promise are recovered and converted into a rejected promise",
			expectedErr: errors.New("panic during promise resolution: oops"),
			setup: func(t *testing.T) Promise {
				p := Resolve("myvalue")

				awaitWithTimeout(t, p, 100*time.Millisecond)

				return p.Then(func(val Value) Value {
					panic("oops")
				})
			},
		},
		{
			name:        "panics in Catch callbacks are recovered and converted into a rejected promise",
			expectedErr: errors.New("panic during promise resolution: oopsie"),
			setup: func(t *testing.T) Promise {
				return New(func(_ ResolveFunc, reject RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					reject(errors.New("someerror"))
				}).Catch(func(err error) Value {
					panic("oopsie")
				})
			},
		},
		{
			name:        "panics in Catch callbacks of a rejected promise are recovered and converted into a rejected promise",
			expectedErr: errors.New("panic during promise resolution: oopsie"),
			setup: func(t *testing.T) Promise {
				p := Reject(errors.New("someerror"))

				awaitWithTimeout(t, p, 100*time.Millisecond)

				return p.Catch(func(err error) Value {
					panic("oopsie")
				})
			},
		},
		{
			name:     "rejected promises can be `recovered` with resolved promises in Catch callbacks",
			expected: "bar",
			setup: func(t *testing.T) Promise {
				return New(func(_ ResolveFunc, reject RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					reject(errors.New("foo"))
				}).Catch(func(err error) Value {
					return Resolve("bar")
				})
			},
		},
		{
			name:        "rejected promises evaluate to errors in Catch callbacks",
			expectedErr: errors.New("bar"),
			setup: func(t *testing.T) Promise {
				return New(func(_ ResolveFunc, reject RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					reject(errors.New("foo"))
				}).Catch(func(err error) Value {
					return Reject(errors.New("bar"))
				})
			},
		},
		{
			name:     "non-error types return resolved promise Catch callbacks",
			expected: "bar",
			setup: func(t *testing.T) Promise {
				return New(func(_ ResolveFunc, reject RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					reject(errors.New("foo"))
				}).Catch(func(err error) Value {
					return "bar"
				})
			},
		},
		{
			name:     "resolved promise is immutable and returns new promise when adding Then callback",
			expected: int(3),
			setup: func(t *testing.T) Promise {
				p := Resolve(2)

				awaitWithTimeout(t, p, 500*time.Millisecond)

				q := p.Then(func(val Value) Value {
					return val.(int) + 1
				})
				if p == q {
					t.Fatal("expected promises to be different")
				}

				return q
			},
		},
		{
			name:     "rejected promise is immutable and returns new promise when adding Catch callback",
			expected: int(5),
			setup: func(t *testing.T) Promise {
				p := Reject(errors.New("foo"))

				awaitWithTimeout(t, p, 500*time.Millisecond)

				q := p.Then(func(val Value) Value {
					t.Fatalf("unexpected execution of Then callback with value: %#v", val)
					return nil
				}).Catch(func(err error) Value {
					return Resolve(5)
				})
				if p == q {
					t.Fatal("expected promises to be different")
				}

				return q
			},
		},
		{
			name:     "rejected promise is immutable and returns new promise when adding Catch callback (non-error, non-promise)",
			expected: struct{ name string }{name: "bar"},
			setup: func(t *testing.T) Promise {
				p := Reject(errors.New("foo"))

				awaitWithTimeout(t, p, 500*time.Millisecond)

				q := p.Then(func(val Value) Value {
					t.Fatalf("unexpected execution of Then callback with value: %#v", val)
					return nil
				}).Catch(func(err error) Value {
					return struct{ name string }{name: "bar"}
				})
				if p == q {
					t.Fatal("expected promises to be different")
				}

				return q
			},
		},
		{
			name:        "rejected promise is immutable and returns new promise when adding Catch callback (error)",
			expectedErr: errors.New("foo foo"),
			setup: func(t *testing.T) Promise {
				p := Reject(errors.New("foo"))

				awaitWithTimeout(t, p, 500*time.Millisecond)

				q := p.Then(func(val Value) Value {
					t.Fatalf("unexpected execution of Then callback with value: %#v", val)
					return nil
				}).Catch(func(err error) Value {
					return fmt.Errorf("foo %v", err)
				})
				if p == q {
					t.Fatal("expected promises to be different")
				}

				return q
			},
		},
		{
			name:     "resolve promise with another promise",
			expected: int(4),
			setup: func(t *testing.T) Promise {
				return New(func(resolve ResolveFunc, _ RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					resolve(Resolve(3).Then(func(val Value) Value {
						return val.(int) + 1
					}))
				})
			},
		},
		{
			name:        "resolve promise with another rejected promise",
			expectedErr: errors.New("foo"),
			setup: func(t *testing.T) Promise {
				return New(func(resolve ResolveFunc, _ RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					resolve(Reject(errors.New("foo")).Then(func(val Value) Value {
						t.Fatalf("unexpected execution of Then callback with value: %#v", val)
						return nil
					}))
				})
			},
		},
		{
			name:        "resolve promise with error",
			expectedErr: errors.New("foo"),
			setup: func(t *testing.T) Promise {
				return New(func(resolve ResolveFunc, _ RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					resolve(errors.New("foo"))
				})
			},
		},
		{
			name:     "typed nil error resolves promise",
			expected: nil,
			setup: func(t *testing.T) Promise {
				return New(func(_ ResolveFunc, reject RejectFunc) {
					reject(errors.New("foo"))
				}).Catch(func(err error) Value {
					var err2 error
					return err2
				})
			},
		},
		{
			name:     "deeply nested promise chain",
			expected: "the answer is: 1024",
			setup: func(t *testing.T) Promise {
				return New(func(resolve ResolveFunc, _ RejectFunc) {
					time.Sleep(10 * time.Millisecond)
					resolve(2)
				}).Then(func(val Value) Value {
					return val.(int) * 2
				}).Then(func(val Value) Value {
					return Resolve(val).Then(func(val Value) Value {
						return val.(int) * 2
					}).Catch(func(err error) Value {
						t.Fatalf("unexpected execution of Catch callback with err: %#v", err)
						return err
					}).Then(func(val Value) Value {
						return Reject(fmt.Errorf("%d", val.(int)))
					})
				}).Catch(func(err error) Value {
					val, err := strconv.Atoi(err.Error())
					if err != nil {
						return err
					}

					return Resolve(val)
				}).Then(func(val Value) Value {
					return val.(int) * 2
				}).Catch(func(err error) Value {
					t.Fatalf("unexpected execution of Catch callback with err: %#v", err)
					return err
				}).Then(func(val Value) Value {
					return Resolve(val).Then(func(val Value) Value {
						return val.(int) * 2
					}).Then(func(val Value) Value {
						return Resolve(val).Then(func(val Value) Value {
							return val.(int) * 2
						}).Then(func(val Value) Value {
							return Resolve(val).Then(func(val Value) Value {
								return Resolve(val.(int) * 2).Then(func(val Value) Value {
									return Reject(fmt.Errorf("%d", val.(int)))
								})
							})
						}).Catch(func(err error) Value {
							val, err := strconv.Atoi(err.Error())
							if err != nil {
								return err
							}

							return Resolve(val).Then(func(val Value) Value {
								return val.(int) * 2
							})
						}).Catch(func(err error) Value {
							t.Fatalf("unexpected execution of Catch callback with err: %#v", err)
							return err
						}).Then(func(val Value) Value {
							return Reject(fmt.Errorf("%d", val.(int)*4)).Then(func(val Value) Value {
								t.Fatalf("unexpected execution of Then callback with value: %#v", val)
								return val
							})
						})
					})
				}).Catch(func(err error) Value {
					return fmt.Sprintf("the answer is: %s", err.Error())
				})
			},
		},
		{
			name:     "immutability: a promise cannot be resolved twice",
			expected: "foo",
			setup: func(t *testing.T) Promise {
				return New(func(resolve ResolveFunc, _ RejectFunc) {
					resolve("foo")
					resolve("bar")
				})
			},
		},
		{
			name:        "immutability: a promise cannot be rejected twice",
			expectedErr: errors.New("foo"),
			setup: func(t *testing.T) Promise {
				return New(func(_ ResolveFunc, reject RejectFunc) {
					reject(errors.New("foo"))
					reject(errors.New("bar"))
				})
			},
		},
		{
			name:     "resolve promise with a thenable",
			expected: "foo",
			setup: func(t *testing.T) Promise {
				thenable := ResolutionFunc(func(resolve ResolveFunc, reject RejectFunc) {
					resolve("foo")
				})

				return New(func(resolve ResolveFunc, _ RejectFunc) {
					resolve(thenable)
				})
			},
		},
		{
			name:     "resolve promise with a thenable #2",
			expected: "foo",
			setup: func(t *testing.T) Promise {
				thenable := ResolutionFunc(func(resolve ResolveFunc, reject RejectFunc) {
					resolve("foo")
				})

				return Resolve(2).Then(func(val Value) Value {
					return thenable
				})
			},
		},
		{
			name:        "reject promise with a thenable",
			expectedErr: errors.New("foo"),
			setup: func(t *testing.T) Promise {
				thenable := ResolutionFunc(func(resolve ResolveFunc, reject RejectFunc) {
					reject(errors.New("foo"))
				})

				return Reject(errors.New("bar")).Catch(func(err error) Value {
					return thenable
				})
			},
		},
		{
			name:     "recover rejected promise with a thenable",
			expected: "foo",
			setup: func(t *testing.T) Promise {
				thenable := ResolutionFunc(func(resolve ResolveFunc, reject RejectFunc) {
					resolve("foo")
				})

				return Reject(errors.New("bar")).Catch(func(err error) Value {
					return thenable
				})
			},
		},
		{
			name:        "resolve promise with a thenable which rejects",
			expectedErr: errors.New("foo"),
			setup: func(t *testing.T) Promise {
				thenable := ResolutionFunc(func(resolve ResolveFunc, reject RejectFunc) {
					reject(errors.New("foo"))
				})

				return New(func(resolve ResolveFunc, _ RejectFunc) {
					resolve(thenable)
				})
			},
		},
		{
			name:        "resolve promise with a thenable which rejects #2",
			expectedErr: errors.New("foo"),
			setup: func(t *testing.T) Promise {
				thenable := ResolutionFunc(func(resolve ResolveFunc, reject RejectFunc) {
					reject(errors.New("foo"))
				})

				return Resolve(2).Then(func(val Value) Value {
					return thenable
				})
			},
		},
		{
			name:        "a promise cannot be resolved with itself",
			expectedErr: ErrCircularResolutionChain,
			setup: func(t *testing.T) Promise {
				p := New(func(resolve ResolveFunc, _ RejectFunc) {
					time.Sleep(100 * time.Millisecond)
					resolve("foo")
				})

				return p.Then(func(val Value) Value {
					return p
				})
			},
		},
		{
			name:        "a promise cannot be resolved with itself #2",
			expectedErr: ErrCircularResolutionChain,
			setup: func(t *testing.T) Promise {
				p := New(func(resolve ResolveFunc, reject RejectFunc) {
					time.Sleep(100 * time.Millisecond)
					reject(errors.New("foo"))
				})

				return p.Catch(func(err error) Value {
					return p
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			p := test.setup(t)

			val, err := awaitWithTimeout(t, p, 2*time.Second)

			switch {
			case test.expectedErr != nil && err == nil:
				t.Fatal("expected error but got nil")
			case test.expectedErr != nil && err != nil:
				if !reflect.DeepEqual(test.expectedErr, err) {
					t.Fatalf("expected error %#v but got %#v", test.expectedErr, err)
				}

				if val != nil {
					t.Fatalf("expected val to be nil but got %#v", val)
				}
			case test.expectedErr == nil && err != nil:
				t.Fatalf("expected no error but got %#v", err)
			case test.expectedErr == nil && err == nil:
				if !reflect.DeepEqual(test.expected, val) {
					t.Fatalf("expected value %#v but got %#v", test.expected, val)
				}
			}

			var expectedState state = fulfilled
			if test.expectedErr != nil {
				expectedState = rejected
			}

			state := p.(*promise).state

			if state != expectedState {
				t.Fatalf("expected promise to be in state %d, but got %d", expectedState, state)
			}
		})
	}
}

func TestPromise_FinallyResolve(t *testing.T) {
	called := 0
	p := Resolve("foo").Finally(func() {
		called++
	})

	val, err := awaitWithTimeout(t, p, 500*time.Millisecond)

	if called != 1 {
		t.Fatalf("expected Finally callback to be invoked once, but got %d", called)
	}

	if err != nil {
		t.Fatalf("expected nil error, got: %#v", err)
	}

	expected := "foo"

	if val != expected {
		t.Fatalf("expected value %#v, got %#v", expected, val)
	}
}

func TestPromise_FinallyReject(t *testing.T) {
	called := 0
	p := Reject(errors.New("foo")).Finally(func() {
		called++
	})

	val, err := awaitWithTimeout(t, p, 500*time.Millisecond)

	if called != 1 {
		t.Fatalf("expected Finally callback to be invoked once, but got %d", called)
	}

	expectedErr := errors.New("foo")
	if !reflect.DeepEqual(expectedErr, err) {
		t.Fatalf("expected error %#v, got: %#v", expectedErr, err)
	}

	if val != nil {
		t.Fatalf("expected nil value, got %#v", val)
	}
}
