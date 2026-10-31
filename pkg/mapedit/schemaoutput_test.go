package mapedit_test

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"testing"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
)

// editorOutputDigest is the SHA-256 of every byte image, field list and error
// the scripted session in TestEditorOutputKeepsItsBytes produces, taken on the
// tree whose editor spelled its own record offsets.
const editorOutputDigest = "ef1e25cd6e356bd8e8a77f8b8c36b61418dc5e69311ddd5c46cd6ed5a18ccdf2"

func TestEditorOutputKeepsItsBytes(t *testing.T) {
	e, err := mapedit.NewMap(alm.DefaultNewMap())
	if err != nil {
		t.Fatal(err)
	}
	h := sha256.New()
	step := func(name string, err error) {
		fmt.Fprintf(h, "%s: %v\n", name, err)
		h.Write(e.Bytes())
	}
	scriptEditorSession(e, h, step)
	saved, err := e.SaveBytes()
	fmt.Fprintf(h, "save: %v\n", err)
	h.Write(saved)
	if got := hex.EncodeToString(h.Sum(nil)); got != editorOutputDigest {
		t.Fatalf("editor output digest %s, want %s", got, editorOutputDigest)
	}
}

func scriptEditorSession(e *mapedit.Editor, h hash.Hash, step func(string, error)) {
	paint := func(name string, b mapedit.Brush, at ...mapedit.Cell) {
		_, err := e.Paint(b, at, testClass)
		step(name, err)
	}
	paint("mountain", mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassMountain), Radius: 2}, mapedit.Cell{X: 10, Y: 10})
	paint("water", mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassWater), Radius: 3, Shape: mapedit.BrushSquare}, mapedit.Cell{X: 30, Y: 20})
	paint("road", mapedit.Brush{Plane: mapedit.PlaneTile, Value: int(mapedit.ClassRoad), Radius: 1}, mapedit.Cell{X: 12, Y: 11}, mapedit.Cell{X: 13, Y: 11})
	paint("block", mapedit.Brush{Plane: mapedit.PlaneImpassable, Value: 1, Radius: 2}, mapedit.Cell{X: 20, Y: 20})
	paint("unblock", mapedit.Brush{Plane: mapedit.PlaneImpassable, Value: 0, Radius: 1}, mapedit.Cell{X: 20, Y: 20})
	paint("height", mapedit.Brush{Plane: mapedit.PlaneHeight, Value: 40, Radius: 3}, mapedit.Cell{X: 40, Y: 40})
	paint("object", mapedit.Brush{Plane: mapedit.PlaneObject, Value: 3, Radius: 1}, mapedit.Cell{X: 44, Y: 44})
	step("raw tile", e.SetCell(mapedit.GridRaw, mapedit.GridTile, 5, 6, 0x2011))
	step("normalized tile", e.SetCell(mapedit.GridNormalized, mapedit.GridTile, 7, 6, 0xffff))
	step("altitude", e.SetCell(mapedit.GridRaw, mapedit.GridAltitude, 8, 6, 77))

	step("angle", e.SetAngle(0.5))
	step("word0c", e.SetWord0C(3))
	step("word10", e.SetWord10(4))
	step("word14", e.SetWord14(5))
	step("word70", e.SetWord70(6))
	step("word74", e.SetWord74(7))
	step("bitmask", e.SetBitmask(0x3f))
	step("name", e.SetName("Schema map"))
	step("description", e.SetDescription("A map the schema writes"))

	for i := 0; i < 2; i++ {
		rec, err := e.NewPlayerRecord()
		step("new player", err)
		if err == nil {
			_, err = e.Place(mapedit.KindPlayer, rec)
			step("place player", err)
		}
	}
	for i, class := range []int16{40, 1, 7} {
		rec, err := e.NewUnitRecord(uint32(10+i)*256+128, uint32(11+i)*256+128, class, 2, uint32(1+i%2))
		step("new unit", err)
		if err == nil {
			_, err = e.PlaceUnit(rec)
			step("place unit", err)
		}
	}
	for i, kind := range []uint32{0x21, 0x08} {
		rec, err := e.NewStructureRecord(uint32(20+i)*256, uint32(21+i)*256, kind, 1)
		step("new structure", err)
		if err == nil {
			_, err = e.Place(mapedit.KindStructure, rec)
			step("place structure", err)
		}
	}
	_, err := e.Place(mapedit.KindLoot, e.GroundSack(14*256+128, 15*256+128, 0x0202))
	step("sack", err)
	_, err = e.AddUnitStock(0)
	step("stock", err)
	_, err = e.UnitStock(1)
	step("unit stock", err)
	step("arm unit", e.AddArm(mapedit.ArmWinUnit, 1))
	step("arm structure", e.AddArm(mapedit.ArmWinStructure, 1))
	step("arm lose", e.AddArm(mapedit.ArmLoseUnit, 2))
	_, err = e.AddDropZone(16, 17)
	step("drop zone", err)
	step("drop cell", e.SetDropCell(18, 19))
	id, err := e.NextScriptID(mapedit.KindAction)
	step(fmt.Sprintf("next id %d", id), err)
	node, err := mapedit.NewScriptNode("schema node", 3, id, mapedit.ScriptParam{Type: 4, Value: 1}, mapedit.ScriptParam{Type: 5, Value: 9})
	step("new node", err)
	if err == nil {
		_, err = e.Place(mapedit.KindAction, node)
		step("place node", err)
	}
	step("set node", e.SetScriptNode(mapedit.KindAction, 0, "renamed", 4, mapedit.ScriptParam{Type: 1, Value: 2}))

	kinds := []mapedit.Kind{mapedit.KindUnit, mapedit.KindStructure, mapedit.KindLoot, mapedit.KindAction,
		mapedit.KindCondition, mapedit.KindTrigger, mapedit.KindPlayer, mapedit.KindEnchantment}
	for _, k := range kinds {
		for i := 0; i < e.Count(k); i++ {
			fields, err := e.Fields(k, i)
			step(fmt.Sprintf("fields %d/%d", k, i), err)
			for _, f := range fields {
				fmt.Fprintf(h, "%s|%s|%d|%d|%t|%t|%d|%d|%d|%d|%d|%d|%d|%t|%d|%d|%d\n", f.Key, f.Label, f.Offset, f.Width,
					f.Signed, f.Text != nil, f.Min, f.Max, f.Check, f.Element, f.Counter, f.Bit, f.Mask, f.Insert,
					f.CopyOffset, f.Toward, i)
				if f.Text != nil {
					step("text "+f.Key, e.SetTextField(k, i, f.Key, "ab"))
					continue
				}
				v, err := e.FieldValue(k, i, f.Key)
				step(fmt.Sprintf("value %s %d", f.Key, v), err)
				next := f.Max
				if f.Max-f.Min > 2 {
					next = f.Min + 1
				}
				step("set "+f.Key, e.SetField(k, i, f.Key, next))
			}
		}
	}

	step("move", e.Move(mapedit.KindUnit, 0, 30*256, 31*256))
	_, err = e.CloneRecord(mapedit.KindUnit, 1, 32*256, 33*256)
	step("clone", err)
	_, err = e.CloneRecord(mapedit.KindStructure, 0, 34*256, 35*256)
	step("clone structure", err)
	step("delete", e.Delete(mapedit.KindUnit, 2))
	step("delete structure", e.Delete(mapedit.KindStructure, 1))
	step("reorder", e.Reorder(mapedit.KindTrigger, 0, 1))
	step(fmt.Sprintf("undo %t", e.Undo()), nil)
	step(fmt.Sprintf("redo %t", e.Redo()), nil)
}
