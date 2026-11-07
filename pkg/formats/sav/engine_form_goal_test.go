package sav

import (
	"encoding/binary"
	"testing"
)

// The second game's Unit command writes the engine order's walk goal at
// +0x0a beside its post at +0: the original's walk stores its goal there
// (R2-ENGINE-844) and the Move programme walks each member to its own goal
// (AI-MOVE-023), so a zero goal sends a moving member toward cell 0
// (R2-ENGINE-883). Each original-made member of a programme-4 Group holds
// its Group's destination there.
func TestTheSecondGameUnitCommandWritesTheWalkGoal(t *testing.T) {
	const post, goal, at = 0x3754, 0x3853, 0x0a
	doc := attackDocument(t, 11, 0)
	engine := documentRaw(t, doc.Objects[1], "U158")
	binary.LittleEndian.PutUint16(engine[0:], post)
	binary.LittleEndian.PutUint16(engine[at:], goal)
	out, changed, err := DocumentInProfile(doc, SecondGame, ProfileKeeps{})
	if err != nil || !changed {
		t.Fatalf("second form: %v %v", changed, err)
	}
	command := documentRaw(t, out.Objects[1], "URaw184")
	if u16(command, 0) != post || u16(command, at) != goal {
		t.Fatalf("command post %#x goal %#x, want %#x %#x", u16(command, 0), u16(command, at), post, goal)
	}
	if u16(command, 2) != 0 || command[8] != 0 || u32(command, 0x0c) != 0 {
		t.Fatalf("the command block holds +2 %#x, +8 %d, +0x0c %#x beyond its post and goal", u16(command, 2), command[8], u32(command, 0x0c))
	}
}
