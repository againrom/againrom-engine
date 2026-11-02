package sim

import "testing"

func TestRuntimeIDSetGivesTheLowestClearBit(t *testing.T) {
	m := NewRuntimeIDSet()
	if !m.Marked(0) || m.LowestClear() != 1 {
		t.Fatal("a new set marks bit 0 only")
	}
	for _, id := range []uint32{1, 2, 4} {
		m.Mark(id)
	}
	if got := m.LowestClear(); got != 3 {
		t.Fatalf("lowest clear = %d, want 3", got)
	}
	m.Mark(0x10003)
	if !m.Marked(3) || m.LowestClear() != 5 {
		t.Fatal("an id is masked to its low 16 bits")
	}
	m.Release(2)
	m.Release(0)
	if m.LowestClear() != 2 || !m.Marked(0) {
		t.Fatal("Release clears a bit but never bit 0")
	}
	for id := uint32(1); id < runtimeIDSpace; id++ {
		m.Mark(id)
	}
	if got := m.LowestClear(); got != 0 {
		t.Fatalf("full set gave %d, want 0", got)
	}
}
