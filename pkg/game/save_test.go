package game

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"againrom/internal/synth"
	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
	"againrom/pkg/vfs"
)

// saveCampaign is a campaign small enough to state and rich enough for the town
// half to have something to say: one chapter, one shop offer, a payment.
func saveCampaign() Campaign {
	return Campaign{
		Main:    []int{10, 20},
		Side:    []int{11, 12},
		Offered: []int{10, 20},
		Chapters: map[int]Chapter{
			10: {Mission: 10, Payment: 250, Shop: []int{11, 12}},
			20: {Mission: 20},
		},
	}
}

// saveTown is a town with something in every one of its five fields.
func saveTown(t *testing.T) *Town {
	t.Helper()
	town := NewTown(saveCampaign())
	town.Arrive()
	if m, ok := town.Take(TownShop, 0); !ok || m != 11 {
		t.Fatalf("setup: Take(shop,0) = (%d,%v), want (11,true)", m, ok)
	}
	town.Won(10) // pays 250 and leaves the chapter behind
	return town
}

// saveParty is one party member with a value in every kind of field
// PartyMember carries, so the payload's reach over the nested structs is
// witnessed rather than assumed (plan D-1).
func saveParty() []mapload.PartyMember {
	m := mapload.PartyMember{
		Class:              7,
		SuppressCorpseLoot: true,
		Body:               "MAN",
		BodyDir:            "men",
		Mage:               true,
		Profile:            data.Profile{Fighter: true, HealthColumn: true},
		FigureDir:          "fig",
		FigureFace:         3,
		Hero:               data.Hero{Body: 11, Reaction: 12, Mind: 13, Spirit: 14},
		KnownSpells:        0x5a,
		Carried:            []uint16{0x1234, 0x5678},
		Weapon:             &data.Weapon{Name: "sword", Row: 4, DamageBase: 9},
		Carry:              &mapload.Carry{Items: []uint16{7}},
	}
	m.Hero.Skill[1] = 5
	m.Worn[0] = 0x0abc
	m.Carry.SkillXP[1] = 42
	m.Carry.Equipped[0] = 0x0abc
	return []mapload.PartyMember{m}
}

// TestATownSaveRoundTripsThroughDisk is AC-2's state half: the campaign half
// alone, out through the envelope and back into a FRESH town over the same
// campaign.
func TestATownSaveRoundTripsThroughDisk(t *testing.T) {
	source := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: saveTown(t), Carried: saveParty(), Offered: 20}}
	capture, _, err := source.Snapshot(false)
	if err != nil || !reflect.DeepEqual(capture.Party, mapload.CloneParty(source.Carried)) {
		t.Fatal("capture lost complete party fields", err)
	}
	f := currentTrainingCity(t)
	f.Campaign = resolved(saveCampaign(), nil)
	f.Town, f.Offered = saveTown(t), 20

	snap, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	if !strings.Contains(label, "town") {
		t.Errorf("label = %q, want it to name the town", label)
	}
	if snap.Mission != 0 || len(snap.World) != 0 {
		t.Errorf("a town save carries mission %d and %d world bytes, want 0 and 0",
			snap.Mission, len(snap.World))
	}

	b, err := f.ExportCurrentSave(snap, label)
	if err != nil {
		t.Fatalf("EncodeSave: %v", err)
	}
	gotLabel, err := OriginalSaveLabel(b, f.textSelector())
	if err != nil || !strings.HasPrefix(gotLabel, label) {
		t.Fatalf("SAV label=%q want prefix %q: %v", gotLabel, label, err)
	}

	// A FRESH front end, as a restarted program is.
	g := &FrontEnd{InstallResources: f.InstallResources}
	open, town, err := g.RestoreOriginal(b)
	if err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if !town || open != nil {
		t.Fatalf("Restore of a town save = (open!=nil %v, town %v), want (false, true)", open != nil, town)
	}
	if g.Town.Gold() != initialPlayerPurse+250 {
		t.Errorf("gold came back %d, want %d", g.Town.Gold(), initialPlayerPurse+250)
	}
	if !g.Town.Done(10) {
		t.Error("the won mission did not come back")
	}
	if !g.Town.Open() {
		t.Error("the town's own latch did not come back")
	}
	if got := g.Town.Available(); !reflect.DeepEqual(got, []int{11}) {
		t.Errorf("available = %v, want [11]", got)
	}
	// The taken mark came back: element 0 of the shop is gone from the live
	// chapter's offers. The live chapter is 20 now, so ask 10's directly.
	if g.Town.taken[offerRef{chapter: 10, building: TownShop, index: 0}] != true {
		t.Error("the consumed shop offer did not come back")
	}
	if g.Offered != 20 {
		t.Errorf("Offered = %d, want 20", g.Offered)
	}
	// Restore is a production ownership boundary. A legacy member with no ID
	// gains one there; every other character field must remain exact.
	wantParty := mapload.OwnParty(f.Carried)
	if !reflect.DeepEqual(g.Carried, wantParty) {
		t.Errorf("the party did not survive the round trip:\n got %+v\nwant %+v", g.Carried, wantParty)
	}
}

