package random

import "testing"

// The first ten values of the MSVC CRT rand() after srand(1), as the CRT
// documents its recurrence; written out independently of Rand.
var msvcFromOne = []int32{41, 18467, 6334, 26500, 19169, 15724, 11478, 29358, 26962, 24464}

func TestMSVCFirstOutputsFromSeedOne(t *testing.T) {
	r := MSVC{State: OriginalStartSeed}
	for i, want := range msvcFromOne {
		if got := r.Rand(); got != want {
			t.Fatalf("draw %d = %d, want %d", i+1, got, want)
		}
	}
}

func TestMSVCRecurrence(t *testing.T) {
	r := MSVC{State: 0xdeadbeef}
	state := uint32(0xdeadbeef)
	for range 1000 {
		state = state*214013 + 2531011
		if got, want := r.Rand(), int32(state>>16)&0x7fff; got != want || r.State != state {
			t.Fatalf("got %d state %#x, want %d state %#x", got, r.State, want, state)
		}
	}
}

func TestRangeZeroWidthDrawsNothing(t *testing.T) {
	r := MSVC{State: 77}
	if got := r.Range(0); got != 0 || r.State != 77 {
		t.Fatalf("Range(0) = %d, state %d; want 0 with no draw", got, r.State)
	}
	if got := r.RangeFrom1(1); got != 1 || r.State != 77 {
		t.Fatalf("RangeFrom1(1) = %d, state %d; want 1 with no draw", got, r.State)
	}
	if got := r.Range(-1); got != 0 || r.State == 77 {
		t.Fatalf("Range(-1) = %d, state %d; want 0 after one draw", got, r.State)
	}
}

func TestRangeIsFloorOfScaledDraw(t *testing.T) {
	for _, n := range []int32{1, 5, 100, 0x7fff, 65537} {
		r := MSVC{State: 9}
		probe := r
		for range 200 {
			raw := probe.Rand()
			want := int32(int64(raw) * int64(n+1) / 32768)
			if got := r.Range(n); got != want || got < 0 || got > n {
				t.Fatalf("Range(%d) = %d, want %d", n, got, want)
			}
		}
	}
}

func TestRangeOfRandMaxIsRawRand(t *testing.T) {
	r, probe := MSVC{State: 5}, MSVC{State: 5}
	for range 100 {
		if got, want := r.Range(RandMax), probe.Rand(); got != want {
			t.Fatalf("Range(RandMax) = %d, want raw %d", got, want)
		}
	}
}

func TestFloatWrapper(t *testing.T) {
	r, probe := MSVC{State: 3}, MSVC{State: 3}
	if got, want := r.Float(), float64(probe.Rand())/32767.0; got != want {
		t.Fatalf("Float = %v, want %v", got, want)
	}
}

func TestSplitMixFromZero(t *testing.T) {
	r := SplitMix64{}
	want := []uint64{0xe220a8397b1dcdaf, 0x6e789e6aa1b965f4, 0x06c45d188009454f, 0xf88bb8a8724c81ec}
	for i, w := range want {
		if got := r.Next(); got != w {
			t.Fatalf("draw %d = %#x, want %#x", i, got, w)
		}
	}
}

func TestUniformAlwaysDraws(t *testing.T) {
	r := SplitMix64{State: 4}
	if got := r.Uniform(0); got != 0 || r.State != 4+splitMixGamma {
		t.Fatalf("Uniform(0) = %d, state %#x", got, r.State)
	}
	if got := r.Upto(0); got != 0 || r.State != 4+splitMixGamma {
		t.Fatalf("Upto(0) = %d drew", got)
	}
}

func TestStreamsDeriveFromSessionAndName(t *testing.T) {
	a := NewService(Session{Seed: 11})
	b := NewService(Session{Seed: 11})
	c := NewService(Session{Seed: 12})
	var sa, sb, sc, other []int
	for range 16 {
		sa = append(sa, a.Stream(Tavern).Raw())
		sb = append(sb, b.Stream(Tavern).Raw())
		sc = append(sc, c.Stream(Tavern).Raw())
		other = append(other, a.Stream(School).Raw())
	}
	if !equal(sa, sb) {
		t.Fatal("one seed and one name gave two sequences")
	}
	if equal(sa, sc) || equal(sa, other) {
		t.Fatal("a different seed or name gave the same sequence")
	}
}

func TestBeginRestartsStreamsInPlace(t *testing.T) {
	s := NewService(Session{Seed: 3})
	st := s.Stream(Music)
	first := []int{st.Raw(), st.Raw(), st.Raw()}
	s.Begin(Session{Seed: 3})
	again := []int{st.Raw(), st.Raw(), st.Raw()}
	if !equal(first, again) {
		t.Fatalf("restart gave %v, want %v", again, first)
	}
}

func TestStreamSeedOverride(t *testing.T) {
	s := NewService(Session{Seed: 99})
	s.SetStreamSeed(Music, 3)
	g := NewGo(3)
	st := s.Stream(Music)
	for range 8 {
		if got, want := st.Raw(), g.Raw(); got != want {
			t.Fatalf("override stream %d, want %d", got, want)
		}
	}
}

type countingShared struct{ m MSVC }

func (c *countingShared) OriginalRand() int32 { return c.m.Rand() }

func TestOriginalStreamsInterleaveOnOneSharedStream(t *testing.T) {
	s := NewService(Session{Seed: 5, Mode: Original, Shared: 1})
	music, tavern := s.Stream(Music), s.Stream(Tavern)
	probe := MSVC{State: 1}
	for i := range 20 {
		var got int
		if i%3 == 0 {
			got = tavern.Raw()
		} else {
			got = music.Raw()
		}
		if want := int(probe.Rand()); got != want {
			t.Fatalf("draw %d = %d, want shared draw %d", i, got, want)
		}
	}
	birds := s.Stream(AmbientBirds)
	before := s.SharedState()
	birds.Raw()
	if s.SharedState() != before {
		t.Fatal("an own stream moved the shared stream")
	}
	w := &countingShared{m: MSVC{State: 42}}
	s.Bind(w)
	at := s.SharedState()
	music.Raw()
	if s.SharedState() != at || w.m.State == 42 {
		t.Fatal("a bound mission did not take the draw")
	}
	s.Unbind(w.m.State)
	if s.SharedState() != w.m.State {
		t.Fatal("unbind did not take the mission's state back")
	}
}

func TestOriginalIntnIsRangeWrapper(t *testing.T) {
	s := NewService(Session{Mode: Original, Shared: 8})
	probe := MSVC{State: 8}
	st := s.Stream(ShopStock)
	for _, n := range []int{1, 2, 7, 100} {
		if got, want := st.Intn(n), int(probe.Range(int32(n-1))); got != want {
			t.Fatalf("Intn(%d) = %d, want %d", n, got, want)
		}
	}
	if s.SharedState() != probe.State {
		t.Fatal("draw counts differ from the range wrapper's")
	}
}

func TestReseedFollowsSeedSiteAndState(t *testing.T) {
	if ReseedValue(1, MissionLoader, 5) != ReseedValue(1, MissionLoader, 5) {
		t.Fatal("reseed is not a function of its inputs")
	}
	if ReseedValue(1, MissionLoader, 5) == ReseedValue(2, MissionLoader, 5) ||
		ReseedValue(1, MissionLoader, 5) == ReseedValue(1, AIManager, 5) ||
		ReseedValue(1, MissionLoader, 5) == ReseedValue(1, MissionLoader, 6) {
		t.Fatal("reseed ignores an input")
	}
}

func equal(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
