package systemclick

import "testing"

// Read gives a usable setting on every platform: a positive time and
// rectangle, the system's where it states one and Fallback's otherwise.
func TestReadGivesAUsableSetting(t *testing.T) {
	s := Read()
	if s.Time <= 0 || s.Width <= 0 || s.Height <= 0 {
		t.Fatalf("Read() = %+v", s)
	}
	t.Logf("double-click setting: %v, %d by %d pixels", s.Time, s.Width, s.Height)
}