// TestSaveTakenOffersEncodeDeterministicallyAcrossRepeatedSnapshots is 1017
// round 3's witness for W-2: snapshotTown reads t.taken off a Go map, whose
// iteration order is randomized per call, and sorts the result before writing
// it (sortOffers(s.Taken)). Two taken offers pass by accident about half the
// time; three make an unsorted encoding disagree with itself on almost every
// run, which is why this test uses three.
func TestSaveTakenOffersEncodeDeterministicallyAcrossRepeatedSnapshots(t *testing.T) {
	camp := Campaign{
		Main:     []int{10},
		Offered:  []int{10},
		Chapters: map[int]Chapter{10: {Mission: 10, Shop: []int{11, 12, 13}}},
	}
	newTown := func() *Town {
		town := NewTown(camp)
		town.Arrive()
		for i := 0; i < 3; i++ {
			if _, ok := town.Take(TownShop, i); !ok {
				t.Fatalf("setup: Take(shop,%d) failed", i)
			}
		}
		return town
	}

	var first []byte
	for i := 0; i < 20; i++ {
		f := currentTrainingCity(t)
		f.Campaign, f.Town = resolved(camp, nil), newTown()
		snap, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatalf("run %d: Snapshot: %v", i, err)
		}
		if len(snap.Taken) != 3 {
			t.Fatalf("run %d: Taken has %d offer(s), want 3", i, len(snap.Taken))
		}
		b, err := f.ExportCurrentSave(snap, label)
		if err != nil {
			t.Fatalf("run %d: EncodeSave: %v", i, err)
		}
		if i == 0 {
			first = b
			continue
		}
		if !bytes.Equal(b, first) {
			t.Fatalf("run %d: encoded payload differs from run 0's; three taken offers "+
				"are not written in a stable order", i)
		}
	}
}

func TestSaveKeepsStableIdentityAndTemporaryMembershipMetadata(t *testing.T) {
	f := currentTrainingCity(t)
	party := mapload.CloneParty(f.Carried)
	party[0].ID, party[0].StartingHero = "hero", true
	party[1].ID, party[1].StartingHero, party[1].Temporary = "temporary:witness", false, true
	for _, member := range party {
		captured := captureCurrentParty(0, member)
		if !reflect.DeepEqual(captured.restore(), member) {
			t.Fatal("current party DTO lost a complete member")
		}
	}
	f.Carried = party
	s, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(s, label)
	if err != nil {
		t.Fatal(err)
	}
	cold := &FrontEnd{InstallResources: f.InstallResources}
	if _, town, err := cold.RestoreOriginal(raw); err != nil || !town {
		t.Fatal("identity SAV LOAD", town, err)
	}
	back, _, err := cold.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}

	if len(back.Party) != len(party) {
		t.Fatal("SAV changed membership count", len(back.Party), len(party))
	}
	for i, want := range party {
		got := back.Party[i]
		if got.ID != want.ID || got.Temporary != want.Temporary || got.StartingHero != want.StartingHero || got.PlayerCharacter != want.PlayerCharacter || got.CompanionNPC != want.CompanionNPC || got.MercenaryType != want.MercenaryType {
			t.Fatalf("SAV identity/membership changed member %d: %q/%v/%v/%v/%d/%d -> %q/%v/%v/%v/%d/%d", i, want.ID, want.Temporary, want.StartingHero, want.PlayerCharacter, want.CompanionNPC, want.MercenaryType, got.ID, got.Temporary, got.StartingHero, got.PlayerCharacter, got.CompanionNPC, got.MercenaryType)
		}
	}

}

func TestSaveDepartedCharactersRoundTrips(t *testing.T) {
	want := Snapshot{Residue: SnapshotResidue{
		DepartedCharacters: []uint32{109, 107},
		Commanded:          []uint32{106, 108},
	}}
	to, _ := readoutWorld(t,
		sim.Entity{ID: 107, HP: 10, MaxHP: 10, Owner: sim.SelfSlot},
		sim.Entity{ID: 109, HP: 10, MaxHP: 10, Owner: sim.SelfSlot})
	to.mission = &missionNotices{guarded: []sim.EntityID{107, 109}, ids: []sim.EntityID{107, 109}}
	to.applyResidue(want.Residue)
	got := to.residue()
	sort.Slice(got.DepartedCharacters, func(i, j int) bool { return got.DepartedCharacters[i] < got.DepartedCharacters[j] })
	sort.Slice(got.Commanded, func(i, j int) bool { return got.Commanded[i] < got.Commanded[j] })
	if !reflect.DeepEqual(got.DepartedCharacters, []uint32{107, 109}) || !reflect.DeepEqual(got.Commanded, []uint32{106, 108}) {
		t.Fatalf("departed/commanded residue = %v/%v", got.DepartedCharacters, got.Commanded)
	}
}

func TestTheCampaignIsNotInTheSave(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: saveTown(t)}}
	snap, _, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot: %v", err)
	}
	other := Campaign{Main: []int{30}, Offered: []int{30},
		Chapters: map[int]Chapter{30: {Mission: 30, Shop: []int{31}}}}
	g := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(other, nil)}}
	g.Town = testRestoreTown(other, snap)
	offers := g.Town.Offers(TownShop)
	if len(offers) != 1 || offers[0].Mission != 31 {
		t.Fatalf("the restored town offers %+v, want the NEW install's own shop row", offers)
	}
	if g.Town.Gold() != initialPlayerPurse+250 {
		t.Errorf("gold = %d, want the save's own %d", g.Town.Gold(), initialPlayerPurse+250)
	}
}

func TestTheStoreWritesListsAndReadsBack(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "saves") // NOT created ahead of time
	s := SaveStore{Dir: dir}

	if got, err := s.List(); err != nil || len(got) != 0 {
		t.Fatalf("List over a missing directory = (%v, %v), want (empty, nil)", got, err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Error("listing created the directory; it is created on the first write alone")
	}

	first := originalWriterLabeled(t, "first save")
	second := originalWriterLabeled(t, "second save")
	n1, err := s.WriteOriginal("", first)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	n2, err := s.WriteOriginal("", second)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n1 == n2 {
		t.Fatalf("two saves landed under one name %q", n1)
	}
	if !strings.HasSuffix(n1, originalSaveExt) {
		t.Errorf("name %q does not carry the extension", n1)
	}

	// Repeated saves of the same bytes claim distinct slots.
	n3, err := s.WriteOriginal("", first)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n3 == n1 {
		t.Fatalf("a repeated save overwrote %q", n1)
	}

	list, err := s.List()
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 3 {
		t.Fatalf("List found %d saves, want 3", len(list))
	}
	labels := map[string]string{}
	for _, e := range list {
		labels[e.Name] = e.Label
	}
	if labels[n2] != "second save - between missions" {
		t.Errorf("the label out of %s's header is %q", n2, labels[n2])
	}

	b, err := s.Read(n2)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if !bytes.Equal(b, second) {
		t.Fatal("read back bytes differ from the complete second save")
	}
}

