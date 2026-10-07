package game

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
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
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: saveTown(t), Carried: saveParty(), Offered: 20}}

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

	b, err := EncodeSave(snap, label)
	if err != nil {
		t.Fatalf("EncodeSave: %v", err)
	}
	back, gotLabel, err := DecodeSave(b)
	if err != nil {
		t.Fatalf("DecodeSave: %v", err)
	}
	if gotLabel != label {
		t.Errorf("label came back %q, want %q", gotLabel, label)
	}

	// A FRESH front end, as a restarted program is.
	g := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}}
	open, town, err := g.Restore(back)
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
		f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(camp, nil)}, CampaignSession: CampaignSession{Town: newTown()}}
		snap, label, err := f.Snapshot(false)
		if err != nil {
			t.Fatalf("run %d: Snapshot: %v", i, err)
		}
		if len(snap.Taken) != 3 {
			t.Fatalf("run %d: Taken has %d offer(s), want 3", i, len(snap.Taken))
		}
		b, err := EncodeSave(snap, label)
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
	party := mapload.OwnParty(saveParty())
	party[0].ID = "hero"
	party[0].StartingHero = true
	temporary := mapload.OwnParty(party)[0]
	temporary.ID = "temporary:witness"
	temporary.StartingHero = false
	temporary.Temporary = true
	party = append(party, temporary)

	b, err := EncodeSave(Snapshot{Party: party}, "identity")
	if err != nil {
		t.Fatal(err)
	}
	back, _, err := DecodeSave(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back.Party, party) {
		t.Fatalf("saved party identity/membership = %+v, want %+v", back.Party, party)
	}
}

func TestSaveDepartedCharactersRoundTrips(t *testing.T) {
	want := Snapshot{Residue: SnapshotResidue{
		DepartedCharacters: []uint32{109, 107},
		Commanded:          []uint32{106, 108},
	}}
	raw, err := EncodeSave(want, "departed characters")
	if err != nil {
		t.Fatal(err)
	}
	got, label, err := DecodeSave(raw)
	if err != nil || label != "departed characters" || !reflect.DeepEqual(got, want) {
		t.Fatalf("departed character envelope = %+v, label %q, error %v; want %+v", got, label, err, want)
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
	if _, _, err := g.Restore(snap); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	offers := g.Town.Offers(TownShop)
	if len(offers) != 1 || offers[0].Mission != 31 {
		t.Fatalf("the restored town offers %+v, want the NEW install's own shop row", offers)
	}
	if g.Town.Gold() != initialPlayerPurse+250 {
		t.Errorf("gold = %d, want the save's own %d", g.Town.Gold(), initialPlayerPurse+250)
	}
}

// TestEveryRefusalHasItsOwnSentence is AC-3. Each case is one file and one
// distinct message, and none of them panics or loads half a game.
func TestEveryRefusalHasItsOwnSentence(t *testing.T) {
	good, err := EncodeSave(Snapshot{Gold: 1}, "x")
	if err != nil {
		t.Fatalf("EncodeSave: %v", err)
	}

	flipped := append([]byte(nil), good...)
	flipped[len(flipped)-1] ^= 0xff

	stale := append([]byte(nil), good...)
	stale[len(saveMagic)] = saveVersion + 1

	foreign := append([]byte(nil), good...)
	copy(foreign, "Asg&....")

	cases := []struct {
		name string
		in   []byte
		want string
	}{
		{"shorter than a header", good[:4], "shorter than a header"},
		{"a foreign magic", foreign, "not an againrom save"},
		{"a stale version byte", stale, "this build reads version"},
		{"a truncated payload", good[:len(good)-10], "header declares"},
		{"a flipped payload byte", flipped, "corrupt"},
	}
	seen := map[string]bool{}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s, _, err := DecodeSave(tc.in)
			if err == nil {
				t.Fatal("the file was accepted")
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("message %q does not name %q", err, tc.want)
			}
			if !reflect.DeepEqual(s, Snapshot{}) {
				t.Error("a refused file produced a partial snapshot")
			}
			if seen[err.Error()] {
				t.Errorf("message %q is not distinct from another refusal's", err)
			}
			seen[err.Error()] = true
		})
	}
	if _, _, err := DecodeSave(foreign); !errors.Is(err, ErrNotSave) {
		t.Errorf("a foreign file answers %v, want ErrNotSave", err)
	}
}

