package mapload

import "testing"

// TestClass8MarkIsTheOneClassifier pins the class-8 mark row to the one
// classifier over every tile word.
func TestClass8MarkIsTheOneClassifier(t *testing.T) {
	var row MarkRow
	for _, r := range SecondMarks.Rows {
		if r.Arm == GroundClass8 {
			row = r
		}
	}
	for w := 0; w < 1<<16; w++ {
		class, _ := classify(uint16(w))
		c := markCell{tile: uint16(w), arms: cellArms(uint16(w), 0, 10, 10, 24, 24, nil)}
		if got := row.on(c); got != (class == classMountain) {
			t.Fatalf("word %#04x: mark %v, classify class %d", w, got, class)
		}
	}
}