var (
	errSaveBoundary        = errors.New("injected save boundary")
	errSaveRemoveBoundary  = errors.New("injected temporary-save removal boundary")
	errSaveCleanupBoundary = errors.New("injected temporary-save close boundary")
)

type faultSaveOps struct {
	fail                 string
	removeFailures       int
	cleanupCloseFailures int
	calls                []string
}

func (o *faultSaveOps) CreateTemp(dir, pattern string) (saveTempFile, error) {
	o.calls = append(o.calls, "create")
	if o.fail == "create" {
		return nil, errSaveBoundary
	}
	f, err := os.CreateTemp(dir, pattern)
	if err != nil {
		return nil, err
	}
	return &faultSaveFile{File: f, owner: o}, nil
}

func (o *faultSaveOps) Publish(oldPath, newPath string) (savePublishResult, error) {
	o.calls = append(o.calls, "publish")
	if o.fail == "publish" {
		return savePublishResult{}, errSaveBoundary
	}
	return publishSaveFile(oldPath, newPath)
}

func (o *faultSaveOps) Remove(path string) error {
	o.calls = append(o.calls, "remove")
	if o.removeFailures > 0 {
		o.removeFailures--
		return errSaveRemoveBoundary
	}
	return os.Remove(path)
}

type faultSaveFile struct {
	*os.File
	owner *faultSaveOps
}

func (f *faultSaveFile) Write(p []byte) (int, error) {
	f.owner.calls = append(f.owner.calls, "write")
	if f.owner.fail == "write" {
		if len(p) == 0 {
			return 0, errSaveBoundary
		}
		n, _ := f.File.Write(p[:1])
		return n, errSaveBoundary
	}
	return f.File.Write(p)
}

func (f *faultSaveFile) Sync() error {
	f.owner.calls = append(f.owner.calls, "sync")
	if f.owner.fail == "sync" {
		return errSaveBoundary
	}
	return f.File.Sync()
}

func (f *faultSaveFile) Close() error {
	f.owner.calls = append(f.owner.calls, "close")
	err := f.File.Close()
	if f.owner.fail == "close" {
		return errSaveBoundary
	}
	if f.owner.cleanupCloseFailures > 0 {
		f.owner.cleanupCloseFailures--
		return errSaveCleanupBoundary
	}
	return err
}

// TestAtomicSavePublicationBoundaries is 1008 AC-1 and AC-2. It drives the
// exact file-operation seam SaveStore.WriteOriginal uses and leaves every path inside
// t.TempDir. The write failure writes one byte first, so cleanup is proved over
// an actual incomplete sibling rather than an empty stand-in.
func TestAtomicSavePublicationBoundaries(t *testing.T) {
	baseName := "game0000.sav"
	old := []byte("an earlier complete save")
	data := []byte("the new complete save")

	for _, boundary := range []string{"create", "write", "sync", "close", "publish"} {
		t.Run(boundary, func(t *testing.T) {
			dir := t.TempDir()
			oldPath := filepath.Join(dir, baseName)
			if err := os.WriteFile(oldPath, old, 0o644); err != nil {
				t.Fatal(err)
			}
			ops := &faultSaveOps{fail: boundary}
			store := SaveStore{Dir: dir, files: ops}
			if name, err := store.WriteOriginal("", data); !errors.Is(err, errSaveBoundary) || name != "" {
				t.Fatalf("Write at %s = (%q, %v), want empty name and injected error", boundary, name, err)
			}
			gotOld, err := os.ReadFile(oldPath)
			if err != nil || !reflect.DeepEqual(gotOld, old) {
				t.Fatalf("earlier save after %s = (%q, %v), want unchanged", boundary, gotOld, err)
			}
			entries, err := os.ReadDir(dir)
			if err != nil {
				t.Fatal(err)
			}
			if len(entries) != 1 || entries[0].Name() != baseName {
				t.Fatalf("directory after %s = %v, want only %s", boundary, entryNames(entries), baseName)
			}
		})
	}

	t.Run("success order and unique publication", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, baseName), old, 0o644); err != nil {
			t.Fatal(err)
		}
		ops := &faultSaveOps{}
		name, err := (SaveStore{Dir: dir, files: ops}).WriteOriginal("", data)
		if err != nil {
			t.Fatal(err)
		}
		if name != "game0001.sav" {
			t.Fatalf("published name = %q", name)
		}
		if want := []string{"create", "write", "sync", "close", "publish", "publish"}; !reflect.DeepEqual(ops.calls, want) {
			t.Fatalf("operation order = %v, want %v", ops.calls, want)
		}
		got, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil || !reflect.DeepEqual(got, data) {
			t.Fatalf("published save = (%q, %v), want complete data", got, err)
		}
	})
}

