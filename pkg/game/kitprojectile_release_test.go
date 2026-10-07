package game

import (
	"reflect"
	"strings"
	"testing"

	"againrom/pkg/formats/sav"
)

// The corpus's one nonempty Projectiles store (SAV-PROJCORP-430).
const kitProjectileSource = "2026-08-15/game0018.sav"
const kitProjectileSourceHash = "1e2eb21f47082ab05a8f7810148fdf722fe8aa4f69a57d79953bd8fe0025eb6b"

func kitProjectileLabel(edition string) string {
	if edition == "RU" {
		return "9610 Стрела RU"
	}
	return "9609 стрела EN"
}

func kitProjectileFile(edition string) string {
	if edition == "RU" {
		return "game9610.sav"
	}
	return "game9609.sav"
}

func kitProjectileStore(t *testing.T, raw []byte) sav.ProjectileStore {
	t.Helper()
	file, err := sav.Open(raw)
	if err != nil {
		t.Fatal(err)
	}
	store, present, err := file.Projectiles()
	if err != nil || !present {
		t.Fatal("no Projectiles subtree", present, err)
	}
	return store
}

// TestKitProjectileBuild writes the kit file into AGAINROM_OWNER_KIT_OUT, or a
// temporary directory.
func TestKitProjectileBuild(t *testing.T) {
	_, source := groundCorpusFile(t, kitProjectileSource, kitProjectileSourceHash)
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, source, "source.sav")
	t.Cleanup(app.StopAudio)
	if f.live == nil || f.live.world == nil || f.liveMission != 40 {
		t.Fatalf("source LOAD opened mission %d", f.liveMission)
	}
	want := kitProjectileStore(t, source)
	if got := f.live.world.SavedProjectiles(); len(got.Items) != 1 || len(want.Items) != 1 || got.Items[0].ID != want.Items[0].ID {
		t.Fatal("the loaded world does not hold the source projectile", got)
	}
	edition := kitOwnerEdition(f)
	for _, line := range kitOwnerObserve(f) {
		t.Log("LIVE", line)
	}
	label := kitProjectileLabel(edition)
	store := SaveStore{Dir: t.TempDir()}
	f.ConfigureSaveSeams(app, store, OriginalStore{}, nil)
	raw := cityRosterF2Save(t, app, store, label)
	if got := kitProjectileStore(t, raw); !reflect.DeepEqual(got, want) {
		t.Fatalf("SAVE changed the Projectiles store: %+v, source %+v", got, want)
	}
	kitOwnerWrite(t, kitOwnerOutput(t, edition), kitProjectileFile(edition), raw)
	loaded := strings.Join(kitOwnerReceive(t, raw), "; ")
	for _, need := range []string{"projectile records 1, projectile drivers 1", "after: projectile records 0, projectile drivers 0"} {
		if !strings.Contains(loaded, need) {
			t.Fatalf("cold LOAD of the projectile file lacks %q: %s", need, loaded)
		}
	}
	t.Logf("BUILD %s label %q bytes %d", edition, label, len(raw))
}
