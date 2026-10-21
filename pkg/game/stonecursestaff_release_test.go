package game

import (
	"os"
	"testing"

	"againrom/pkg/sim"
)

// The mage's pack holds a staff of Stone Curse with a plain saved row.
const (
	stoneStaffCode  = 37165
	stoneStaffSpell = 20
	stoneStaffPower = 45
)

func stoneStaffWorn(w *sim.World, id sim.EntityID) bool {
	worn, _ := w.Equipped(id)
	e, _ := w.Entity(id)
	return worn[0] == stoneStaffCode && e.WeaponSpell == stoneStaffSpell && e.WeaponSpellLevel == stoneStaffPower
}

func TestReleaseStoneCurseStaffEquipsFromThePack(t *testing.T) {
	path := os.Getenv("AGAINROM_WHATISATTACKED_SAV")
	if path == "" {
		t.Skip("AGAINROM_WHATISATTACKED_SAV is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := shopOrderFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "whatisattacked.sav")
	t.Cleanup(app.StopAudio)
	w, _ := f.LiveWorld()
	mage := offMapMageA
	pack, _ := w.CarriedItems(mage)
	index := -1
	for k, it := range pack {
		if it.Code == stoneStaffCode {
			index = k
		}
	}
	if index < 0 || stoneStaffWorn(w, mage) {
		t.Fatal("the save does not hold the staff in the mage's pack")
	}
	e, _ := w.Entity(mage)
	f.live.view.Camera().CenterOn(float64(e.X)*32+16, float64(e.Y)*32+16)
	ogreStep(t, f, app)
	if err := app.HeadlessSelectEntity(uint32(mage)); err != nil {
		t.Fatal(err)
	}
	if !f.live.view.SaveApplication().InventoryOpen {
		if err := app.HeadlessKey("i"); err != nil {
			t.Fatal(err)
		}
	}
	app.HeadlessStep()
	px, py, err := app.HeadlessPackCellPoint(index)
	if err != nil {
		t.Fatal(err)
	}
	dx, dy, err := app.HeadlessDollSlotPoint(1)
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range []struct {
		a    string
		x, y int
	}{{"hover", px, py}, {"press", px, py}, {"move", (px + dx) / 2, (py + dy) / 2}, {"move", dx, dy}, {"release", dx, dy}} {
		if err := app.HeadlessPointer(s.a, s.x, s.y); err != nil {
			t.Fatal(s.a, err)
		}
	}
	ogreStep(t, f, app)
	if !stoneStaffWorn(w, mage) {
		t.Fatal("the drag to the weapon slot did not wear the staff of Stone Curse")
	}
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	saved, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	back := shopOrderFront(t)
	opener, town, err := back.RestoreOriginal(saved)
	if err != nil || town || opener == nil {
		t.Fatal("restore", err, town)
	}
	restored := back.App("stone-staff-restore")
	restored.Layout(1024, 768)
	t.Cleanup(restored.StopAudio)
	if err := restored.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	if cold, _ := back.LiveWorld(); !stoneStaffWorn(cold, mage) {
		t.Fatal("SAVE and LOAD lost the worn staff of Stone Curse")
	}
}

// The original's resave of that save wears the staff.
func TestReleaseOriginalResaveWearsTheStoneCurseStaff(t *testing.T) {
	path := os.Getenv("AGAINROM_STONE_STAFF_ORIGINAL_RESAVE")
	if path == "" {
		t.Skip("AGAINROM_STONE_STAFF_ORIGINAL_RESAVE is not set")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	f := shopOrderFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "originalresavedwithstonecurse.sav")
	t.Cleanup(app.StopAudio)
	w, _ := f.LiveWorld()
	wearers := 0
	for _, e := range w.Entities() {
		if stoneStaffWorn(w, e.ID) {
			wearers++
		}
	}
	if wearers != 1 {
		t.Fatal("actors wearing the staff of Stone Curse:", wearers)
	}
}