// TestSaveFailureReportsCleanupRefusalAndRecoversOnRetry covers the boundary
// after the primary save has already failed. Physical deletion can itself be
// refused by the operating system, so the caller must receive both failures
// and the next save must recover the hidden staging file before creating a new
// one. The write and sync cases also force Close to fail during cleanup.
func TestSaveFailureReportsCleanupRefusalAndRecoversOnRetry(t *testing.T) {
	baseName := "game0000.sav"
	old := originalWriterLabeled(t, "old complete save")
	data := originalWriterLabeled(t, "new complete save")

	for _, boundary := range []string{"write", "sync", "close", "publish"} {
		t.Run(boundary, func(t *testing.T) {
			dir := t.TempDir()
			oldPath := filepath.Join(dir, baseName)
			if err := os.WriteFile(oldPath, old, 0o644); err != nil {
				t.Fatal(err)
			}
			ops := &faultSaveOps{fail: boundary, removeFailures: 1}
			if boundary == "write" || boundary == "sync" {
				ops.cleanupCloseFailures = 1
			}
			store := SaveStore{Dir: dir, files: ops}
			name, err := store.WriteOriginal("", data)
			if name != "" || !errors.Is(err, errSaveBoundary) || !errors.Is(err, errSaveRemoveBoundary) {
				t.Fatalf("failed Write = (%q, %v), want primary and cleanup removal failures", name, err)
			}
			if boundary == "write" || boundary == "sync" {
				if !errors.Is(err, errSaveCleanupBoundary) {
					t.Fatalf("failed Write error = %v, want cleanup close failure too", err)
				}
			}
			gotOld, readErr := os.ReadFile(oldPath)
			if readErr != nil || !reflect.DeepEqual(gotOld, old) {
				t.Fatalf("earlier save after %s = (%q, %v), want byte-exact", boundary, gotOld, readErr)
			}
			entries, readErr := os.ReadDir(dir)
			if readErr != nil {
				t.Fatal(readErr)
			}
			var temps []string
			for _, entry := range entries {
				if looksLikeSaveTemp(entry.Name()) {
					temps = append(temps, entry.Name())
				}
			}
			if len(temps) != 1 {
				t.Fatalf("hidden staging files after refused cleanup = %v, want one", temps)
			}
			listed, listErr := store.List()
			if listErr != nil || len(listed) != 1 || listed[0].Name != baseName {
				t.Fatalf("visible saves after refused cleanup = (%v, %v), want only %s", listed, listErr, baseName)
			}

			ops.fail = ""
			name, err = store.WriteOriginal("", data)
			if err != nil || name != "game0001.sav" {
				t.Fatalf("retry Write = (%q, %v), want recovered unique publication", name, err)
			}
			entries, readErr = os.ReadDir(dir)
			if readErr != nil {
				t.Fatal(readErr)
			}
			for _, entry := range entries {
				if looksLikeSaveTemp(entry.Name()) {
					t.Fatalf("recovered Write left staging file %q", entry.Name())
				}
			}
			gotOld, readErr = os.ReadFile(oldPath)
			if readErr != nil || !reflect.DeepEqual(gotOld, old) {
				t.Fatalf("earlier save after recovery = (%q, %v), want byte-exact", gotOld, readErr)
			}
			gotNew, readErr := os.ReadFile(filepath.Join(dir, name))
			if readErr != nil || !reflect.DeepEqual(gotNew, data) {
				t.Fatalf("new save after recovery = (%q, %v), want byte-exact", gotNew, readErr)
			}
		})
	}
}

func TestSaveRecoveryRetriesRemovalBeforeCreatingAnotherTemp(t *testing.T) {
	dir := t.TempDir()
	ops := &faultSaveOps{fail: "publish", removeFailures: 2}
	store := SaveStore{Dir: dir, files: ops}
	if _, err := store.WriteOriginal("", []byte("complete")); !errors.Is(err, errSaveRemoveBoundary) {
		t.Fatalf("first Write error = %v, want refused cleanup", err)
	}
	creates := countStrings(ops.calls, "create")
	ops.fail = ""
	if name, err := store.WriteOriginal("", []byte("complete")); name != "" || !errors.Is(err, errSaveRemoveBoundary) {
		t.Fatalf("recovery refusal = (%q, %v), want removal error and no save", name, err)
	}
	if got := countStrings(ops.calls, "create"); got != creates {
		t.Fatalf("recovery refusal created another temp: create count %d, want %d", got, creates)
	}
	if name, err := store.WriteOriginal("", []byte("complete")); err != nil || name != "game0000.sav" {
		t.Fatalf("eventual recovery Write = (%q, %v)", name, err)
	}
}

