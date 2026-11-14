package game

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"againrom/internal/witness"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func TestShortcutInventoryFrames(t *testing.T) {
	dir := os.Getenv("AGAINROM_SHORTCUT_FRAMES")
	if dir == "" || os.Getenv("AGAINROM_ASSETS") == "" {
		t.Skip("needs shortcut frame directory and install")
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	party := MissionPartyAs(false, f.StartWeapon.Value(), f.Bodies, f.Table)
	item := mapload.ItemInstanceFromCode(0xe07, f.Table)
	party[0].CarriedItems = append(party[0].CarriedItems, item)
	party[0].Carried = append(party[0].Carried, item.Code)
	f.Carried = party
	a := f.App("shortcut inventory")
	a.Layout(1024, 768)
	f.ConfigureSaveSeams(a, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	if err := a.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	openMissionPack(t, a)
	id := f.live.mission.ids[0]
	e, _ := f.live.entity(id)
	inspectionCentre(f.live, int(e.X), int(e.Y))
	if err := a.HeadlessSelectEntity(uint32(id)); err != nil {
		t.Fatal(err)
	}
	x, y, err := a.HeadlessPackCellPoint(0)
	if err != nil {
		t.Fatal(err)
	}
	before, _, err := a.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartInventoryPack})
	if err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessPointer("hover", x, y); err != nil {
		t.Fatal(err)
	}
	if err := a.HeadlessKey("shift-f4"); err != nil {
		t.Fatal(err)
	}
	pic, _, err := a.HeadlessPicture(ui.HeadlessRequest{Part: ui.HeadlessPartInventoryPack})
	if err != nil {
		t.Fatal(err)
	}
	if f.quickSpells[0].Item == "" || bytes.Equal(before.Pix, pic.Pix) {
		t.Fatal("item or visible mark missing")
	}
	name := filepath.Base(os.Getenv("AGAINROM_ASSETS")) + "-inventory.png"
	if err := os.WriteFile(filepath.Join(dir, name), witness.Bytes(t, pic), 0o644); err != nil {
		t.Fatal(err)
	}
}
