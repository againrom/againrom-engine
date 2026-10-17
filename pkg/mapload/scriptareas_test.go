package mapload

import (
	"image"
	"testing"

	"againrom/pkg/formats/alm"
)

func areaNode(opcode, id uint32, slots ...[2]uint32) alm.ScriptNode {
	n := alm.ScriptNode{Opcode: opcode, ID: id}
	for i, s := range slots {
		n.Type[i], n.Value[i] = s[0], s[1]
	}
	return n
}

// areaMap is a 40x30 map with unit 7 on cell 10,12 and unit 8 placed twice.
func areaMap() *alm.Map {
	return &alm.Map{Width: 40, Height: 30, Units: []alm.Unit{
		{X: 10*256 + 128, Y: 12*256 + 3, UnitID: 7},
		{X: 1 * 256, Y: 1 * 256, UnitID: 8}, {X: 2 * 256, Y: 2 * 256, UnitID: 8},
	}}
}

func pair(left, right, cmp uint32) alm.ScriptTrigger {
	return alm.ScriptTrigger{Left: [3]uint32{left}, Right: [3]uint32{right}, Cmp: [3]uint32{cmp}}
}

func onlyArea(t *testing.T, sc alm.Script) (ScriptArea, ScriptPlaceNote) {
	t.Helper()
	areas, notes := ScriptAreas(areaMap(), sc)
	if len(areas) != 1 || len(notes) < 1 {
		t.Fatalf("areas %+v notes %+v, want one area", areas, notes)
	}
	for _, n := range notes {
		if n.Condition == areas[0].Condition && n.Node == areas[0].Node {
			return areas[0], n
		}
	}
	t.Fatalf("no note for %+v", areas[0])
	return ScriptArea{}, ScriptPlaceNote{}
}

func TestScriptAreaOfACellIsThatCell(t *testing.T) {
	sc := alm.Script{Conditions: []alm.ScriptNode{
		areaNode(14, 1, [2]uint32{ScriptX, 5}, [2]uint32{ScriptY, 6}),
		areaNode(ScriptConstant, 2, [2]uint32{ScriptInt, 1}),
	}, Triggers: []alm.ScriptTrigger{pair(1, 2, 0)}}
	a, n := onlyArea(t, sc)
	if a.Shape != AreaCell || a.Rect != image.Rect(5, 6, 6, 7) || !n.Drawn || n.Reason != "" || !a.Condition || a.Node != 0 {
		t.Fatalf("sack check: %+v %+v", a, n)
	}
	// An action's cells: instant 21 casts from one cell at another.
	sc = alm.Script{Actions: []alm.ScriptNode{areaNode(21, 4, [2]uint32{ScriptX, 1}, [2]uint32{ScriptY, 2}, [2]uint32{ScriptX, 39}, [2]uint32{ScriptY, 29}, [2]uint32{ScriptInt, 3})},
		Triggers: []alm.ScriptTrigger{{Acts: [4]uint32{4}}}}
	areas, _ := ScriptAreas(areaMap(), sc)
	if len(areas) != 2 || areas[0].Rect != image.Rect(1, 2, 2, 3) || areas[1].Rect != image.Rect(39, 29, 40, 30) || areas[0].Condition {
		t.Fatalf("cast cells: %+v", areas)
	}
}

func TestScriptAreaOfABoxIsInclusiveAndARevertedBoxIsNone(t *testing.T) {
	box := func(x0, y0, x1, y1 uint32) alm.Script {
		return alm.Script{Conditions: []alm.ScriptNode{
			areaNode(2, 1, [2]uint32{ScriptUnit, 7}, [2]uint32{ScriptX, x0}, [2]uint32{ScriptY, y0}, [2]uint32{ScriptX, x1}, [2]uint32{ScriptY, y1}),
			areaNode(ScriptConstant, 2, [2]uint32{ScriptInt, 1}),
		}, Triggers: []alm.ScriptTrigger{pair(1, 2, 0)}}
	}
	a, _ := onlyArea(t, box(3, 4, 8, 6))
	if a.Shape != AreaBox || a.Rect != image.Rect(3, 4, 9, 7) {
		t.Fatalf("box: %+v", a)
	}
	areas, notes := ScriptAreas(areaMap(), box(8, 4, 3, 6))
	if len(areas) != 0 || notes[0].Reason != ReversedBox {
		t.Fatalf("reversed box: %+v %+v", areas, notes)
	}
}