func TestSaveRecoveryDoesNotRemoveAnotherProcessTemp(t *testing.T) {
	dir := t.TempDir()
	foreign := filepath.Join(dir, ".save-original.sav.tmp-foreign-process-1")
	if err := os.WriteFile(foreign, []byte("another writer is still using this namespace"), 0o600); err != nil {
		t.Fatal(err)
	}
	name, err := (SaveStore{Dir: dir}).WriteOriginal("", []byte("complete"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "game0000.sav" {
		t.Fatalf("published name = %q", name)
	}
	if got, err := os.ReadFile(foreign); err != nil || string(got) != "another writer is still using this namespace" {
		t.Fatalf("foreign staging file after save = (%q, %v), want untouched", got, err)
	}
}

func countStrings(values []string, target string) int {
	var n int
	for _, value := range values {
		if value == target {
			n++
		}
	}
	return n
}

func looksLikeSaveTemp(name string) bool {
	return strings.HasPrefix(name, ".save-") && strings.Contains(name, originalSaveExt+".tmp-")
}

func entryNames(entries []os.DirEntry) []string {
	names := make([]string, len(entries))
	for i := range entries {
		names[i] = entries[i].Name()
	}
	return names
}

func TestTheStoreRefusesANameThatIsNotABareFileName(t *testing.T) {
	s := SaveStore{Dir: t.TempDir()}
	for _, name := range []string{"", "..", ".", "../x.ags", `sub\x.ags`, "sub/x.ags"} {
		if _, err := s.Read(name); err == nil {
			t.Errorf("Read(%q) was accepted", name)
		}
	}
}

// TestAFileThatWillNotReadIsSkippedAndTheListIsStillShown is List's own rule:
// one corrupt save is not the reason the player cannot load the others.
func TestDefaultSaveDirIsBesideTheBinaryAndOverridable(t *testing.T) {
	want := filepath.Join(t.TempDir(), "elsewhere")
	got, err := DefaultSaveDir(want)
	if err != nil || got != want {
		t.Fatalf("DefaultSaveDir(override) = (%q, %v), want (%q, nil)", got, err, want)
	}
	// WITHOUT AN OVERRIDE THE ANSWER DEPENDS ON WHERE THIS TEST BINARY IS, and
	// `go test` builds into the temporary directory exactly as `go run` does —
	// so a test that demanded "beside the executable" here would be demanding
	// the throwaway path. It asserts what saveDirFor was handed instead, and
	// the choice itself is checked below over paths no machine has to have.
	got, err = DefaultSaveDir("")
	if err != nil {
		t.Fatalf("DefaultSaveDir(\"\") = %v", err)
	}
	exe, err := os.Executable()
	if err != nil {
		t.Skip("this platform cannot locate its own executable")
	}
	wd, err := os.Getwd()
	if err != nil {
		t.Skip("this platform cannot locate its working directory")
	}
	if w := saveDirFor(exe, wd, os.TempDir()); got != w {
		t.Errorf("DefaultSaveDir(\"\") = %q, want %q", got, w)
	}
}

func TestASourceLaunchDoesNotSaveIntoTheDirectoryItIsAboutToLose(t *testing.T) {
	sep := string(filepath.Separator)
	tmp := filepath.Join(sep+"m", "AppData", "Local", "Temp")
	installed := filepath.Join(sep+"games", "rom")
	module := filepath.Join(sep+"m", "againrom", "implementation")
	for _, c := range []struct {
		name, exe, wd, want string
	}{{
		name: "an installed build saves beside itself",
		exe:  filepath.Join(installed, "againrom.exe"),
		wd:   filepath.Join(sep+"somewhere", "else"),
		want: filepath.Join(installed, "saves"),
	}, {
		name: "a go run build saves beside the module it was run from",
		exe:  filepath.Join(tmp, "go-build292600318", "b001", "exe", "againrom.exe"),
		wd:   module,
		want: filepath.Join(module, "saves"),
	}, {
		// The two readings come from different OS calls and neither promises
		// the other's casing.
		name: "casing does not decide it",
		exe:  filepath.Join(strings.ToLower(tmp), "go-build1", "b001", "exe", "againrom.exe"),
		wd:   module,
		want: filepath.Join(module, "saves"),
	}, {
		// A directory whose NAME merely starts with the temp path is not
		// inside it, which is the off-by-one this comparison is prone to.
		name: "a sibling of the temp directory is not inside it",
		exe:  filepath.Join(tmp+"Files", "againrom.exe"),
		wd:   module,
		want: filepath.Join(tmp+"Files", "saves"),
	}} {
		if got := saveDirFor(c.exe, c.wd, tmp); got != c.want {
			t.Errorf("%s: saveDirFor(%q, %q) = %q, want %q", c.name, c.exe, c.wd, got, c.want)
		}
	}
}

func TestASourceLaunchThroughTheBuildCacheIsAlsoASourceLaunch(t *testing.T) {
	sep := string(filepath.Separator)
	local := filepath.Join(sep+"m", "AppData", "Local")
	tmp := filepath.Join(local, "Temp")
	cache := filepath.Join(local, "go-build")
	module := filepath.Join(sep+"m", "againrom", "implementation")

	exe := filepath.Join(cache, "b8", "b86b166f330e5f76a657ef0249137fd31a94af94-d", "againrom.exe")
	want := filepath.Join(module, "saves")
	if got := saveDirFor(exe, module, tmp, cache); got != want {
		t.Errorf("a build-cache launch saves to %q, want %q", got, want)
	}

	// The cache is NOT inside the temporary directory, which is why naming the
	// temporary directory alone could never have caught this. Stated as its own
	// assertion so a machine that arranged the two differently would say so
	// here rather than making the case above pass for the wrong reason.
	if insideDir(tmp, cache) {
		t.Fatalf("this test's own premise is wrong: %q is inside %q", cache, tmp)
	}

	// An installed build is still an installed build with both roots handed in.
	installed := filepath.Join(sep+"games", "rom")
	if got, w := saveDirFor(filepath.Join(installed, "againrom.exe"), module, tmp, cache),
		filepath.Join(installed, "saves"); got != w {
		t.Errorf("an installed build saves to %q, want %q", got, w)
	}
}

// TestToolchainRootsNamesTheBuildCache is the machine-reading half: the choice
// above is only reached for a real launch if the cache is actually among the
// roots DefaultSaveDir hands it.
func TestToolchainRootsNamesTheBuildCache(t *testing.T) {
	t.Setenv("GOCACHE", filepath.Join(t.TempDir(), "go-build"))
	roots := toolchainRoots()
	if !insideAny(roots, filepath.Join(os.Getenv("GOCACHE"), "b8", "x-d")) {
		t.Errorf("toolchainRoots() = %v, and none of them holds $GOCACHE", roots)
	}
	if !insideAny(roots, filepath.Join(os.TempDir(), "go-build1", "b001", "exe")) {
		t.Errorf("toolchainRoots() = %v, and none of them holds the temporary directory", roots)
	}
	if insideAny(roots, filepath.Join(string(filepath.Separator)+"games", "rom")) {
		t.Errorf("toolchainRoots() = %v, and an installed path reads as one of them", roots)
	}
}

func TestSavingAPlainMapIsRefused(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: saveTown(t)}}
	f.liveDriver(&mapWorld{}, 0, nil)
	if _, _, err := f.Snapshot(true); err == nil {
		t.Fatal("a plain map was saved")
	} else if !strings.Contains(err.Error(), "list") {
		t.Errorf("the refusal %q does not say what was refused", err)
	}
}

// TestSavingBeforeAnyGameIsRefused: a front end that has reached nothing has
// nothing to write down.
func TestSavingBeforeAnyGameIsRefused(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: NewTown(saveCampaign())}}
	if _, _, err := f.Snapshot(false); err == nil {
		t.Fatal("a game that has not started was saved")
	}
}

