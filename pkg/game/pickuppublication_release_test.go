package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/formats/sav"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

var pickupPublicationTokenFields = []string{"RuntimeID", "T0C", "T0E", "T08", "T18", "T1C", "Identity", "Reference"}

func TestReleaseMissionLoadShowsPendingPickupItems(t *testing.T) {
	path := os.Getenv("AGAINROM_SAVER_SAV")
	if path == "" {
		t.Skip("AGAINROM_SAVER_SAV is not set: mission LOAD pickup witness requires the owner's SAV")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "43f059f26b05c4fb888bf37be6f063918d6c0f536e1089512fab0e971bed82ac" {
		t.Fatalf("owner SAV changed: %s", got)
	}
	flagged := 0
	for _, head := range pickupPublicationItems(t, raw) {
		if head["T08"] != 0 {
			flagged++
		}
	}
	if flagged != 9 {
		t.Fatalf("owner SAV has %d flagged Items, want 9", flagged)
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "saver.sav")
	rows := f.live.view.MessageLines()
	if len(rows) == 0 {
		t.Fatal("direct mission LOAD showed no pending pickup items")
	}
	if len(rows) != flagged {
		t.Fatalf("direct mission LOAD showed %d lines, source has %d flagged Item records: %+v", len(rows), flagged, rows)
	}
	for _, row := range rows {
		if _, ok := pickupLineCount(&f.Words, row.Text); !ok || row.Ink != ui.MessageWhite || row.Life != 3*time.Second {
			t.Fatalf("session entry line %+v is not one Item record's pickup line, white for 3000 ms", row)
		}
	}
	t.Logf("direct mission LOAD published %d flagged Item records as separate lines: %+v", len(rows), rows)
	t.Run("message lifetime", func(t *testing.T) {
		watch := releaseFront(t)
		watchApp, _ := openOriginalSAVApp(t, watch, raw, "saver.sav")
		if got := watch.live.view.MessageLines(); !reflect.DeepEqual(got, rows) {
			t.Fatalf("fresh LOAD lines %+v, want %+v", got, rows)
		}
		// Lifetimes run one after another (MISSION-MSGLINE-057): the oldest
		// line leaves on the first tick whose counted time exceeds 3000 ms,
		// and the next line is counted from zero. Headless ticks are 20 ms
		// apart, so a line leaves 151 ticks after the one before it. The
		// list's clock starts at its first tick, which counts nothing, so the
		// first line leaves at frame 152.
		var leftAt []int
		for step := 1; step <= 9*151+1 && len(leftAt) < len(rows); step++ {
			if err := watchApp.HeadlessStep(); err != nil {
				t.Fatalf("map frame %d: %v", step, err)
			}
			got := watch.live.view.MessageLines()
			if len(got) != len(rows)-len(leftAt) {
				leftAt = append(leftAt, step)
				if want := rows[len(leftAt):]; len(got) != len(want) || len(want) > 0 && !reflect.DeepEqual(got, want) {
					t.Fatalf("after line %d left at frame %d the line holds %+v, want the %d newer lines %+v", len(leftAt), step, got, len(want), want)
				}
			}
		}
		if len(leftAt) != len(rows) || leftAt[0] != 152 {
			t.Fatalf("%d of %d LOAD lines left, at frames %v, want all, the first at 152", len(leftAt), len(rows), leftAt)
		}
		for i := 1; i < len(leftAt); i++ {
			if gap := leftAt[i] - leftAt[i-1]; gap != 151 {
				t.Fatalf("LOAD line %d left %d frames after line %d (%v), want 151", i+1, gap, i, leftAt)
			}
		}
		t.Logf("the %d LOAD lines left oldest first at frames %v", len(leftAt), leftAt)
	})
	_, _, saved := menuSAVE(t, f, app, OriginalStore{})
	for _, head := range pickupPublicationItems(t, saved) {
		if head["T08"] != 0 {
			t.Fatalf("re-saved Item %#x kept pickup flag %d", head["Identity"], head["T08"])
		}
	}
	cold := releaseFront(t)
	openOriginalSAVApp(t, cold, saved, "resaved.sav")
	if got := cold.live.view.MessageLines(); len(got) != 0 {
		t.Fatalf("SAVE then LOAD repeated pickup lines: %+v", got)
	}
	_, originalResave := groundCorpusFile(t, "2026-09-27/oldsaves7/game0006.sav", "b652cb4c5745b6dfa1a4cec12ea3be1dbb5991f44bbde97716755d6a5b8197ba")
	fromOriginal := releaseFront(t)
	openOriginalSAVApp(t, fromOriginal, originalResave, "game0006.sav")
	if got := fromOriginal.live.view.MessageLines(); len(got) != 0 {
		t.Fatalf("original re-save repeated pickup lines: %+v", got)
	}
}

func TestReleaseTownLoadedItemsStayQuietOnMissionEntry(t *testing.T) {
	path := os.Getenv("AGAINROM_CITY_SAV")
	if path == "" {
		t.Skip("AGAINROM_CITY_SAV is not set: town LOAD pickup witness requires the owner's SAV")
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256(raw)); got != "b8e25c3be3262c0cc715e74671a7475dfd52f843560e49c7ad14f035183c5541" {
		t.Fatalf("owner town SAV changed: %s", got)
	}
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	app := f.App("town pickup")
	app.Layout(1024, 768)
	name := filepath.Base(path)
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), raw, 0o600); err != nil {
		t.Fatal(err)
	}
	save, list, load := f.SaveSeams(SaveStore{Dir: t.TempDir()}, OriginalStore{Dir: dir}, func() time.Time { return time.Unix(1, 0) })
	app.SetSaveSeams(save, list, load)
	label := ""
	for _, entry := range list() {
		if entry.Name == name || entry.Name == localOriginalSaveToken(name) {
			label = entry.Label
		}
	}
	if label == "" {
		t.Fatal("owner town save is absent from the LOAD list")
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(label); err != nil || app.Screen() != ui.ScreenTown {
		t.Fatalf("town LOAD: %s: %v", app.Screen(), err)
	}
	if err := app.OpenMission(f.MissionOpener(f.Town.Chapter())); err != nil {
		t.Fatal(err)
	}
	if got := f.live.view.MessageLines(); len(got) != 0 {
		t.Fatalf("town LOAD then mission entry posted pickup lines: %+v", got)
	}
}