func TestScriptAreaOfADistanceIsAChebyshevSquare(t *testing.T) {
	// Check 3 carries its own radius.
	sc := alm.Script{Conditions: []alm.ScriptNode{
		areaNode(3, 1, [2]uint32{ScriptUnit, 7}, [2]uint32{ScriptX, 20}, [2]uint32{ScriptY, 10}, [2]uint32{ScriptInt, 4}),
		areaNode(ScriptConstant, 2, [2]uint32{ScriptInt, 1}),
	}, Triggers: []alm.ScriptTrigger{pair(1, 2, 0)}}
	a, _ := onlyArea(t, sc)
	if a.Shape != AreaSquare || a.Radius != 4 || a.Rect != image.Rect(16, 6, 25, 15) || a.Centre != image.Pt(20, 10) {
		t.Fatalf("check 3: %+v", a)
	}
	// Check 15 takes its radius from the constant it is compared with, on
	// either side of the pair; the square is clipped to the map.
	for _, c := range []struct {
		left, right, cmp uint32
		radius           int
	}{{1, 2, 5, 6}, {1, 2, 3, 5}, {2, 1, 4, 6}, {2, 1, 2, 5}, {1, 2, 2, 6}} {
		sc := alm.Script{Conditions: []alm.ScriptNode{
			areaNode(15, 1, [2]uint32{ScriptPlayer, 1}, [2]uint32{ScriptX, 2}, [2]uint32{ScriptY, 3}),
			areaNode(ScriptConstant, 2, [2]uint32{ScriptConst, 6}),
		}, Triggers: []alm.ScriptTrigger{pair(c.left, c.right, c.cmp)}}
		a, n := onlyArea(t, sc)
		if a.Radius != c.radius || a.Rect != image.Rect(0, 0, 2+c.radius+1, 3+c.radius+1) || n.Reason != "" {
			t.Fatalf("check 15 %+v: %+v %+v", c, a, n)
		}
	}
	// Without a constant only the point is drawn.
	sc = alm.Script{Conditions: []alm.ScriptNode{
		areaNode(7, 1, [2]uint32{ScriptUnit, 7}, [2]uint32{ScriptX, 2}, [2]uint32{ScriptY, 3}),
		areaNode(19, 2, [2]uint32{ScriptInt, 50}),
	}, Triggers: []alm.ScriptTrigger{pair(1, 2, 5)}}
	a, n := onlyArea(t, sc)
	if a.Shape != AreaCell || n.Reason != PointOnly {
		t.Fatalf("check 7 without a constant: %+v %+v", a, n)
	}
}

func TestScriptAreaAroundAUnitNeedsItsOnePlacement(t *testing.T) {
	check6 := func(unit uint32) alm.Script {
		return alm.Script{Conditions: []alm.ScriptNode{
			areaNode(6, 1, [2]uint32{ScriptUnit, unit}, [2]uint32{ScriptUnit, 8}),
			areaNode(ScriptConstant, 2, [2]uint32{ScriptInt, 3}),
		}, Triggers: []alm.ScriptTrigger{pair(1, 2, 5)}}
	}
	a, _ := onlyArea(t, check6(7))
	if a.Shape != AreaSquare || a.Centre != image.Pt(10, 12) || a.Radius != 3 {
		t.Fatalf("around unit 7: %+v", a)
	}
	for _, u := range []uint32{8, 10001, 99} {
		areas, notes := ScriptAreas(areaMap(), check6(u))
		if len(areas) != 0 || notes[0].Reason != UnitNotPlaced {
			t.Fatalf("unit %d: %+v %+v", u, areas, notes)
		}
	}
}

func TestScriptAreaNotesWhatIsNotDrawn(t *testing.T) {
	sc := alm.Script{
		Conditions: []alm.ScriptNode{areaNode(5, 1, [2]uint32{ScriptUnit, 7}), areaNode(ScriptConstant, 2, [2]uint32{ScriptInt, 1})},
		Actions:    []alm.ScriptNode{areaNode(4, 1), areaNode(27, 2, [2]uint32{ScriptUnit, 7}, [2]uint32{ScriptX, 50}, [2]uint32{ScriptY, 3})},
		Triggers:   []alm.ScriptTrigger{{Left: [3]uint32{1}, Right: [3]uint32{2}, Acts: [4]uint32{1, 2, 9}}},
	}
	areas, notes := ScriptAreas(areaMap(), sc)
	reasons := map[string]int{}
	for _, n := range notes {
		reasons[n.Reason]++
	}
	if len(areas) != 0 || len(notes) != 5 || reasons[NoPlace] != 3 || reasons[OutsideMap] != 1 || reasons[Unresolved] != 1 {
		t.Fatalf("areas %+v notes %+v", areas, notes)
	}
}

// Every signature a claim states names one type code per slot, a cell as an
// X slot then a Y slot, and every named operation the authoring list offers
// has one.
func TestScriptSignaturesAreWellFormed(t *testing.T) {
	for _, condition := range []bool{true, false} {
		for _, op := range ScriptOperations(condition) {
			sig, ok := ScriptSignature(condition, op.Opcode, 4)
			if ScriptOperationNamed(condition, op.Opcode) && !ok {
				t.Errorf("condition %v opcode %d: named %v, signature %v", condition, op.Opcode, ScriptOperationNamed(condition, op.Opcode), ok)
			}
			for s, slot := range sig {
				if slot.Type > ScriptStructure || (slot.Type == ScriptX) != (s+1 < len(sig) && sig[s+1].Type == ScriptY && sig[s+1].Role == "") {
					t.Errorf("condition %v opcode %d slot %d: %+v", condition, op.Opcode, s, sig)
				}
				if slot.Type != ScriptNone && slot.Type != ScriptY && slot.Role == "" {
					t.Errorf("condition %v opcode %d slot %d has no role", condition, op.Opcode, s)
				}
			}
		}
	}
	for _, c := range ScriptGroupCommands {
		sig, ok := ScriptSignature(false, ScriptGroupCommand, c)
		groupSlot := 9
		if c == 15 {
			groupSlot = 1 // Follow stores the group after the command, as the shipped nodes do
		}
		if !ok || sig[0].Type != ScriptConst || sig[groupSlot].Type != ScriptGroup || c == 15 && sig[9].Type != ScriptNone {
			t.Errorf("group command %d: %+v", c, sig)
		}
	}
	if _, ok := ScriptSignature(false, ScriptGroupCommand, 6); ok {
		t.Error("group command 6 has no case and no signature")
	}
}