func TestTheResidueRidesAndComesBack(t *testing.T) {
	from := &mapWorld{
		commanded: map[sim.EntityID]bool{4: true, 9: true},
		swing:     map[sim.EntityID]int{4: 3},
		phase:     map[sim.EntityID]sim.AttackPhase{4: sim.AttackCharging},
		groupTag:  17,
		fog:       newFogPlane(4, 2),
	}
	from.fog.explored[5] = 1

	r := from.residue()
	sort.Slice(r.Commanded, func(i, j int) bool { return r.Commanded[i] < r.Commanded[j] })
	if !reflect.DeepEqual(r.Commanded, []uint32{4, 9}) {
		t.Errorf("commanded = %v, want [4 9]", r.Commanded)
	}

	to := &mapWorld{
		commanded: map[sim.EntityID]bool{},
		swing:     map[sim.EntityID]int{},
		phase:     map[sim.EntityID]sim.AttackPhase{},
		fog:       newFogPlane(4, 2),
		view:      nil,
	}
	to.applyResidue(r)
	if !to.commanded[4] || !to.commanded[9] {
		t.Error("commanded did not come back — resumed units would fall back to the placeholder script")
	}
	if to.swing[4] != 3 || to.phase[4] != sim.AttackCharging {
		t.Errorf("the swing memory came back as %d/%v", to.swing[4], to.phase[4])
	}
	if to.groupTag != 17 {
		t.Errorf("groupTag = %d, want 17", to.groupTag)
	}
	if to.fog.explored[5] != 1 {
		t.Error("what had been explored did not come back")
	}
}

// TestAFogPlaneOfAnotherSizeIsRefused is plan D-3's mismatch rule: a plane whose
// map changed under the save is dropped rather than painted onto the wrong
// cells.
func TestAFogPlaneOfAnotherSizeIsRefused(t *testing.T) {
	to := &mapWorld{
		commanded: map[sim.EntityID]bool{},
		swing:     map[sim.EntityID]int{},
		phase:     map[sim.EntityID]sim.AttackPhase{},
		fog:       newFogPlane(4, 2),
	}
	to.applyResidue(SnapshotResidue{FogCols: 8, FogRows: 8, FogExplored: bytesOf(64, 1), Commanded: []uint32{1}})
	for i, b := range to.fog.explored {
		if b != 0 {
			t.Fatalf("cell %d was painted from a plane of another size", i)
		}
	}
	if !to.commanded[1] {
		t.Error("the rest of the residue was dropped with the fog plane; only the plane is refused")
	}
}

func bytesOf(n int, v byte) []byte {
	b := make([]byte, n)
	for i := range b {
		b[i] = v
	}
	return b
}

func TestEveryMapWorldFieldIsRuled(t *testing.T) {
	rides := map[string]bool{"applicationState": true, "world": true, "commanded": true, "swing": true, "phase": true,
		"groupTag": true, "fog": true, "pending": true, "pendingIgnored": true, "castRun": true, "healBursts": true,
		"visualIDs": true, "visualNext": true}
	derivable := map[string]bool{
		"playerColours":     true,
		"cheats":            true,
		"visualLocalNext":   true,
		"fame":              true,
		"projectileCounter": true,
		"sched":             true, "units": true, "view": true, "clock": true, "last": true,
		"sounds": true, "swingSound": true, "spellSound": true, "semanticSound": true, "tiers": true, "chars": true, "art": true,
		"figures": true, "npcFaces": true, "speakerActors": true, "actorNames": true,
		"invParty": true, "spellNames": true,
		"portraits": true, "figurePics": true, "figureMasks": true, "ownerFrames": true, "structureStateScratch": true, "structureOwners": true, "invIconCache": true,
		"spellAtlasImg": true, "spellAtlasTried": true, "spellIcons": true,
		"invSubject": true, "invSubjectSet": true, "invCodes": true,
		"invEquipment": true, "invEquipmentItems": true,
		"invFigureEquipment": true, "invFigureEquipmentItems": true, "invComposedEquipment": true,
		"invDollSuppressSlot": true, "invSkill": true, "invTraining": true,
		"invSkillSet": true, "skillPosted": true, "bodyEquipment": true,
		"invWeaponEverEquipped": true, "invLayers": true, "invFigureLayers": true,
		"stopped": true, "unpaced": true, "mission": true, "projectiles": true,
		"derives": true, "derivedSkills": true, "skillBonus": true, "derivedPotions": true,
	}
	cosmetic := map[string]bool{"prev": true, "stamped": true, "walk": true, "relocated": true, "died": true, "hurt": true, "blows": true, "strikes": true, "scene": true,
		"pendingDamage": true, "soundEntities": true,
		"spellSoundCues": true,
		"markElements":   true, "pickup": true, "shots": true, "flights": true, "drawnMoving": true,
		"weaponSwap": true}

	ft := reflect.TypeOf(mapWorld{})
	for i := 0; i < ft.NumField(); i++ {
		name := ft.Field(i).Name
		n := 0
		for _, set := range []map[string]bool{rides, derivable, cosmetic} {
			if set[name] {
				n++
			}
		}
		if n != 1 {
			t.Errorf("mapWorld field %q is ruled %d times in docs/0143-save-and-load/spec.md FR-5, want exactly 1", name, n)
		}
	}
	for _, set := range []map[string]bool{rides, derivable, cosmetic} {
		for name := range set {
			if _, ok := ft.FieldByName(name); !ok {
				t.Errorf("FR-5 rules a field %q that mapWorld no longer has", name)
			}
		}
	}
	if got, want := ft.NumField(), len(rides)+len(derivable)+len(cosmetic); got != want {
		t.Errorf("mapWorld has %d fields and FR-5 rules %d", got, want)
	}
}

// TestTheWorldHalfIsTheSimulationsOwnBytes: the envelope carries them opaque and
// this story authors nothing inside them, so a world out and back through a save
// is the world sim.MarshalBinary already round-trips.

func TestAStaleSimulationVersionIsRefusedByTheSimulation(t *testing.T) {
	w, err := sim.NewWorld(7, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), nil)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	raw[0] = 49 // predates oldestReadableVersion (50), refused by name
	back := Snapshot{Mission: 10, World: raw}

	ms := &Mission{World: &sim.World{}}
	if err := resumeWorld(ms, &back, nil); err == nil {
		t.Fatal("a stale simulation version was accepted")
	}
}

