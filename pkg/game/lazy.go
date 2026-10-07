package game

// lazy is one resolved value, the reason there is none, and whether resolution
// was attempted.
//
// It replaces two idioms FrontEnd wrote out by hand nineteen times. One is an
// exported `X`/`XErr` pair resolved once at construction: a cosmetic asset that
// may be absent, where the reason is kept because an install whose art will not
// read is a fact about that install. The other is an unexported
// `xCache`/`xLoaded` pair resolved on first use, where the FLAG and not the
// value says the load was attempted, so an install missing a picture is read
// once and not once a frame.
//
// Both need the same three facts, so both are this type. Neither is safe for
// concurrent use and neither was before: every caller is on the frame
// goroutine.
type lazy[T any] struct {
	value T
	err   error
	tried bool
}

// Value is the resolved value, or T's zero when resolution failed or has not
// run. A caller that must tell those two apart reads Err or Tried.
func (l lazy[T]) Value() T { return l.value }

// Err is why there is no value. Nil covers both a resolution that succeeded
// and one that has not run.
func (l lazy[T]) Err() error { return l.err }

// Tried reports whether resolution has been attempted. A FAILED ATTEMPT IS
// STILL AN ATTEMPT: the first-use caches set this before they read, so a load
// that fails answers its own zero value forever rather than being retried on
// the next frame.
func (l lazy[T]) Tried() bool { return l.tried }

// resolved is one finished attempt's value and error together. It is how the
// eagerly loaded fields are filled, so a caller cannot reach the value without
// the variable that says why there is not one.
func resolved[T any](v T, err error) lazy[T] { return lazy[T]{value: v, err: err, tried: true} }

// begin records that resolution is being attempted, before it runs. The
// first-use caches call it on their way in, which is what makes a failure
// final rather than repeated.
func (l *lazy[T]) begin() { l.tried = true }

// store keeps a successful resolution's value and returns it.
func (l *lazy[T]) store(v T) T {
	l.value = v
	return l.value
}
