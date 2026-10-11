package game

import (
	"sync"
	"sync/atomic"
)

// lazy is one resolved value, the reason there is none, and whether resolution
// was attempted: resolved at construction, filled on first use through begin
// and store, or read once through a lazyLoad every copy shares. Only lazyLoad
// is safe for concurrent use.
type lazy[T any] struct {
	value T
	err   error
	tried bool
	load  *lazyLoad[T]
}

// lazyLoad runs its read on the first Value or Err.
type lazyLoad[T any] struct {
	once  sync.Once
	read  func() (T, error)
	done  atomic.Bool
	value T
	err   error
}

func deferredRead[T any](read func() (T, error)) *lazyLoad[T] {
	return &lazyLoad[T]{read: read}
}

func (d *lazyLoad[T]) resolve() *lazyLoad[T] {
	d.once.Do(func() {
		d.value, d.err = d.read()
		d.done.Store(true)
	})
	return d
}

// Value is the resolved value, or T's zero when resolution failed or has not
// run. A caller that must tell those two apart reads Err or Tried.
func (l lazy[T]) Value() T {
	if l.load != nil {
		return l.load.resolve().value
	}
	return l.value
}

// Err is why there is no value. Nil covers both a resolution that succeeded
// and one that has not run.
func (l lazy[T]) Err() error {
	if l.load != nil {
		return l.load.resolve().err
	}
	return l.err
}

// Tried reports whether resolution has been attempted. A FAILED ATTEMPT IS
// STILL AN ATTEMPT: the first-use caches set this before they read, so a load
// that fails answers its own zero value forever rather than being retried on
// the next frame.
func (l lazy[T]) Tried() bool { return l.tried }

// resolved is one finished attempt's value and error together. It is how the
// eagerly loaded fields are filled, so a caller cannot reach the value without
// the variable that says why there is not one.
func resolved[T any](v T, err error) lazy[T] { return lazy[T]{value: v, err: err, tried: true} }

func onFirstRead[T any](d *lazyLoad[T]) lazy[T] { return lazy[T]{tried: true, load: d} }

// begin records that resolution is being attempted, before it runs. The
// first-use caches call it on their way in, which is what makes a failure
// final rather than repeated.
func (l *lazy[T]) begin() { l.tried = true }

// store keeps a successful resolution's value and returns it.
func (l *lazy[T]) store(v T) T {
	l.value = v
	return l.value
}