// pickupLineCount is the count a pickup line states: one for string 85, a
// space and a name, and N for a line that ends in ` (`, string 86, N and
// string 87, `)`. It answers false for text of neither form.
func pickupLineCount(words *ui.Words, line string) (uint32, bool) {
	rest, ok := strings.CutPrefix(line, words.PickedUp+" ")
	if !ok || rest == "" {
		return 0, false
	}
	open, closing := " ("+words.PickedUpNow+" ", " "+words.PickedUpPieces+")"
	if !strings.HasSuffix(rest, closing) {
		return 1, true
	}
	i := strings.LastIndex(rest, open)
	if i < 0 {
		return 0, false
	}
	n, err := strconv.ParseUint(rest[i+len(open):len(rest)-len(closing)], 10, 32)
	if err != nil || n < 2 {
		return 0, false
	}
	return uint32(n), true
}

// pickupPublicationItems maps each Item record's Identity to its Token head.
func pickupPublicationItems(t *testing.T, raw []byte) map[uint32]map[string]uint32 {
	t.Helper()
	doc, err := sav.DecodeDocumentData(raw)
	if err != nil {
		t.Fatal(err)
	}
	out := map[uint32]map[string]uint32{}
	for i := range doc.Objects {
		r := &doc.Objects[i]
		if !savedItemClass(r.Class) {
			continue
		}
		head := map[string]uint32{}
		for _, name := range pickupPublicationTokenFields {
			v, err := savedStructureValue(r, name)
			if err != nil {
				t.Fatal(r.Class, name, err)
			}
			head[name] = v
		}
		if _, dup := out[head["Identity"]]; dup {
			t.Fatal("Item Identity is not unique", head["Identity"])
		}
		out[head["Identity"]] = head
	}
	return out
}

// pickupPublicationHero is the source's player Human and the Identities of
// the Items it carries and wears.
func pickupPublicationHero(t *testing.T, f *FrontEnd) (sim.EntityID, map[uint32]bool) {
	t.Helper()
	var hero sim.EntityID
	held := map[uint32]bool{}
	r := f.live.world.SavedObjects()
	for _, row := range r.Items {
		for _, at := range r.Locations(row.ID) {
			if at.Owner.Kind != sim.SavedOwnerActorPack && at.Owner.Kind != sim.SavedOwnerActorWorn {
				continue
			}
			if at.Owner.Entity == f.live.mission.ids[0] {
				hero = at.Owner.Entity
				held[row.Token.Identity] = true
			}
		}
	}
	e, ok := f.live.entity(hero)
	if !ok || hero == 0 || e.Owner != sim.SelfSlot || !sim.InPersistBand(e.TypeID) || len(held) == 0 {
		t.Fatal("source player Human with saved Items is absent", hero, e.Owner, e.TypeID, len(held))
	}
	return hero, held
}

