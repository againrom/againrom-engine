// Package intmath is the one integer helper leaf; it imports nothing here.
package intmath

type Signed interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64
}

// Abs is |v|; the smallest value of its type maps to itself.
func Abs[T Signed](v T) T {
	if v < 0 {
		return -v
	}
	return v
}
