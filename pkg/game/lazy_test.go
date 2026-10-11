package game

import (
	"errors"
	"sync"
	"testing"
)

func TestDeferredReadRunsOnceForEveryCopy(t *testing.T) {
	reads := 0
	want := errors.New("the picture will not read")
	value := new(int)
	d := deferredRead(func() (*int, error) { reads++; return value, want })
	a := onFirstRead(d)
	b := a
	if reads != 0 || d.done.Load() {
		t.Fatal("a deferred read ran before anything asked for it")
	}
	if !a.Tried() {
		t.Fatal("a deferred read does not count as attempted")
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if a.Value() != value || b.Err() != want {
				t.Error("a copy answered another value or reason")
			}
		}()
	}
	wg.Wait()
	if reads != 1 || !d.done.Load() {
		t.Fatalf("reads = %d, done = %v; want one read", reads, d.done.Load())
	}
	if r := resolved(value, want); r.Value() != value || r.Err() != want || !r.Tried() {
		t.Fatal("a resolved lazy no longer answers its own value and reason")
	}
}
