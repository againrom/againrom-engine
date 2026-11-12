package game

import (
	"againrom/pkg/ui"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

// The sack pickup record: every driver of a sack pickup on synthetic fixtures,
// under each game's World profile. Each tick records the World digest and the
// digest of its byte form, each tracked actor's cell, order, state and pack,
// the purse, the ground sacks, the window's queued commands and the posted
// lines. The record is compared with testdata/sackpickup_record.txt;
// AGAINROM_WRITE_SACKPICKUP_RECORD=1 rewrites it.
func TestSackPickupRecord(t *testing.T) {
	var b strings.Builder
	for _, game := range []string{"first", "second"} {
		r := &sackRecord{t: t, b: &b, game: game}
		r.click()
		r.clickDocuments()
		r.clickRefusedDocument()
		r.clickRefusedOwner()
		r.clickNoSack()
		r.sackGone()
		r.laterOrder()
		r.load()
		r.twoPending()
		r.twoPendingBlocked()
		r.keyUnderfoot()
		r.keyPredecessors()
		r.keyMidAttack()
		r.keyCrossing()
		r.keyRefusedOwner()
		r.cheat()
		r.cheatNoHero()
		r.cheatRefused()
		r.scenario()
		r.scenarioRefused()
		r.missionRun()
		r.missionRefused()
	}
	compareSackRecord(t, filepath.Join("testdata", "sackpickup_record.txt"), b.String())
}

func compareSackRecord(t *testing.T, golden, got string) {
	t.Helper()
	if os.Getenv("AGAINROM_WRITE_SACKPICKUP_RECORD") == "1" {
		if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	if string(want) == got {
		return
	}
	w, g := strings.Split(string(want), "\n"), strings.Split(got, "\n")
	for i := 0; i < len(w) || i < len(g); i++ {
		var a, b string
		if i < len(w) {
			a = w[i]
		}
		if i < len(g) {
			b = g[i]
		}
		if a != b {
			t.Fatalf("sack pickup record differs at line %d:\nrecorded %q\nnow      %q", i+1, a, b)
		}
	}
}

const (
	sackRecordDocument = uint16(data.QuestDocumentCode)
	sackRecordBound    = 300
)

type sackRecord struct {
	t     *testing.T
	b     *strings.Builder
	game  string
	w     *sim.World
	ms    *Mission
	mw    *mapWorld
	ids   []sim.EntityID
	last  map[string]string
	ticks int
}

func sackRecordHero(id sim.EntityID, x, y int32) sim.Entity {
	return sim.Entity{ID: id, X: x, Y: y, Owner: sim.SelfSlot, HP: 20, MaxHP: 20, Speed: 16, Facing: 64, DesiredFacing: 64}
}

func (r *sackRecord) script() *sim.Script {
	if r.game == "first" {
		return nil
	}
	s, err := sim.NewROM2Script(nil, nil, nil)
	if err != nil {
		r.t.Fatal(err)
	}
	return s
}

// mission builds the World and its mission: party lists the party's entity
// ids and starting the index of the starting hero, -1 for none.
func (r *sackRecord) mission(name string, ents []sim.Entity, sacks []sim.Sack, party []sim.EntityID, starting int) {
	r.t.Helper()
	w, err := sim.NewLootWorld(1, sim.Bounds{Width: worldFixtureW, Height: worldFixtureH}, sim.ModeCanonical, sim.Terrain{},
		ents, r.script(), sim.Relations{}, sacks)
	if err != nil {
		r.t.Fatalf("%s %s: NewLootWorld: %v", r.game, name, err)
	}
	members := make([]mapload.PartyMember, len(party))
	for i := range members {
		members[i] = mapload.PartyMember{PlayerCharacter: true, StartingHero: i == starting}
	}
	r.w, r.mw = w, nil
	r.ms = &Mission{Number: 1, Map: worldFixtureMap(), World: w, Party: members, Start: mapload.Start{IDs: party}}
	r.ids = r.ids[:0]
	for _, e := range ents {
		r.ids = append(r.ids, e.ID)
	}
	r.last, r.ticks = map[string]string{}, 0
	fmt.Fprintf(r.b, "== %s %s\n", r.game, name)
}

// window opens the map window over the mission.
func (r *sackRecord) window() {
	r.t.Helper()
	r.mw = openMission(r.ms, nil, nil, worldFixtureViewer(r.t, r.ms.Map), missionSource{}, nil, nil)
	r.mw.invParty.table = eqDefsTable(r.t)
	r.snap("open")
}

func (r *sackRecord) open(name string, ents []sim.Entity, sacks []sim.Sack, party []sim.EntityID, starting int) {
	r.t.Helper()
	r.mission(name, ents, sacks, party, starting)
	r.window()
}

// line writes one keyed line when it differs from the key's previous line.
func (r *sackRecord) line(key, text string) {
	if r.last[key] == text {
		return
	}
	r.last[key] = text
	fmt.Fprintf(r.b, "  %s\n", text)
}

func (r *sackRecord) snap(label string) {
	r.t.Helper()
	form := "none"
	if raw, err := r.w.MarshalBinary(); err != nil {
		form = "error " + err.Error()
	} else {
		sum := sha256.Sum256(raw)
		form = fmt.Sprintf("%x", sum[:8])
	}
	fmt.Fprintf(r.b, "%s tick=%d hash=%016x form=%s\n", label, r.w.Tick(), r.w.Hash(), form)
	for _, id := range r.ids {
		key := fmt.Sprint("actor ", id)
		e, ok := r.w.Entity(id)
		if !ok {
			r.line(key, key+" absent")
			continue
		}
		stacks, _ := r.w.CarriedStacks(id)
		var pack []string
		for _, st := range stacks {
			pack = append(pack, fmt.Sprintf("%#x:%d/%d", st.Code, st.Count, st.ObjectID))
		}
		r.line(key, fmt.Sprintf("%s at=(%d,%d) transit=%d state=%#x pending=%+v target=%t(%d,%d) attack=%t/%d phase=%d post=(%d,%d) pack=[%s]",
			key, e.X, e.Y, e.Transit, e.ActorState, e.PendingOrder, e.HasTarget, e.TargetX, e.TargetY,
			e.HasAttackTarget, e.AttackTarget, e.AttackPhase, e.PostX, e.PostY, strings.Join(pack, " ")))
	}
	var sacks []string
	for _, s := range r.w.Sacks() {
		sacks = append(sacks, fmt.Sprintf("(%d,%d) gold=%d items=%x", s.X, s.Y, s.Gold, s.Items))
	}
	r.line("ground", fmt.Sprintf("purse=%d sacks=[%s]", r.w.Purse(sim.SelfSlot), strings.Join(sacks, "; ")))
	if r.mw == nil {
		return
	}
	var queue []string
	for _, c := range r.mw.pending {
		queue = append(queue, fmt.Sprintf("kind=%d entity=%d at=(%d,%d) group=%d", c.Kind, c.Entity, c.X, c.Y, c.Group))
	}
	r.line("queue", "queue=["+strings.Join(queue, "; ")+"]")
	var lines []string
	for _, l := range r.mw.view.MessageLines() {
		lines = append(lines, fmt.Sprintf("%q", l.Text))
	}
	r.line("lines", "lines=["+strings.Join(lines, " ")+"]")
}

// settled reports that no tracked actor holds a pickup, a completion or a walk.
func (r *sackRecord) settled() bool {
	for _, id := range r.ids {
		e, ok := r.w.Entity(id)
		if !ok {
			continue
		}
		if e.PendingOrder.Kind == sim.PendingPickup || e.PendingOrder.Kind == sim.PendingPickupComplete || e.ActorState == 2 || e.HasTarget || e.Transit != 0 {
			return false
		}
	}
	return true
}

// tick advances one window tick, or one bare Step without a window.
func (r *sackRecord) tick() {
	r.t.Helper()
	if r.mw != nil {
		r.mw.tick()
	} else {
		sim.Step(r.w, nil)
	}
	r.ticks++
	r.snap(fmt.Sprint("t", r.ticks))
}

// run ticks until the actors have settled and then three ticks more.
func (r *sackRecord) run() {
	r.t.Helper()
	for n := 0; n < sackRecordBound; n++ {
		r.tick()
		if r.settled() {
			for range 3 {
				r.tick()
			}
			return
		}
	}
	fmt.Fprintf(r.b, "bound %d spent\n", sackRecordBound)
}

func (r *sackRecord) click() {
	r.open("click", []sim.Entity{sackRecordHero(7, 3, 3)},
		[]sim.Sack{{X: 5, Y: 3, Gold: 500, Items: []uint16{eqSwordCode}}}, []sim.EntityID{7}, 0)
	r.mw.grab(7, 5, 3, ui.GrabAtCell)
	r.snap("grab")
	r.run()
}

func (r *sackRecord) clickDocuments() {
	r.open("click-documents", []sim.Entity{sackRecordHero(7, 3, 3), sackRecordHero(8, 3, 6)},
		[]sim.Sack{{X: 5, Y: 3, Gold: 20, Items: []uint16{eqSwordCode, sackRecordDocument}}}, []sim.EntityID{7, 8}, 1)
	r.mw.grab(7, 5, 3, ui.GrabAtCell)
	r.snap("grab")
	r.run()
}

func (r *sackRecord) clickRefusedDocument() {
	r.open("click-refused-document", []sim.Entity{sackRecordHero(7, 3, 3)},
		[]sim.Sack{{X: 5, Y: 3, Gold: 20, Items: []uint16{sackRecordDocument}}}, []sim.EntityID{7}, -1)
	r.mw.grab(7, 5, 3, ui.GrabAtCell)
	r.snap("grab")
	r.run()
	r.mw.grab(0, 0, 0, ui.GrabUnderfoot)
	r.snap("key")
}

func (r *sackRecord) clickRefusedOwner() {
	hero := sackRecordHero(7, 3, 3)
	hero.Owner = 50
	r.open("click-refused-owner", []sim.Entity{hero},
		[]sim.Sack{{X: 5, Y: 3, Gold: 20, Items: []uint16{eqSwordCode}}}, []sim.EntityID{7}, 0)
	r.mw.grab(7, 5, 3, ui.GrabAtCell)
	r.snap("grab")
	r.run()
}

func (r *sackRecord) clickNoSack() {
	r.open("click-no-sack", []sim.Entity{sackRecordHero(7, 3, 3)},
		[]sim.Sack{{X: 5, Y: 3, Gold: 20}}, []sim.EntityID{7}, 0)
	r.mw.grab(7, 20, 20, ui.GrabAtCell)
	r.snap("grab")
	r.mw.grab(7, 5, 3, ui.GrabAtCell)
	r.snap("grab-gold")
	r.tick()
}

func (r *sackRecord) sackGone() {
	r.open("sack-gone", []sim.Entity{sackRecordHero(7, 3, 3), sackRecordHero(9, 5, 3)},
		[]sim.Sack{{X: 5, Y: 3, Gold: 20, Items: []uint16{eqSwordCode}}}, []sim.EntityID{7}, 0)
	r.mw.grab(7, 5, 3, ui.GrabAtCell)
	r.snap("grab")
	r.tick()
	if err := r.w.TakeSack(9, 5, 3); err != nil {
		r.t.Fatalf("%s sack-gone: %v", r.game, err)
	}
	r.mw.enqueue(9, 5, 9)
	r.snap("gone")
	r.run()
}

func (r *sackRecord) laterOrder() {
	r.open("later-order", []sim.Entity{sackRecordHero(7, 3, 3)},
		[]sim.Sack{{X: 5, Y: 3, Gold: 20, Items: []uint16{eqSwordCode}}}, []sim.EntityID{7}, 0)
	r.mw.grab(7, 5, 3, ui.GrabAtCell)
	r.snap("grab")
	r.tick()
	r.tick()
	r.mw.enqueue(7, 5, 3)
	r.snap("move")
	r.run()
}

func (r *sackRecord) load() {
	r.open("load", []sim.Entity{sackRecordHero(7, 3, 3)},
		[]sim.Sack{{X: 5, Y: 3, Gold: 20, Items: []uint16{eqSwordCode}}}, []sim.EntityID{7}, 0)
	r.mw.grab(7, 5, 3, ui.GrabAtCell)
	r.snap("grab")
	for range 3 {
		r.tick()
	}
	raw, err := r.w.MarshalBinary()
	if err != nil {
		r.t.Fatalf("%s load: %v", r.game, err)
	}
	restored := new(sim.World)
	if err := restored.UnmarshalBinary(raw); err != nil {
		r.t.Fatalf("%s load: %v", r.game, err)
	}
	r.w, r.ms.World = restored, restored
	r.window()
	r.run()
}

func (r *sackRecord) twoPending() {
	r.open("two-pending", []sim.Entity{sackRecordHero(7, 3, 3), sackRecordHero(8, 3, 5)},
		[]sim.Sack{{X: 5, Y: 3, Items: []uint16{eqSwordCode}}, {X: 5, Y: 5, Gold: 30}}, []sim.EntityID{7, 8}, 0)
	r.mw.grab(7, 5, 3, ui.GrabAtCell)
	r.mw.grab(8, 5, 5, ui.GrabAtCell)
	r.snap("grab")
	r.run()
}

func (r *sackRecord) twoPendingBlocked() {
	r.open("two-pending-blocked", []sim.Entity{sackRecordHero(7, 3, 3), sackRecordHero(8, 3, 5)},
		[]sim.Sack{{X: 7, Y: 3, Items: []uint16{eqSwordCode}}, {X: 4, Y: 5, Gold: 30}}, []sim.EntityID{7, 8}, 0)
	r.mw.grab(7, 7, 3, ui.GrabAtCell)
	r.mw.grab(8, 4, 5, ui.GrabAtCell)
	r.snap("grab")
	r.run()
}

func (r *sackRecord) keyUnderfoot() {
	hero := sackRecordHero(7, 5, 3)
	hero.HasTarget, hero.TargetX, hero.TargetY = true, 20, 20
	r.open("key-underfoot", []sim.Entity{hero},
		[]sim.Sack{{X: 5, Y: 3, Gold: 17, Items: []uint16{eqSwordCode}}}, []sim.EntityID{7}, 0)
	r.mw.grab(0, 0, 0, ui.GrabUnderfoot)
	r.snap("key")
	r.run()
}

func (r *sackRecord) keyPredecessors() {
	r.open("key-predecessors", []sim.Entity{sackRecordHero(7, 3, 3), sackRecordHero(99, 20, 20)},
		[]sim.Sack{{X: 5, Y: 3, Items: []uint16{eqSwordCode}}, {X: 3, Y: 3, Gold: 17}}, []sim.EntityID{7}, 0)
	r.mw.grab(7, 5, 3, ui.GrabAtCell)
	r.mw.pending = append(r.mw.pending, sim.Equip(7, 999, 1), sim.MoveTo(99, sim.CellPoint{X: 21, Y: 20}))
	r.snap("queued")
	r.mw.grab(0, 0, 0, ui.GrabUnderfoot)
	r.snap("key")
	r.run()
}

func (r *sackRecord) keyMidAttack() {
	hero := sackRecordHero(7, 5, 3)
	hero.HP, hero.MaxHP = 100, 100
	hero.AttackCharge, hero.AttackTarget, hero.HasAttackTarget, hero.AttackPhase, hero.AttackCountdown = 10, 2, true, sim.AttackCharging, 5
	victim := sim.Entity{ID: 2, X: 6, Y: 3, Owner: 1, HP: 100, MaxHP: 100}
	r.open("key-mid-attack", []sim.Entity{victim, hero},
		[]sim.Sack{{X: 5, Y: 3, Gold: 17, Items: []uint16{eqSwordCode}}}, []sim.EntityID{7}, 0)
	r.mw.grab(0, 0, 0, ui.GrabUnderfoot)
	r.snap("key")
	r.run()
}

func (r *sackRecord) keyCrossing() {
	r.open("key-crossing", []sim.Entity{sackRecordHero(7, 3, 3)},
		[]sim.Sack{{X: 4, Y: 3, Gold: 500, Items: []uint16{eqSwordCode}}}, []sim.EntityID{7}, 0)
	r.mw.enqueue(7, 4, 3)
	for n := 0; n < 40 && len(r.w.Sacks()) != 0; n++ {
		r.tick()
		r.mw.grab(0, 0, 0, ui.GrabUnderfoot)
		r.snap("key")
	}
	r.run()
}

func (r *sackRecord) keyRefusedOwner() {
	hero := sackRecordHero(7, 5, 3)
	hero.Owner = 50
	r.open("key-refused-owner", []sim.Entity{hero},
		[]sim.Sack{{X: 5, Y: 3, Gold: 17, Items: []uint16{eqSwordCode}}}, []sim.EntityID{7}, 0)
	r.mw.grab(0, 0, 0, ui.GrabUnderfoot)
	r.snap("key")
	r.run()
}

func (r *sackRecord) cheatSacks() []sim.Sack {
	return []sim.Sack{{X: 5, Y: 3, Gold: 5, Items: []uint16{eqSwordCode}}, {X: 10, Y: 10, Items: []uint16{sackRecordDocument}}, {X: 12, Y: 4, Gold: 9}}
}

func (r *sackRecord) cheat() {
	r.open("cheat", []sim.Entity{sackRecordHero(7, 3, 3), sackRecordHero(8, 3, 6)}, r.cheatSacks(), []sim.EntityID{7, 8}, 1)
	err := r.mw.cheatPickupAll(7)
	fmt.Fprintf(r.b, "cheat error=%v\n", err)
	r.snap("cheat")
	r.run()
}

func (r *sackRecord) cheatNoHero() {
	r.open("cheat-no-hero", []sim.Entity{sackRecordHero(7, 3, 3)}, r.cheatSacks(), []sim.EntityID{7}, -1)
	err := r.mw.cheatPickupAll(7)
	fmt.Fprintf(r.b, "cheat error=%v\n", err)
	r.snap("cheat")
	r.run()
}

func (r *sackRecord) cheatRefused() {
	hero := sackRecordHero(7, 3, 3)
	hero.Owner = 50
	r.open("cheat-refused", []sim.Entity{hero}, r.cheatSacks(), []sim.EntityID{7}, 0)
	err := r.mw.cheatPickupAll(7)
	fmt.Fprintf(r.b, "cheat error=%v\n", err != nil)
	r.snap("cheat")
}

func (r *sackRecord) scenario() {
	r.mission("scenario", []sim.Entity{sackRecordHero(7, 3, 3)},
		[]sim.Sack{{X: 5, Y: 3, Gold: 20, Items: []uint16{eqSwordCode}}}, []sim.EntityID{7}, 0)
	err := NewPlayWorld(r.ms).pickUp(7, 5, 3)
	fmt.Fprintf(r.b, "pick_item error=%v\n", err)
	r.snap("pick_item")
	r.run()
}

func (r *sackRecord) scenarioRefused() {
	r.mission("scenario-refused", []sim.Entity{sackRecordHero(7, 3, 3)},
		[]sim.Sack{{X: 5, Y: 3, Gold: 20, Items: []uint16{sackRecordDocument}}}, []sim.EntityID{7}, -1)
	err := NewPlayWorld(r.ms).pickUp(7, 5, 3)
	fmt.Fprintf(r.b, "pick_item error=%v\n", err)
	r.snap("pick_item")
	r.run()
}

func (r *sackRecord) missionWalk(name string, items []uint16, starting int) {
	r.mission(name, []sim.Entity{sackRecordHero(7, 3, 3)},
		[]sim.Sack{{X: 4, Y: 3, Gold: 20, Items: items}}, []sim.EntityID{7}, starting)
	sim.Step(r.w, []sim.Command{sim.MoveTo(7, sim.CellPoint{X: 4, Y: 3})})
	r.snap("move")
	for n := 0; n < 40; n++ {
		sack, ok := r.ms.PickUpUnderfoot(7)
		fmt.Fprintf(r.b, "underfoot taken=%t sack=(%d,%d) gold=%d items=%x\n", ok, sack.X, sack.Y, sack.Gold, sack.Items)
		r.snap("underfoot")
		if ok {
			break
		}
		r.tick()
	}
	r.run()
}

func (r *sackRecord) missionRun() {
	r.missionWalk("mission", []uint16{eqSwordCode, sackRecordDocument}, 0)
}

func (r *sackRecord) missionRefused() {
	r.missionWalk("mission-refused", []uint16{sackRecordDocument}, -1)
}