// A product pickup is a publication: the SAVE after it writes Token +0x08 = 0
// for every picked Item (SAV-1114). A LOAD of flagged Items publishes the
// player Human's carried and worn Items at session entry and posts rows for
// flagged carried Items (SAV-POSTLOAD-222); a ground Sack's Items keep their flag.
func TestReleasePickupPublicationClearsSavedPickupFlag(t *testing.T) {
	_, raw := groundCorpusFile(t, "2026-08-15/game0016.sav", "5e67d1282398076498867ac0124046d5c0e7f1acd2d0ffc1e66888c5296e0345")
	source := pickupPublicationItems(t, raw)
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "pickup.sav")
	hero, held := pickupPublicationHero(t, f)
	wantAnnouncement := map[string]uint32{}
	reg := f.live.world.SavedObjects()
	for _, row := range reg.Items {
		for _, at := range reg.Locations(row.ID) {
			if at.Owner.Kind == sim.SavedOwnerActorPack && at.Owner.Entity == hero {
				line := f.Words.PickedUp + " " + itemName(data.ItemCode(row.Value.Code), f.Table)
				if row.Value.Count > 1 {
					line += fmt.Sprintf(" (%s %d %s)", f.Words.PickedUpNow, row.Value.Count, f.Words.PickedUpPieces)
				}
				wantAnnouncement[line]++
			}
		}
	}
	sack := groundAt(f.live.world.Sacks(), 42, 39)
	if sack == nil || len(sack.ItemInstances) != 2 {
		t.Fatal("natural item Sack control missing")
	}
	sackItems := map[uint32]bool{}
	for _, item := range sack.ItemInstances {
		row, ok := f.live.world.SavedObjects().Item(item.ObjectID)
		if !ok {
			t.Fatal("source Sack Item has no identity")
		}
		sackItems[row.Token.Identity] = true
	}
	if len(f.live.view.MessageLines()) != 0 {
		t.Fatal("message line is not empty before the pickup")
	}
	// The headless teleport stands the hero on the Sack; the underfoot key is
	// the product pickup route.
	livePlaceAndWalk(t, f.live, hero, 42, 39)
	carried, _ := f.live.world.CarriedStacks(hero)
	f.live.takeSackFor(hero)
	if groundAt(f.live.world.Sacks(), 42, 39) != nil {
		t.Fatal("production pickup did not consume the Sack")
	}
	now, _ := f.live.world.CarriedStacks(hero)
	words := f.live.view.Words()
	wantLines := announced(pickupLinesForTake(reachedStacks(carried, now), int32(sack.Gold), f.live.invParty.table, &words)...)
	if got := f.live.view.MessageLines(); len(wantLines) == 0 || !reflect.DeepEqual(got, wantLines) {
		t.Fatalf("pickup lines %+v, want %+v", got, wantLines)
	}
	_, _, saved := menuSAVE(t, f, app, OriginalStore{})
	after := pickupPublicationItems(t, saved)
	for identity := range sackItems {
		if head, ok := after[identity]; !ok || head["T08"] != 0 {
			t.Fatalf("picked Item %#x saved %+v, want T08 = 0", identity, head)
		}
	}

	flagged := alterSAV(t, raw, func(doc *sav.DocumentData) bool {
		n := 0
		for i := range doc.Objects {
			r := &doc.Objects[i]
			if identity, err := savedStructureValue(r, "Identity"); err == nil && savedItemClass(r.Class) && (held[identity] || sackItems[identity]) {
				n++
				savedObjectSetValue(r, "T08", 1)
			}
		}
		return n == len(held)+len(sackItems)
	})
	cold := releaseFront(t)
	coldApp, _ := openOriginalSAVApp(t, cold, flagged, "flagged.sav")
	gotAnnouncement := map[string]uint32{}
	for _, row := range cold.live.view.MessageLines() {
		if row.Ink != ui.MessageWhite || row.Life != 3*time.Second {
			t.Fatalf("session entry line %+v is not white for 3000 ms", row)
		}
		gotAnnouncement[row.Text]++
	}
	if !reflect.DeepEqual(gotAnnouncement, wantAnnouncement) {
		t.Fatalf("session entry pickup lines %+v, want carried Items %+v", gotAnnouncement, wantAnnouncement)
	}
	_, _, resaved := menuSAVE(t, cold, coldApp, OriginalStore{})
	out := pickupPublicationItems(t, resaved)
	if len(out) != len(source) {
		t.Fatalf("Item population %d, source %d", len(out), len(source))
	}
	for identity, head := range source {
		want := head
		if sackItems[identity] {
			want = map[string]uint32{}
			for k, v := range head {
				want[k] = v
			}
			want["T08"] = 1
		}
		if !reflect.DeepEqual(out[identity], want) {
			t.Fatalf("Item %#x (held %v, ground %v) saved %+v, want %+v", identity, held[identity], sackItems[identity], out[identity], want)
		}
	}
	t.Logf("picked %d; flagged held %d cleared, ground %d kept; controls %d", len(sackItems), len(held), len(sackItems), len(source)-len(held)-len(sackItems))
}