func TestTheSaveSeamsCrossOnlyStringsAndBools(t *testing.T) {
	f, _ := originalCityRouteFixture(t)
	dir := filepath.Join(t.TempDir(), "saves")
	at := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	save, list, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return at })

	name, err := save(false)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	rows := list()
	if len(rows) != 1 || rows[0].Name != localOriginalSaveToken(name) {
		t.Fatalf("list = %+v, want the one save just written", rows)
	}
	if !strings.Contains(rows[0].Label, "town") {
		t.Errorf("label %q does not say what the save is", rows[0].Label)
	}

	g := &FrontEnd{InstallResources: f.InstallResources}
	gsave, _, gload := g.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	_ = gsave
	open, town, err := gload(rows[0].Name)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !town || open != nil {
		t.Fatalf("load = (open!=nil %v, town %v), want (false, true)", open != nil, town)
	}
	if g.Town.Gold() != f.Town.Gold() {
		t.Errorf("the loaded game has %d gold, want %d", g.Town.Gold(), f.Town.Gold())
	}
	if _, _, err := gload("no-such-file.ags"); err == nil {
		t.Error("loading a name that is not there was accepted")
	}
}

// TestASaveTakenInTheTownAfterAMissionIsATownSave is the hole the onMap
// argument closes: f.live is set when a map opens and nothing on this side runs
// when the map screen is left, so a save taken in the town after a finished
// mission would otherwise be written as that mission's.
func TestASaveTakenInTheTownAfterAMissionIsATownSave(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: saveTown(t), Carried: saveParty()}}
	w, err := sim.NewWorld(3, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), nil)
	if err != nil {
		t.Fatal(err)
	}
	f.liveDriver(&mapWorld{world: w, commanded: map[sim.EntityID]bool{},
		swing: map[sim.EntityID]int{}, phase: map[sim.EntityID]sim.AttackPhase{}}, 10, saveParty())

	onMap, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatalf("Snapshot(true): %v", err)
	}
	if onMap.Mission != 10 || len(onMap.World) == 0 {
		t.Fatalf("a save taken on the map screen carries mission %d and %d world bytes",
			onMap.Mission, len(onMap.World))
	}

	inTown, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatalf("Snapshot(false): %v", err)
	}
	if inTown.Mission != 0 || len(inTown.World) != 0 {
		t.Fatalf("a save taken in the town after mission %d carries %d world bytes — "+
			"it is the finished mission's world, not the town's", onMap.Mission, len(inTown.World))
	}
	if !strings.Contains(label, "town") {
		t.Errorf("label = %q, want it to name the town", label)
	}
}

func saveTestArchive(t *testing.T, files ...synth.File) *Archives {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, ScenarioArchive)
	if err := os.WriteFile(path, synth.Archive(files), 0o644); err != nil {
		t.Fatal(err)
	}
	containers, err := vfs.Open([]string{path}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return &Archives{Containers: containers}
}

// TestFailedCandidateLoadKeepsTheRunningGame drives the real App -> LoadGame
// -> FrontEnd seam over a live synthetic mission. Every case passes the .ags
// envelope and gob decoder. The failure occurs while the candidate mission is
// being built, after the old screen is already visible behind the load window.
func TestFailedCandidateLoadKeepsTheRunningGame(t *testing.T) {
	rewrite := func(t *testing.T, raw []byte, edit func(*sav.DocumentData)) []byte {
		t.Helper()
		doc, err := sav.DecodeDocumentData(raw)
		if err != nil {
			t.Fatal(err)
		}
		edit(&doc)
		raw, err = sav.EncodeDocumentData(doc)
		if err != nil {
			t.Fatal(err)
		}
		return raw
	}
	cases := []struct {
		name  string
		alter func(*testing.T, *FrontEnd, []byte) []byte
		want  string
	}{
		{
			name: "invalid SAV signature",
			alter: func(t *testing.T, _ *FrontEnd, raw []byte) []byte {
				raw[0] = '!'
				return raw
			},
			want: "magic",
		},
		{
			name: "invalid campaign difficulty",
			alter: func(t *testing.T, _ *FrontEnd, raw []byte) []byte {
				return rewrite(t, raw, func(doc *sav.DocumentData) { doc.Head.Difficulty = 4 })
			},
			want: "difficulty",
		},
		{
			name: "truncated SAV header",
			alter: func(t *testing.T, _ *FrontEnd, raw []byte) []byte {
				return raw[:12]
			},
			want: "shorter",
		},
		{
			name: "mission map cannot be read",
			alter: func(t *testing.T, f *FrontEnd, raw []byte) []byte {
				f.Archives = saveTestArchive(t)
				return raw
			},
			want: "read scenario/10.alm",
		},
		{
			name: "mission map cannot be decoded",
			alter: func(t *testing.T, f *FrontEnd, raw []byte) []byte {
				f.Archives = saveTestArchive(t, synth.File{Path: "10.alm", Data: []byte("not an alm")})
				return raw
			},
			want: "decode scenario/10.alm",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := currentPoolFixtureFront(t, 91, 92)
			campaign := saveCampaign()
			campaign.Side = []int{11, 12}
			f.Campaign = resolved(campaign, nil)
			f.Town = NewTown(campaign)
			f.Town.Arrive()
			if mission, ok := f.Town.Take(TownShop, 0); !ok || mission != 11 {
				t.Fatal("prepare taken shop offer", mission, ok)
			}
			f.Town.gold += 250
			f.Carried = f.NextParty()
			f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
			f.Offered = 20
			townScreen := f.TownScreen().(*townScreen)
			townScreen.room, townScreen.talkPage.part = roomTavern, 2

			app := f.App("transactional-load")
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatalf("open current mission: %v", err)
			}
			snap, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatalf("snapshot current mission: %v", err)
			}

			oldTown, oldLive, oldWorld := f.Town, f.live, f.live.world
			oldCarried := mapload.CloneParty(f.Carried)
			oldOffered := f.Offered
			oldBytes, err := oldWorld.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			oldHash := oldWorld.Hash()

			payload, err := f.ExportCurrentSave(snap, tc.name)
			if err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			name, err := store.WriteOriginal("", payload)
			if err != nil {
				t.Fatal(err)
			}
			app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))

			if err := app.HeadlessKey("escape"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessKey("down"); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessKey("enter"); err != nil {
				t.Fatal(err)
			}
			if app.Screen() != ui.ScreenLoad || app.HeadlessGameplayScreen() != ui.ScreenMap {
				t.Fatalf("load window = screen %v over %v", app.Screen(), app.HeadlessGameplayScreen())
			}
			// Corrupt the listed temporary fixture after the load window captured its row.
			if err := os.WriteFile(filepath.Join(store.Dir, name), tc.alter(t, f, payload), 0o644); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessActivate(tc.name); err != nil {
				t.Fatal(err)
			}

			if app.Screen() != ui.ScreenLoad || app.HeadlessGameplayScreen() != ui.ScreenMap {
				t.Fatalf("refusal left screen %v over %v, want load over map", app.Screen(), app.HeadlessGameplayScreen())
			}
			if !strings.Contains(strings.ToLower(app.HeadlessMessage()), strings.ToLower(tc.want)) {
				t.Fatalf("refusal = %q, want %q", app.HeadlessMessage(), tc.want)
			}
			if f.Town != oldTown || f.live != oldLive || f.live.world != oldWorld {
				t.Fatal("failed load replaced Town, live driver or world pointer")
			}
			if !reflect.DeepEqual(f.Carried, oldCarried) || f.Offered != oldOffered {
				t.Fatalf("failed load changed carried/offered to (%+v, %d)", f.Carried, f.Offered)
			}
			if townScreen.room != roomTavern || townScreen.talkPage.part != 2 {
				t.Fatalf("failed load moved town UI to room %v line %d", townScreen.room, townScreen.talkPage.part)
			}
			gotBytes, err := f.live.world.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			if f.live.world.Hash() != oldHash || !reflect.DeepEqual(gotBytes, oldBytes) {
				t.Fatal("failed load changed the running world's hash or canonical bytes")
			}
		})
	}
}

