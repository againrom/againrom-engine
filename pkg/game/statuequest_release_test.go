package game

import (
	"crypto/sha256"
	"fmt"
	"os"
	"testing"

	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func statueQuestActors(t *testing.T, f *FrontEnd) (sim.EntityID, sim.EntityID, sim.EntityID) {
	t.Helper()
	hero := f.live.mission.ids[0]
	var rood sim.EntityID
	found := false
	for i, member := range f.live.mission.party {
		if member.CompanionNPC == 26 {
			rood, found = f.live.mission.ids[i], true
		}
	}
	if !found {
		t.Fatal("mission has no Rood party role")
	}
	for _, e := range f.live.world.Entities() {
		if e.MapUnitID == 205 {
			return hero, rood, e.ID
		}
	}
	t.Fatal("mission has no authored Lord")
	return 0, 0, 0
}

func statueQuestProgress(t *testing.T, f *FrontEnd, count int32) {
	t.Helper()
	w := f.live.world
	if w.ScriptRegister(60) != count || w.Outcome() != sim.OutcomeUndecided {
		t.Fatalf("transferred=%d want=%d outcome=%d", w.ScriptRegister(60), count, w.Outcome())
	}
	for i := int32(0); i < 5; i++ {
		if w.ScriptLatched(5+i) != (i < count) {
			t.Fatalf("delivery latch %d=%v for %d parts", 5+i, w.ScriptLatched(5+i), count)
		}
	}
}

func statueQuestEvent(t *testing.T, f *FrontEnd, app *ui.App, event int) {
	t.Helper()
	data, ok := ReadEventText(f.Archives.Containers, 151, event)
	if !ok {
		t.Fatal("missing installed quest event", event)
	}
	want, ok := expectedInstalledDialoguePart(t, data, 1, HeroAudience(f.LiveParty()))
	if !ok || want == "" {
		t.Fatal("missing installed quest event part", event)
	}
	text, kind, open := f.LiveNotice()
	if !open || kind != ui.NoticeDialogue || text != want {
		t.Fatalf("event %d not visible: open=%t kind=%d text=%q want=%q", event, open, kind, text, want)
	}
	quest81Dismiss(t, f, app)
}

func statueQuestFinish(t *testing.T, f *FrontEnd, app *ui.App, rood, lord sim.EntityID, menu bool) []byte {
	t.Helper()
	w := f.live.world
	statueQuestProgress(t, f, 5)
	if !w.ScriptLatched(10) {
		t.Fatal("five parts did not issue Uncurse Lord")
	}
	statueQuestEvent(t, f, app, 3)
	for code := uint16(0x0e2d); code <= 0x0e31; code++ {
		if quest81Carries(w, rood, code) {
			t.Fatalf("delivered part %04x remains in Rood's inventory", code)
		}
	}
	remaining := 0
	for _, e := range w.ActiveEffects() {
		if e.Target == lord && e.Spell == 20 {
			remaining = int(e.Remaining)
		}
	}
	if remaining == 0 || remaining > 9600 {
		t.Fatalf("quest lacks a finite Stone Curse: %d", remaining)
	}
	for tick := 0; tick < remaining+512 && !w.ScriptLatched(22); tick++ {
		f.LiveAdvance(1)
	}
	e, ok := w.Entity(lord)
	if !ok || w.HasEffectSpell(lord, 20) || e.X != 130 || e.Y != 12 || !w.ScriptLatched(22) {
		t.Fatalf("Lord did not thaw and reach sword handoff: stone=%t actor=%+v message=%t", w.HasEffectSpell(lord, 20), e, w.ScriptLatched(22))
	}
	statueQuestEvent(t, f, app, 9)
	sack := groundAt(w.Sacks(), 130, 13)
	if sack == nil || len(sack.Items) != 1 || sack.Items[0] != 0x7166 || quest81Carries(w, rood, 0x7166) {
		t.Fatalf("pre-existing sword changed before pickup: %+v", sack)
	}
	livePlaceAndWalk(t, f.live, rood, 130, 13)
	app.Layout(1024, 768)
	f.live.view.Camera().CenterOn(130*32, 13*32)
	quest81OK(t, app.HeadlessSelectEntity(uint32(rood)))
	quest81OK(t, app.HeadlessKey("grab"))
	items, _ := w.Carried(rood)
	swords := 0
	for _, item := range items {
		if item == 0x7166 {
			swords++
		}
	}
	if swords != 1 || groundAt(w.Sacks(), 130, 13) != nil {
		t.Fatalf("sword pickup count=%d; sack remains=%t", swords, groundAt(w.Sacks(), 130, 13) != nil)
	}
	var store SaveStore
	var raw []byte
	if menu {
		store, _, raw = menuSAVE(t, f, app, OriginalStore{})
	} else {
		store, _, raw = campaignSave(t, f, true)
	}
	cold, _ := campaignCold(t, store)
	quest81Paired(t, f, cold, 32)
	if !quest81Carries(cold.live.world, rood, 0x7166) || cold.live.world.ScriptRegister(60) != 5 {
		t.Fatal("cold SAV lost the sword or completed delivery")
	}
	t.Logf("five deliveries, events03/09, natural thaw in <=%d ticks, Lord130,12, one existing sword, SAV/cold LOAD/32 next ticks", remaining+512)
	return raw
}

func TestReleaseStatueQuestRoleAndColdSAV(t *testing.T) {
	f := releaseFront(t)
	f.SetDeterministicFrames(true)
	party := f.NextParty()
	rood, ok := mapload.CampaignNPCMember(f.Table, 26, 140, party)
	if !ok {
		t.Fatal("installed NPC26 unavailable")
	}
	party = append(party, rood)
	prepareAcceptedCampaignMission(t, f, 151)
	app := f.App("Statue quest")
	app.Layout(1024, 768)
	quest81OK(t, app.OpenMission(f.MissionOpenerWith(151, party)))
	hero, carrier, lord := statueQuestActors(t, f)
	w := f.live.world
	for _, at := range [][2]int32{{59, 77}, {8, 101}, {133, 135}, {80, 8}, {99, 9}} {
		quest81OK(t, w.TakeSack(hero, at[0], at[1]))
	}
	livePlaceAndWalk(t, f.live, hero, 129, 14)
	livePlaceAndWalk(t, f.live, carrier, 131, 14)
	f.LiveAdvance(32)
	statueQuestProgress(t, f, 0)
	if w.ScriptRegister(10) != 1 || !w.HasEffectSpell(lord, 20) {
		t.Fatal("wrong-holder control did not evaluate Rood adjacent to the statue")
	}
	quest81Dismiss(t, f, app)
	livePlaceAndWalk(t, f.live, carrier, 130, 15)
	for code := uint16(0x0e2d); code < 0x0e31; code++ {
		quest81OK(t, w.MoveCarried(hero, carrier, code, 1))
	}
	f.LiveAdvance(32)
	statueQuestProgress(t, f, 0)
	if w.ScriptRegister(10) != 2 {
		t.Fatal("distance-two control did not retain its boundary")
	}
	store, _, _ := campaignSave(t, f, true)
	cold, _ := campaignCold(t, store)
	quest81Paired(t, f, cold, 16)
	livePlaceAndWalk(t, f.live, carrier, 130, 14)
	f.LiveAdvance(32)
	statueQuestProgress(t, f, 4)
	store, _, _ = campaignSave(t, f, true)
	cold, coldApp := campaignCold(t, store)
	quest81Paired(t, f, cold, 16)
	f, app, w = cold, coldApp, cold.live.world
	quest81OK(t, w.MoveCarried(hero, carrier, 0x0e31, 1))
	f.LiveAdvance(32)
	statueQuestFinish(t, f, app, carrier, lord, false)
}

func TestReleaseStatueQuestOwnerReadySAV(t *testing.T) {
	path := os.Getenv("AGAINROM_SEYDEN_SAV")
	if path == "" {
		t.Skip("set AGAINROM_SEYDEN_SAV to the owner ready-to-deliver mission151 SAV")
	}
	raw, err := os.ReadFile(path)
	quest81OK(t, err)
	const sourceSHA = "a1a1f4a154c7f63c393949e5aba615c9a68782aa1592dde7a70832ddf1baf793"
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != sourceSHA {
		t.Fatal("owner ready-to-deliver SAV SHA differs")
	}
	f := releaseFront(t)
	app, _ := openOriginalSAVApp(t, f, raw, "statue-ready.sav")
	_, rood, lord := statueQuestActors(t, f)
	w := f.live.world
	statueQuestProgress(t, f, 0)
	e, _ := w.Entity(rood)
	if e.X != 130 || e.Y != 14 {
		t.Fatal("owner Rood is no longer adjacent", e.X, e.Y)
	}
	for code := uint16(0x0e2d); code <= 0x0e31; code++ {
		if !quest81Carries(w, rood, code) {
			t.Fatal("owner Rood lacks a required part", code)
		}
	}
	f.LiveAdvance(32)
	written := statueQuestFinish(t, f, app, rood, lord, true)
	if path := os.Getenv("AGAINROM_SEYDEN_OUTPUT"); path != "" {
		out, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
		quest81OK(t, err)
		_, err = out.Write(written)
		closeErr := out.Close()
		quest81OK(t, err)
		quest81OK(t, closeErr)
	}
	t.Logf("source=%s; ordinary LOAD repaired the pending authored role without inventory or position edits", sourceSHA)
}
