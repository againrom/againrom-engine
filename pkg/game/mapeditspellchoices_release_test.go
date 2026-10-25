package game

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"againrom/internal/firstgame"
	"againrom/pkg/ui"
)

func installedSpellSlot(opcode uint32, types [10]uint32) int {
	argument := -1
	switch opcode {
	case 21:
		argument = 4
	case 24:
		argument = 2
	case 25, 30:
		argument = 0
	case 29:
		argument = 2
	}
	if argument < 0 {
		return -1
	}
	var scalars []int
	for slot, tag := range types {
		if !slices.Contains([]uint32{2, 3, 4, 8, 9}, tag) {
			scalars = append(scalars, slot)
		}
	}
	if argument < len(scalars) {
		return scalars[argument]
	}
	return -1
}

func TestReleaseScriptSpellCurrentPicksPreserveEveryMap(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	names := LoadTextTable(x.archives.Containers, SpellNamesTextPath, InstallTextCode(x.archives.Containers, x.archives.Game().Edition()))
	counts := map[uint32]int{}
	multisets := map[uint32][]uint32{}
	dir := t.TempDir()
	keys := installedMapKeys(x)
	for i, key := range keys {
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(key, err)
		}
		e.Layout(1280, 800)
		closeMapPanel(t, e)
		before := bytes.Clone(e.Document().SourceBytes)
		if sc := e.Document().Script; sc != nil {
			for _, n := range sc.Actions {
				slot := installedSpellSlot(n.Opcode, n.Types)
				if slot < 0 {
					continue
				}
				v := n.Values[slot]
				name, ok := names.At(int(v) - 1)
				if !ok {
					t.Fatal(key, n.Opcode, v, "no installed spell name")
				}
				kind := scriptParameterKind(false, n.Opcode, n.Types, slot)
				want := fmt.Sprintf("%d: %s", v, name)
				found := false
				for _, row := range sc.Targets[kind] {
					found = found || row.Value == v && row.ChoiceText() == want
				}
				if !found {
					t.Fatal(key, n.Opcode, v, "no named current row", want)
				}
				var prompt []ui.InspectionChoice
				for _, f := range e.Document().Edits.Fields("Action", e.Document().Records[n.Record].Index) {
					if f.Key == fmt.Sprintf("value%d", slot) {
						prompt = f.Choices
					}
				}
				if !slices.Contains(prompt, ui.InspectionChoice{Value: int64(v), Label: want}) {
					t.Fatal("E missing row", key, want)
				}
				e.Select(n.Record, false)
				chooseParameterValue(t, e, parameterField(n, slot), v, sc.Targets[kind])
				r, ok := e.PopupButtonRect(0)
				trigClick(t, e, r, ok, "Apply current spell")
				if e.Dirty() || !bytes.Equal(before, e.Document().SourceBytes) {
					t.Fatal(key, n.Opcode, "current pick wrote bytes")
				}
				counts[n.Opcode]++
				multisets[n.Opcode] = append(multisets[n.Opcode], v)
			}
		}
		path := filepath.Join(dir, fmt.Sprintf("map-%d.alm", i))
		if _, err := e.Document().Edits.SaveAs(path); err != nil {
			t.Fatal(key, err)
		}
		after, err := os.ReadFile(path)
		if err != nil || !bytes.Equal(before, after) {
			t.Fatal(key, "Save As bytes", err)
		}
	}
	want := map[uint32][]uint32{
		21: {3, 3, 3, 8, 8, 8, 17, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 21, 21, 21, 21, 21},
		24: {5, 5, 5, 5, 10, 10, 10, 10, 16, 16, 16, 16, 20, 20, 20, 22, 22, 22, 22, 23},
		25: {3, 9}, 29: {3, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19, 19}, 30: {20, 20},
	}
	for _, opcode := range []uint32{21, 24, 25, 29, 30} {
		slices.Sort(multisets[opcode])
		if !slices.Equal(multisets[opcode], want[opcode]) {
			t.Errorf("action %d values %v want %v", opcode, multisets[opcode], want[opcode])
		}
		t.Logf("root=%s action=%d named=%d current-picks byte-equal", os.Getenv("AGAINROM_ASSETS"), opcode, counts[opcode])
	}
	t.Logf("root=%s all maps Save As byte-equal=%d", os.Getenv("AGAINROM_ASSETS"), len(keys))
}

func TestReleaseScriptSpellPickerFrames(t *testing.T) {
	firstgame.SkipOnSecondGame(t)
	x := installedInspector(t)
	dir := os.Getenv("AGAINROM_SCRIPT_SPELL_FRAMES")
	for _, key := range installedMapKeys(x) {
		if !strings.HasPrefix(key, "scenario/") {
			continue
		}
		e := x.Editor()
		if err := e.Open(key); err != nil {
			t.Fatal(err)
		}
		e.Layout(1280, 800)
		closeMapPanel(t, e)
		if e.Document().Script == nil {
			continue
		}
		for _, n := range e.Document().Script.Actions {
			if n.Opcode != 21 {
				continue
			}
			e.Select(n.Record, false)
			r, ok := e.FieldRect(parameterField(n, 4))
			trigClick(t, e, r, ok, "spell dropdown")
			cover := image.Rectangle{}
			for row := range e.DropdownChoices() {
				if rr, ok := e.DropdownRowRect(row); ok {
					cover = cover.Union(rr)
				}
			}
			path := filepath.Join(dir, "cast-spell-"+installLanguage(x)+".png")
			writeEditorFrame(t, e, path, dir != "", cover)
			if dir != "" {
				full, err := os.Open(path)
				if err != nil {
					t.Fatal(err)
				}
				pic, err := png.Decode(full)
				closeErr := full.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
				crop := cover.Union(r).Inset(-12).Intersect(pic.Bounds())
				cropped := image.NewRGBA(image.Rectangle{Max: crop.Size()})
				draw.Draw(cropped, cropped.Bounds(), pic, crop.Min, draw.Src)
				file, err := os.Create(filepath.Join(dir, "cast-spell-crop-"+installLanguage(x)+".png"))
				if err != nil {
					t.Fatal(err)
				}
				err = png.Encode(file, cropped)
				closeErr = file.Close()
				if err != nil || closeErr != nil {
					t.Fatal(err, closeErr)
				}
			}
			t.Logf("frame root=%s map=%s record=%d value=%d", os.Getenv("AGAINROM_ASSETS"), key, n.Record, n.Values[4])
			return
		}
	}
	t.Fatal("no installed campaign Cast node")
}
