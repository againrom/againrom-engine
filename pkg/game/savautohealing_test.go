package game

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func TestAutoHealing1191AmbiguousPlayerCannotChangeProfileOrWorld(t *testing.T) {
	f := purseFront1115(t)
	open, _, err := f.RestoreOriginal(purseLiteral1115(t, f, [2]uint32{1, 1}, [2]uint32{19, 23}))
	if err != nil {
		t.Fatal(err)
	}
	if err := f.App("ambiguous healing owner").OpenMission(open); err != nil {
		t.Fatal(err)
	}
	f.Options = OptionsStore{Path: filepath.Join(t.TempDir(), "options.txt")}
	profile := []byte("AutoCasting=2\nFutureOption=retained\n")
	if err := os.WriteFile(f.Options.Path, profile, 0600); err != nil {
		t.Fatal(err)
	}
	before := f.live.world.Hash()
	if _, present := f.live.world.AutoHealing(sim.SelfSlot); present {
		t.Fatal("same-slot Players acquired a shared policy")
	}
	if err := f.setGameOption(true, ui.GameOptionAutoHealing, 0); err == nil || !strings.Contains(err.Error(), "exact player") {
		t.Fatal("ambiguous option was not refused", err)
	}
	after, err := os.ReadFile(f.Options.Path)
	if err != nil || !bytes.Equal(profile, after) || f.live.world.Hash() != before || len(f.live.pending) != 0 {
		t.Fatal("refused option changed profile, world or queue", err)
	}
}

func TestAutoHealing1191DocumentDisagreementRefusesBeforeAdoption(t *testing.T) {
	f, source := formation1159Snapshot(t)
	before := f.live.world.Hash()
	source.SavedDocument, _ = cloneSavedDocument(source.SavedDocument)
	object := source.SavedDocument.GroupBindings.Players[0].ObjectIndex
	record := &source.SavedDocument.Document.Objects[object-1]
	old, err := savedStructureValue(record, "F58")
	if err != nil {
		t.Fatal(err)
	}
	newGroupSetValue1115(t, record, "F58", old^1)
	if _, err := EncodeSave(source, "mismatched player policy"); err == nil || !strings.Contains(err.Error(), "autohealing") {
		t.Fatal("mismatched source policy encoded", err)
	}
	if _, _, err := f.Restore(source); err == nil || !strings.Contains(err.Error(), "autohealing") || f.live.world.Hash() != before {
		t.Fatal("mismatched source policy restored or changed current world", err)
	}
}