func TestOversizedSaveRefusalKeepsTheRunningGame(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	f.Campaign = resolved(saveCampaign(), nil)
	f.Town = saveTown(t)
	f.Carried = f.NextParty()
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	app := f.App("bounded-load")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	snap, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := f.ExportCurrentSave(snap, "oversized on disk")
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	name, err := store.WriteOriginal("", payload)
	if err != nil {
		t.Fatal(err)
	}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("down"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenLoad || app.HeadlessGameplayScreen() != ui.ScreenMap {
		t.Fatalf("load window = screen %v over %v", app.Screen(), app.HeadlessGameplayScreen())
	}
	oldTown, oldLive, oldWorld := f.Town, f.live, f.live.world
	oldHash := oldWorld.Hash()
	if err := os.Truncate(filepath.Join(store.Dir, name), 64<<20); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("oversized on disk"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenLoad || app.HeadlessGameplayScreen() != ui.ScreenMap {
		t.Fatalf("refusal left screen %v over %v, want load over map", app.Screen(), app.HeadlessGameplayScreen())
	}
	if !strings.Contains(app.HeadlessMessage(), "safe maximum") {
		t.Fatalf("oversized refusal = %q, want safe maximum", app.HeadlessMessage())
	}
	if f.Town != oldTown || f.live != oldLive || f.live.world != oldWorld || f.live.world.Hash() != oldHash {
		t.Fatal("oversized load refusal changed the running game")
	}
}

// TestPreparedMissionCommitsOnceThroughTheLoadWindow is the successful half of
// the same production seam. The current game remains live during preparation;
// choosing the row then adopts the already-built driver and enters its viewer.
func TestPreparedMissionCommitsOnceThroughTheLoadWindow(t *testing.T) {
	f := currentPoolFixtureFront(t, 91, 92)
	campaign := saveCampaign()
	campaign.Side = []int{11, 12}
	f.Campaign = resolved(campaign, nil)
	f.Town = NewTown(campaign)
	f.Town.Arrive()
	if mission, ok := f.Town.Take(TownShop, 0); !ok || mission != 11 {
		t.Fatal("prepare taken shop offer", mission, ok)
	}
	f.Town.gold += 250
	f.Carried = f.NextParty()
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	app := f.App("prepared-load")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	snap, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	snap.Gold = 4321
	oldLive := f.live
	wantHash := f.live.world.Hash()
	payload, err := f.ExportCurrentSave(snap, "prepared mission")
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.WriteOriginal("", payload); err != nil {
		t.Fatal(err)
	}
	app.SetSaveSeams(f.SaveSeams(store, OriginalStore{}, nil))
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("down"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("enter"); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate("prepared mission"); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenMap || f.live == nil || f.live == oldLive {
		t.Fatalf("successful prepared load = screen %v live replaced %v", app.Screen(), f.live != oldLive)
	}
	if f.Town.Gold() != 4321 || f.live.world.Hash() != wantHash {
		t.Fatalf("committed town/world = gold %d hash %016x, want 4321/%016x",
			f.Town.Gold(), f.live.world.Hash(), wantHash)
	}
}

// TestLoadingPutsTheTownScreenBackAtTheSquare: the open room belongs to the game
// being left, and a loaded game is entered at the square.
func TestLoadingPutsTheTownScreenBackAtTheSquare(t *testing.T) {
	f := currentTrainingCity(t)
	ts := f.TownScreen().(*townScreen)
	ts.room, ts.talkPage.part = roomTavern, 2
	snapshot, label, err := f.Snapshot(false)
	if err != nil {
		t.Fatal(err)
	}
	raw, err := f.ExportCurrentSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := f.RestoreOriginal(raw); err != nil {
		t.Fatal(err)
	}

	if ts.room != roomSquare || ts.talkPage.part != 0 {
		t.Errorf("the town screen came back in room %v with %d lines said, want the square", ts.room, ts.talkPage.part)
	}
	if got, _ := f.TownScreen().(*townScreen); got != ts {
		t.Error("TownScreen built a second screen; the flow holds the first for the life of the process")
	}
}