func TestTheStoreWritesListsAndReadsBack(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "saves") // NOT created ahead of time
	s := SaveStore{Dir: dir}

	if got, err := listAGS(s); err != nil || len(got) != 0 {
		t.Fatalf("List over a missing directory = (%v, %v), want (empty, nil)", got, err)
	}
	if _, err := os.Stat(dir); !errors.Is(err, os.ErrNotExist) {
		t.Error("listing created the directory; it is created on the first write alone")
	}

	first, err := EncodeSave(Snapshot{Gold: 1}, "town — gold 1")
	if err != nil {
		t.Fatal(err)
	}
	second, err := EncodeSave(Snapshot{Gold: 2, Mission: 10}, "mission 10 — gold 2")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, 8, 12, 10, 0, 0, 0, time.UTC)
	n1, err := s.Write(base, first)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	n2, err := s.Write(base.Add(time.Minute), second)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n1 == n2 {
		t.Fatalf("two saves landed under one name %q", n1)
	}
	if !strings.HasSuffix(n1, saveExt) {
		t.Errorf("name %q does not carry the extension", n1)
	}

	// A SECOND SAVE INSIDE ONE SECOND GETS ITS OWN FILE.
	n3, err := s.Write(base, first)
	if err != nil {
		t.Fatalf("Write: %v", err)
	}
	if n3 == n1 {
		t.Fatalf("a save in the same second overwrote %q", n1)
	}

	list, err := listAGS(s)
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
	if labels[n2] != "mission 10 — gold 2" {
		t.Errorf("the label out of %s's header is %q", n2, labels[n2])
	}

	b, err := s.Read(n2)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	snap, _, err := DecodeSave(b)
	if err != nil {
		t.Fatalf("DecodeSave: %v", err)
	}
	if snap.Gold != 2 || snap.Mission != 10 {
		t.Errorf("read back gold %d mission %d, want 2 and 10", snap.Gold, snap.Mission)
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
// exact file-operation seam SaveStore.Write uses and leaves every path inside
// t.TempDir. The write failure writes one byte first, so cleanup is proved over
// an actual incomplete sibling rather than an empty stand-in.
func TestAtomicSavePublicationBoundaries(t *testing.T) {
	at := time.Date(2026, 8, 16, 12, 34, 56, 0, time.UTC)
	baseName := "save-20260816-123456.ags"
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
			if name, err := store.Write(at, data); !errors.Is(err, errSaveBoundary) || name != "" {
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
		name, err := (SaveStore{Dir: dir, files: ops}).Write(at, data)
		if err != nil {
			t.Fatal(err)
		}
		if name != "save-20260816-123456-2.ags" {
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
	at := time.Date(2026, 8, 16, 12, 34, 56, 0, time.UTC)
	baseName := "save-20260816-123456.ags"
	old, err := EncodeSave(Snapshot{Gold: 11}, "old complete save")
	if err != nil {
		t.Fatal(err)
	}
	data, err := EncodeSave(Snapshot{Gold: 22}, "new complete save")
	if err != nil {
		t.Fatal(err)
	}

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
			name, err := store.Write(at, data)
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
			listed, listErr := listAGS(store)
			if listErr != nil || len(listed) != 1 || listed[0].Name != baseName {
				t.Fatalf("visible saves after refused cleanup = (%v, %v), want only %s", listed, listErr, baseName)
			}

			ops.fail = ""
			name, err = store.Write(at, data)
			if err != nil || name != "save-20260816-123456-2.ags" {
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
	at := time.Date(2026, 8, 16, 12, 34, 56, 0, time.UTC)
	dir := t.TempDir()
	ops := &faultSaveOps{fail: "publish", removeFailures: 2}
	store := SaveStore{Dir: dir, files: ops}
	if _, err := store.Write(at, []byte("complete")); !errors.Is(err, errSaveRemoveBoundary) {
		t.Fatalf("first Write error = %v, want refused cleanup", err)
	}
	creates := countStrings(ops.calls, "create")
	ops.fail = ""
	if name, err := store.Write(at, []byte("complete")); name != "" || !errors.Is(err, errSaveRemoveBoundary) {
		t.Fatalf("recovery refusal = (%q, %v), want removal error and no save", name, err)
	}
	if got := countStrings(ops.calls, "create"); got != creates {
		t.Fatalf("recovery refusal created another temp: create count %d, want %d", got, creates)
	}
	if name, err := store.Write(at, []byte("complete")); err != nil || name != "save-20260816-123456.ags" {
		t.Fatalf("eventual recovery Write = (%q, %v)", name, err)
	}
}

func TestSaveRecoveryDoesNotRemoveAnotherProcessTemp(t *testing.T) {
	at := time.Date(2026, 8, 16, 12, 34, 56, 0, time.UTC)
	dir := t.TempDir()
	foreign := filepath.Join(dir, ".save-20260816-123456.ags.tmp-foreign-process-1")
	if err := os.WriteFile(foreign, []byte("another writer is still using this namespace"), 0o600); err != nil {
		t.Fatal(err)
	}
	name, err := (SaveStore{Dir: dir}).Write(at, []byte("complete"))
	if err != nil {
		t.Fatal(err)
	}
	if name != "save-20260816-123456.ags" {
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
	return strings.HasPrefix(name, ".save-") && strings.Contains(name, saveExt+".tmp-")
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

func TestASaveNamingAMissionWithNoWorldIsRefused(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}}
	if _, _, err := f.Restore(Snapshot{Mission: 10}); err == nil {
		t.Fatal("a mission save with no world half was accepted")
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

	// Through the envelope, because that is the trip it actually makes.
	b, err := EncodeSave(Snapshot{Mission: 1, World: []byte{0}, Residue: r}, "x")
	if err != nil {
		t.Fatal(err)
	}
	back, _, err := DecodeSave(b)
	if err != nil {
		t.Fatal(err)
	}

	to := &mapWorld{
		commanded: map[sim.EntityID]bool{},
		swing:     map[sim.EntityID]int{},
		phase:     map[sim.EntityID]sim.AttackPhase{},
		fog:       newFogPlane(4, 2),
		view:      nil,
	}
	to.applyResidue(back.Residue)
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
		"groupTag": true, "fog": true, "pending": true, "pendingIgnored": true, "bolts": true, "castRun": true, "healBursts": true,
		"visualIDs": true, "visualNext": true}
	derivable := map[string]bool{
		"visualLocalNext": true,
		"fame":            true,
		"sched":           true, "units": true, "view": true, "clock": true, "last": true,
		"sounds": true, "swingSound": true, "spellSound": true, "semanticSound": true, "tiers": true, "chars": true, "art": true,
		"figures": true, "npcFaces": true, "speakerActors": true, "actorNames": true,
		"invParty": true, "spellNames": true,
		"portraits": true, "figurePics": true, "figureMasks": true, "ownerFrames": true, "structureStateScratch": true, "invIconCache": true,
		"spellAtlasImg": true, "spellAtlasTried": true, "spellIcons": true,
		"invSubject": true, "invSubjectSet": true, "invCodes": true,
		"invEquipment": true, "invEquipmentItems": true,
		"invFigureEquipment": true, "invFigureEquipmentItems": true, "invComposedEquipment": true,
		"invDollSuppressSlot": true, "invSkill": true,
		"invSkillSet": true, "skillPosted": true, "bodyEquipment": true,
		"invWeaponEverEquipped": true, "invLayers": true, "invFigureLayers": true,
		"stopped": true, "unpaced": true, "mission": true, "projectiles": true,
		"derives": true, "derivedSkills": true, "skillBonus": true, "derivedPotions": true,
	}
	cosmetic := map[string]bool{"prev": true, "walk": true, "died": true, "hurt": true, "blows": true, "strikes": true, "scene": true,
		"pendingDamage": true, "soundEntities": true, "topRowDecided": true,
		"spellSoundCues": true,
		"markElements":   true, "stoneHold": true, "pickup": true, "shots": true, "drawnMoving": true}

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
func TestTheWorldHalfIsTheSimulationsOwnBytes(t *testing.T) {
	w, err := sim.NewWorld(7, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), nil)
	if err != nil {
		t.Fatalf("NewWorld: %v", err)
	}
	raw, err := w.MarshalBinary()
	if err != nil {
		t.Fatalf("MarshalBinary: %v", err)
	}
	b, err := EncodeSave(Snapshot{Mission: 10, World: raw}, "mission 10")
	if err != nil {
		t.Fatal(err)
	}
	back, _, err := DecodeSave(b)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(back.World, raw) {
		t.Fatal("the world half did not come back byte for byte")
	}
	var got sim.World
	if err := got.UnmarshalBinary(back.World); err != nil {
		t.Fatalf("the simulation refused its own bytes out of the envelope: %v", err)
	}
	if got.Tick() != w.Tick() {
		t.Errorf("tick = %d, want %d", got.Tick(), w.Tick())
	}
}

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
	b, err := EncodeSave(Snapshot{Mission: 10, World: raw}, "x")
	if err != nil {
		t.Fatal(err)
	}
	back, _, err := DecodeSave(b)
	if err != nil {
		t.Fatalf("the ENVELOPE must accept it — only the world half is stale: %v", err)
	}
	ms := &Mission{World: &sim.World{}}
	if err := resumeWorld(ms, &back, nil); err == nil {
		t.Fatal("a stale simulation version was accepted")
	}
}

func TestTheSaveSeamsCrossOnlyStringsAndBools(t *testing.T) {
	f, _ := originalCityRouteFixture(t)
	dir := filepath.Join(t.TempDir(), "saves")
	at := time.Date(2026, 8, 12, 9, 0, 0, 0, time.UTC)
	save, list, _ := agsSaveSeams(f, SaveStore{Dir: dir}, OriginalStore{}, func() time.Time { return at })

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
	gsave, _, gload := agsSaveSeams(g, SaveStore{Dir: dir}, OriginalStore{}, nil)
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
	cases := []struct {
		name  string
		alter func(*testing.T, *FrontEnd, *Snapshot)
		want  string
	}{
		{
			name: "stale simulation form",
			alter: func(t *testing.T, _ *FrontEnd, s *Snapshot) {
				t.Helper()
				// The immediately preceding form is deliberately readable. Use
				// 52, a reserved version no released build ever wrote, to keep
				// this arm a decoder refusal rather than a migration test.
				s.World[0] = 52
			},
			want: "too old",
		},
		{
			name: "late source actor manifest binding",
			alter: func(t *testing.T, _ *FrontEnd, s *Snapshot) {
				s.ActorManifest = &SnapshotActorManifest{Version: 1, Actors: []SnapshotActor{{ID: 0}, {ID: 9999}}}
			},
			want: "manifest",
		},
		{
			name: "invalid world after mission construction",
			alter: func(t *testing.T, _ *FrontEnd, s *Snapshot) {
				t.Helper()
				s.World = append([]byte(nil), s.World[:8]...)
			},
			want: "byte form truncated",
		},
		{
			name: "mission map cannot be read",
			alter: func(t *testing.T, _ *FrontEnd, s *Snapshot) {
				t.Helper()
				s.Mission = 20
			},
			want: "read scenario/20.alm",
		},
		{
			name: "mission map cannot be decoded",
			alter: func(t *testing.T, f *FrontEnd, s *Snapshot) {
				t.Helper()
				s.Mission = 20
				f.Archives = saveTestArchive(t, synth.File{Path: "20.alm", Data: []byte("not an alm")})
			},
			want: "decode scenario/20.alm",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := missionFrontEnd(t)
			f.Campaign = resolved(saveCampaign(), nil)
			f.Town = saveTown(t)
			f.Carried = saveParty()
			f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
			f.Offered = 20
			townScreen := f.TownScreen().(*townScreen)
			townScreen.room, townScreen.said = roomTavern, 2

			app := f.App("transactional-load")
			if err := app.OpenMission(f.MissionOpener(10)); err != nil {
				t.Fatalf("open current mission: %v", err)
			}
			snap, _, err := f.Snapshot(true)
			if err != nil {
				t.Fatalf("snapshot current mission: %v", err)
			}
			tc.alter(t, f, &snap)

			oldTown, oldLive, oldWorld := f.Town, f.live, f.live.world
			oldCarried := mapload.CloneParty(f.Carried)
			oldOffered := f.Offered
			oldBytes, err := oldWorld.MarshalBinary()
			if err != nil {
				t.Fatal(err)
			}
			oldHash := oldWorld.Hash()

			payload, err := EncodeSave(snap, tc.name)
			if tc.name == "invalid world after mission construction" {
				if err == nil {
					t.Fatal("current World envelope validation accepted a truncated native form")
				}
				// An untrusted producer can bypass the new formation-pair check.
				// Keep the ordinary LOAD atomicity witness on the same bad World.
				payload = uncheckedDocumentEnvelope1115(t, snap)
				label := []byte(tc.name)
				binary.LittleEndian.PutUint16(payload[9:11], uint16(len(label)))
				payload = append(append(bytes.Clone(payload[:11]), label...), payload[11:]...)
			} else if err != nil {
				t.Fatal(err)
			}
			store := SaveStore{Dir: t.TempDir()}
			if _, err := store.Write(time.Date(2026, 8, 16, 14, 0, 0, 0, time.UTC), payload); err != nil {
				t.Fatal(err)
			}
			app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))

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
			if townScreen.room != roomTavern || townScreen.said != 2 {
				t.Fatalf("failed load moved town UI to room %v line %d", townScreen.room, townScreen.said)
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
	f := missionFrontEnd(t)
	f.Campaign = resolved(saveCampaign(), nil)
	f.Town = saveTown(t)
	f.Carried = saveParty()
	f.Carried[0].StartingHero, f.Carried[0].PlayerCharacter = true, true
	app := f.App("bounded-load")
	if err := app.OpenMission(f.MissionOpener(10)); err != nil {
		t.Fatal(err)
	}
	snap, _, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	payload, err := EncodeSave(snap, "oversized on disk")
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	name, err := store.Write(time.Date(2026, 8, 16, 14, 30, 0, 0, time.UTC), payload)
	if err != nil {
		t.Fatal(err)
	}
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
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
	f := missionFrontEnd(t)
	f.Campaign = resolved(saveCampaign(), nil)
	f.Town = saveTown(t)
	f.Carried = saveParty()
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
	payload, err := EncodeSave(snap, "prepared mission")
	if err != nil {
		t.Fatal(err)
	}
	store := SaveStore{Dir: t.TempDir()}
	if _, err := store.Write(time.Date(2026, 8, 16, 15, 0, 0, 0, time.UTC), payload); err != nil {
		t.Fatal(err)
	}
	app.SetSaveSeams(agsSaveSeams(f, store, OriginalStore{}, nil))
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

func releasedSaveFixtureSnapshot(t *testing.T) Snapshot {
	t.Helper()
	w, err := sim.NewWorld(9, sim.Bounds{Width: 8, Height: 8}, sim.ModeCanonical, make([]byte, 64), nil)
	if err != nil {
		t.Fatal(err)
	}
	if !w.SetPurse(sim.SelfSlot, 777) {
		t.Fatal("fixture purse refused")
	}
	world, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	return Snapshot{
		Open: true, Gold: 321, Won: []int{10}, Available: []int{20}, Offered: 20,
		Mission: 10, World: world, WorldSelectedOnce: []int{10, 20},
	}
}

// preMercenaryStateSaveFixtureBase64 is the committed synthetic .ags from
// master immediately before Snapshot gained its mercenary arrays. It already
// includes PartyMember.WeaponMaterialized. Its decode-only test preserves the
// backward-compatibility witness while the current encoding is pinned by
// currentReleasedSaveFixtureSHA256 below. It contains no installed-game bytes.
const preMercenaryStateSaveFixtureBase64 = "QUdSTVNBVkUBIQByZWxlYXNlZCBlbnZlbG9wZSAxIHNpbXVsYXRpb24gNTOZyJzi8RYAAP+EfwMBAQhTbmFwc2hvdAH/gAABCgEE" +
	"T3BlbgECAAEER29sZAEEAAEDV29uAf+CAAEJQXZhaWxhYmxlAf+CAAEFVGFrZW4B/4YAAQVQYXJ0eQH/nAABB09mZmVyZWQBBAAB" +
	"B01pc3Npb24BBAABBVdvcmxkAQoAAQdSZXNpZHVlAf+eAAAAE/+BAgEBBVtdaW50Af+CAAEEAAAj/4UCAQEUW11nYW1lLlNuYXBz" +
	"aG90T2ZmZXIB/4YAAf+EAAA+/4MDAQENU25hcHNob3RPZmZlcgH/hAABAwEHQ2hhcHRlcgEEAAEIQnVpbGRpbmcBBAABBUluZGV4" +
	"AQQAAAAk/5sCAQEVW11tYXBsb2FkLlBhcnR5TWVtYmVyAf+cAAH/iAAA/gE7/4cDAQELUGFydHlNZW1iZXIB/4gAARYBAklEAQwA" +
	"AQlUZW1wb3JhcnkBAgABBE5hbWUBDAABD1BsYXllckNoYXJhY3RlcgECAAEMU3RhcnRpbmdIZXJvAQIAAQ1NZXJjZW5hcnlUeXBl" +
	"AQYAAQxDb21wYW5pb25OUEMBBAABBUNsYXNzAQQAAQRCb2R5AQwAAQdCb2R5RGlyAQwAAQRNYWdlAQIAAQdQcm9maWxlAf+KAAEJ" +
	"RmlndXJlRGlyAQwAAQpGaWd1cmVGYWNlAQQAAQRIZXJvAf+MAAELS25vd25TcGVsbHMBBgABBFdvcm4B/5AAAQdDYXJyaWVkAf+S" +
	"AAEGV2VhcG9uAf+UAAESV2VhcG9uTWF0ZXJpYWxpemVkAQIAAQVDYXJyeQH/lgABBVNhdmVkAf+YAAAAQf+JAwEBB1Byb2ZpbGUB" +
	"/4oAAQMBB0ZpZ2h0ZXIBAgABDEhlYWx0aENvbHVtbgECAAEKTWFuYUNvbHVtbgECAAAAR/+LAwEBBEhlcm8B/4wAAQUBBEJvZHkB" +
	"BAABCFJlYWN0aW9uAQQAAQRNaW5kAQQAAQZTcGlyaXQBBAABBVNraWxsAf+OAAAAGP+NAQEBCFs2XWludDMyAf+OAAEEAQwAABr/" +
	"jwEBAQpbMTJddWludDE2Af+QAAEGARgAABb/kQIBAQhbXXVpbnQxNgH/kgABBgAA/7f/kwMBAQZXZWFwb24B/5QAAQ0BBE5hbWUB" +
	"DAABA1JvdwEEAAEEQ29kZQEGAAEKRGFtYWdlQmFzZQEEAAEMRGFtYWdlU3ByZWFkAQQAAQVUb0hpdAEEAAEHRGVmZW5jZQEEAAEK" +
	"QXR0YWNrVHlwZQEEAAEKQ2hhcmdlVGltZQEEAAEJUmVsYXhUaW1lAQQAAQVSYW5nZQEEAAEJU3BlbGxOYW1lAQwAAQpTcGVsbFBv" +
	"d2VyAQQAAAA5/5UDAQEFQ2FycnkB/5YAAQMBB1NraWxsWFAB/44AAQVJdGVtcwH/kgABCEVxdWlwcGVkAf+QAAAAe/+XAwEBBVNh" +
	"dmVkAf+YAAEIAQRDZWxsAf+aAAECSFABBAABBU1heEhQAQQAAQRNYW5hAQQAAQdNYXhNYW5hAQQAARFIZWFsdGhSZWdlblBlcmlv" +
	"ZAEEAAEPTWFuYVJlZ2VuUGVyaW9kAQQAAQlNYXBVbml0SUQBBgAAAB7/mQMBAQRDZWxsAf+aAAECAQFYAQQAAQFZAQQAAAB3/50D" +
	"AQEPU25hcHNob3RSZXNpZHVlAf+eAAEHAQlDb21tYW5kZWQB/6AAAQVTd2luZwH/ogABBVBoYXNlAf+kAAEIR3JvdXBUYWcBBgAB" +
	"B0ZvZ0NvbHMBBAABB0ZvZ1Jvd3MBBAABC0ZvZ0V4cGxvcmVkAQoAAAAW/58CAQEIW111aW50MzIB/6AAAQYAAB7/oQQBAQ5tYXBb" +
	"dWludDMyXWludAH/ogABBgEEAAAg/6MEAQEQbWFwW3VpbnQzMl11aW50OAH/pAABBgEGAAD+EVz/gAEBAf4CggEBFAEBKAMoARQB" +
	"/hFDNQAAAAAAAAAACQAAAAAAAAAIAAAACAAAAAAAAAAAQAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI" +
	"CAgICAgICAgICAgICAgIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAkDAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAgIC" +
	"AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgIAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAQAA"

// releasedSaveFixtureLabel is the envelope label the encode-side fixture is
// written and read at. It carries no version number, on the test's own reason
// below: the label is part of the pinned bytes, so a number in it would have to
// be re-spelled at every byte-form bump.
const releasedSaveFixtureLabel = "released envelope 1 at the current simulation form"

func TestPreMercenaryStateSaveFixtureStillDecodes(t *testing.T) {
	old, err := base64.StdEncoding.DecodeString(preMercenaryStateSaveFixtureBase64)
	if err != nil {
		t.Fatal(err)
	}
	s, label, err := DecodeSave(old)
	if err != nil {
		t.Fatalf("pre-mercenary-state fixture no longer loads: %v", err)
	}
	if label != "released envelope 1 simulation 53" || !s.Open || s.Gold != 321 ||
		s.Mission != 10 || s.Offered != 20 || !reflect.DeepEqual(s.Won, []int{10}) ||
		!reflect.DeepEqual(s.Available, []int{20}) || s.MercenaryState ||
		len(s.WorldSelectedOnce) != 0 || s.Residue.DepartedCharacters != nil {
		t.Fatalf("pre-mercenary-state fixture decoded as label %q snapshot %+v", label, s)
	}
}

// preWeaponMaterializedSaveFixtureBase64 is releasedSaveFixtureBase64's own
// bytes AS COMMITTED ON master BEFORE PartyMember.WeaponMaterialized existed
// (round-2 adversarial review, twelfth pass, C3, DIV-095): retrieved with
// `git show master:pkg/game/save_test.go` at this story's own base commit,
// unmodified. It is what a save this build (or an earlier build of it)
// actually wrote before the field was added, and its own test below decodes
// it rather than re-encoding and comparing bytes — the CURRENT encoder
// legitimately produces different bytes now, since PartyMember's gob type
// descriptor grew, and a byte-exact comparison against this fixture would
// fail for a reason that is not a defect.
const preWeaponMaterializedSaveFixtureBase64 = "QUdSTVNBVkUBIQByZWxlYXNlZCBlbnZlbG9wZSAxIHNpbXVsYXRpb24gNTPPINLe2hYAAP+EfwMBAQhTbmFwc2hvdAH/gAABCgEE" +
	"T3BlbgECAAEER29sZAEEAAEDV29uAf+CAAEJQXZhaWxhYmxlAf+CAAEFVGFrZW4B/4YAAQVQYXJ0eQH/nAABB09mZmVyZWQBBAAB" +
	"B01pc3Npb24BBAABBVdvcmxkAQoAAQdSZXNpZHVlAf+eAAAAE/+BAgEBBVtdaW50Af+CAAEEAAAj/4UCAQEUW11nYW1lLlNuYXBz" +
	"aG90T2ZmZXIB/4YAAf+EAAA+/4MDAQENU25hcHNob3RPZmZlcgH/hAABAwEHQ2hhcHRlcgEEAAEIQnVpbGRpbmcBBAABBUluZGV4" +
	"AQQAAAAk/5sCAQEVW11tYXBsb2FkLlBhcnR5TWVtYmVyAf+cAAH/iAAA/gEk/4cDAQELUGFydHlNZW1iZXIB/4gAARUBAklEAQwA" +
	"AQlUZW1wb3JhcnkBAgABBE5hbWUBDAABD1BsYXllckNoYXJhY3RlcgECAAEMU3RhcnRpbmdIZXJvAQIAAQ1NZXJjZW5hcnlUeXBl" +
	"AQYAAQxDb21wYW5pb25OUEMBBAABBUNsYXNzAQQAAQRCb2R5AQwAAQdCb2R5RGlyAQwAAQRNYWdlAQIAAQdQcm9maWxlAf+KAAEJ" +
	"RmlndXJlRGlyAQwAAQpGaWd1cmVGYWNlAQQAAQRIZXJvAf+MAAELS25vd25TcGVsbHMBBgABBFdvcm4B/5AAAQdDYXJyaWVkAf+S" +
	"AAEGV2VhcG9uAf+UAAEFQ2FycnkB/5YAAQVTYXZlZAH/mAAAAEH/iQMBAQdQcm9maWxlAf+KAAEDAQdGaWdodGVyAQIAAQxIZWFs" +
	"dGhDb2x1bW4BAgABCk1hbmFDb2x1bW4BAgAAAEf/iwMBAQRIZXJvAf+MAAEFAQRCb2R5AQQAAQhSZWFjdGlvbgEEAAEETWluZAEE" +
	"AAEGU3Bpcml0AQQAAQVTa2lsbAH/jgAAABj/jQEBAQhbNl1pbnQzMgH/jgABBAEMAAAa/48BAQEKWzEyXXVpbnQxNgH/kAABBgEY" +
	"AAAW/5ECAQEIW111aW50MTYB/5IAAQYAAP+3/5MDAQEGV2VhcG9uAf+UAAENAQROYW1lAQwAAQNSb3cBBAABBENvZGUBBgABCkRh" +
	"bWFnZUJhc2UBBAABDERhbWFnZVNwcmVhZAEEAAEFVG9IaXQBBAABB0RlZmVuY2UBBAABCkF0dGFja1R5cGUBBAABCkNoYXJnZVRp" +
	"bWUBBAABCVJlbGF4VGltZQEEAAEFUmFuZ2UBBAABCVNwZWxsTmFtZQEMAAEKU3BlbGxQb3dlcgEEAAAAOf+VAwEBBUNhcnJ5Af+W" +
	"AAEDAQdTa2lsbFhQAf+OAAEFSXRlbXMB/5IAAQhFcXVpcHBlZAH/kAAAAHv/lwMBAQVTYXZlZAH/mAABCAEEQ2VsbAH/mgABAkhQ" +
	"AQQAAQVNYXhIUAEEAAEETWFuYQEEAAEHTWF4TWFuYQEEAAERSGVhbHRoUmVnZW5QZXJpb2QBBAABD01hbmFSZWdlblBlcmlvZAEE" +
	"AAEJTWFwVW5pdElEAQYAAAAe/5kDAQEEQ2VsbAH/mgABAgEBWAEEAAEBWQEEAAAAd/+dAwEBD1NuYXBzaG90UmVzaWR1ZQH/ngAB" +
	"BwEJQ29tbWFuZGVkAf+gAAEFU3dpbmcB/6IAAQVQaGFzZQH/pAABCEdyb3VwVGFnAQYAAQdGb2dDb2xzAQQAAQdGb2dSb3dzAQQA" +
	"AQtGb2dFeHBsb3JlZAEKAAAAFv+fAgEBCFtddWludDMyAf+gAAEGAAAe/6EEAQEObWFwW3VpbnQzMl1pbnQB/6IAAQYBBAAAIP+j" +
	"BAEBEG1hcFt1aW50MzJddWludDgB/6QAAQYBBgAA/hFc/4ABAQH+AoIBARQBASgDKAEUAf4RQzUAAAAAAAAAAAkAAAAAAAAACAAA" +
	"AAgAAAAAAAAAAEAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAACAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAJAwAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAICAgICAgICAgICAgICAgICAgICAgICAgIC" +
	"AgICAgICAgICAgICAgICAgICAgICAgICAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA" +
	"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAEAAA=="

// TestPreWeaponMaterializedSaveFixtureStillDecodesItsOuterEnvelope is
// DIV-095's restored gob witness (round-2 adversarial review, twelfth pass,
// C3): a save this build (or an earlier build of it) actually wrote, byte
// for byte, BEFORE PartyMember.WeaponMaterialized existed. The outer
// Snapshot still decodes under the current reader. Its embedded simulation
// form is deliberately refused now: form 53 cannot say which actors suppress
// corpse loot, so accepting it as form 54 would invent hashed simulation
// state.
//
// This particular fixture's own Snapshot never populates Party (see
// releasedSaveFixtureSnapshot, master's and this branch's identical
// constructor for the payload these bytes encode), so the loop below runs
// over zero members today. Its own value is what it proves regardless of
// that count: gob's append-only evolution rule decodes a field the wire
// stream never wrote to its Go zero value, so WeaponMaterialized comes back
// false for a member from this era not because this fixture demonstrates it
// on a real member, but because the language guarantees it and the loop
// compiles and runs against the CURRENT mapload.PartyMember struct without
// error — the decode succeeding at all, over a type descriptor one field
// narrower than today's, is this test's own claim.
func TestPreWeaponMaterializedSaveFixtureStillDecodesItsOuterEnvelope(t *testing.T) {
	want, err := base64.StdEncoding.DecodeString(preWeaponMaterializedSaveFixtureBase64)
	if err != nil {
		t.Fatal(err)
	}
	s, label, err := DecodeSave(want)
	if err != nil {
		t.Fatalf("pre-WeaponMaterialized fixture no longer decodes: %v", err)
	}
	if label != "released envelope 1 simulation 53" || !s.Open || s.Gold != 321 ||
		s.Mission != 10 || s.Offered != 20 || !reflect.DeepEqual(s.Won, []int{10}) ||
		!reflect.DeepEqual(s.Available, []int{20}) {
		t.Fatalf("pre-WeaponMaterialized fixture decoded as label %q snapshot %+v", label, s)
	}
	var w sim.World
	if err := w.UnmarshalBinary(s.World); err == nil || !strings.Contains(err.Error(), "version 53") {
		t.Fatalf("pre-WeaponMaterialized fixture's simulation form error = %v, want an explicit version-53 refusal", err)
	}
	for i, member := range s.Party {
		if member.WeaponMaterialized {
			t.Errorf("member %d (%q) decoded WeaponMaterialized true from a save that could not have written it",
				i, member.Name)
		}
	}
}

// TestLoadingPutsTheTownScreenBackAtTheSquare: the open room belongs to the game
// being left, and a loaded game is entered at the square.
func TestLoadingPutsTheTownScreenBackAtTheSquare(t *testing.T) {
	f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Town: saveTown(t)}}
	ts := f.TownScreen().(*townScreen)
	ts.room, ts.said = roomTavern, 2
	if _, _, err := f.Restore(Snapshot{Open: true, Gold: 5}); err != nil {
		t.Fatalf("Restore: %v", err)
	}
	if ts.room != roomSquare || ts.said != 0 {
		t.Errorf("the town screen came back in room %v with %d lines said, want the square", ts.room, ts.said)
	}
	if got, _ := f.TownScreen().(*townScreen); got != ts {
		t.Error("TownScreen built a second screen; the flow holds the first for the life of the process")
	}
}
