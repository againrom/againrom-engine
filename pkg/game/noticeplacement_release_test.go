package game

import (
	"fmt"
	"path/filepath"
	"testing"
)

func TestReleasePlacedHeroNoticeSurvivesSAVContinuation(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "notice-placement.sav")
	hero, _ := pickupPublicationHero(t, f)
	if err := f.live.world.HeadlessPlace(hero, 42, 39); err != nil {
		t.Fatal(err)
	}

	var noticeText string
	for step := 0; step <= 64; step++ {
		text, kind, open := f.LiveNotice()
		if open {
			noticeText = text
			t.Logf("script notice at step %d: kind=%v text=%q", step, kind, text)
			break
		}
		if step == 64 {
			t.Fatal("placed hero did not open the shipped notice")
		}
		if err := app.HeadlessStep(); err != nil {
			t.Fatal(err)
		}
	}
	if noticeText == "" {
		t.Fatal("shipped notice opened without text")
	}
	if err := app.HeadlessActivate("notice"); err != nil {
		t.Fatal(err)
	}
	store, name, _ := menuSAVE(t, f, app, OriginalStore{})
	cold := loadAreaContinuation(t, filepath.Join(store.Dir, name))
	compare := func(cut string) {
		if f.live.world.Hash() != cold.live.world.Hash() {
			currentMenuWorldDiagnostics(t, f.live.world, cold.live.world)
			t.Fatalf("%s: World hash changed: %016x / %016x", cut, f.live.world.Hash(), cold.live.world.Hash())
		}
		assertCurrentWorldEqual(t, f.live.world, cold.live.world, cut)
	}
	compare("notice SAVE/cold LOAD")
	for tick := 1; tick <= 16; tick++ {
		f.live.tick()
		cold.live.tick()
		compare(fmt.Sprintf("notice continuation tick %d", tick))
	}
}
