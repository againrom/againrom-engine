package game

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func installedEscort(t *testing.T) (*FrontEnd, *ui.App, sim.EntityID, sim.EntityID, int32, int32) {
	t.Helper()
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	app := f.App("escort continuation")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpener(100)); err != nil {
		t.Fatal(err)
	}
	for k := 0; k < 16 && f.live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	releasePauseMission(t, f, app)
	hero := f.live.mission.ids[0]
	var follower sim.EntityID
	found := false
	for k := 0; k < 64 && !found; k++ {
		for _, e := range f.live.world.Entities() {
			if e.Group == 21 && e.ActorState == 0x11 && e.HasEscortTarget && e.EscortTarget == hero && e.Alive() && !e.OffMap {
				follower, found = e.ID, true
				break
			}
		}
		if !found {
			f.live.tick()
		}
	}
	if !found {
		t.Fatal("mission 100 did not issue its installed group 21 Follow order")
	}
	m := f.live.mission.state.Map
	block := mapload.PassabilityWith(m, f.Table)
	for _, size := range []int{13, 11, 9} {
		x, y, ok := openGroundSquare(block, int(m.Width), int(m.Height), size)
		if ok {
			return f, app, follower, hero, int32(x + size/2), int32(y + size/2)
		}
	}
	t.Fatal("mission 100 has no open square for the escort placement")
	return nil, nil, 0, 0, 0, 0
}

func escortPlace(t *testing.T, f *FrontEnd, id sim.EntityID, x, y int32) sim.Entity {
	t.Helper()
	if err := f.live.world.HeadlessPlace(id, x, y); err != nil {
		t.Fatal(err)
	}
	e, ok := f.live.entity(id)
	if !ok {
		t.Fatal("placed escort actor absent", id)
	}
	return e
}

func escortCentre(t *testing.T, f *FrontEnd, id sim.EntityID) {
	t.Helper()
	actions := f.live.world.Actions()
	for i := range actions.Actors {
		a := &actions.Actors[i]
		if a.Entity != id {
			continue
		}
		a.Transit, a.TransitTotal, a.Stride = 0, 0, sim.NativeStride{}
		a.DesiredFacing, a.TurnRemaining, a.TurnTotal, a.TurnState = a.Facing, 0, 0, nil
		a.Route = nil
	}
	if err := f.live.world.RestoreActions(actions, nil); err != nil {
		t.Fatal(err)
	}
}

func escortNativeContinuation(t *testing.T, f *FrontEnd) {
	t.Helper()
	snapshot, label, err := f.Snapshot(true)
	if err != nil {
		t.Fatal(err)
	}
	wire, err := EncodeSave(snapshot, label)
	if err != nil {
		t.Fatal(err)
	}
	decoded, _, err := DecodeSave(wire)
	if err != nil {
		t.Fatal(err)
	}
	back := releaseFront(t)
	opener, town, err := back.Restore(decoded)
	if err != nil || town || opener == nil {
		t.Fatal("engine save restore", err, town)
	}
	app := back.App("escort engine restore")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	if err := app.OpenMission(opener); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(snapshot.World, marshalWorld(t, back.live.world)) {
		t.Fatal("engine save changed escort World bytes")
	}
	var control sim.World
	if err := control.UnmarshalBinary(snapshot.World); err != nil {
		t.Fatal(err)
	}
	mapload.BindSourceDerive(&control)
	for tick := range 33 {
		sim.Step(&control, nil)
		sim.Step(back.live.world, nil)
		if control.Hash() != back.live.world.Hash() {
			t.Fatalf("engine escort continuation diverged at tick %d", tick+1)
		}
	}
}

func escortF2Save(t *testing.T, f *FrontEnd, app *ui.App, name string) (string, []byte) {
	t.Helper()
	before := f.live.world.Hash()
	for attempt := 0; attempt < 16 && app.HeadlessNoticeOpen(); attempt++ {
		text, kind, _ := f.LiveNotice()
		if kind == ui.NoticeFailure {
			t.Fatalf("escort checkpoint reached a failure notice: %q", text)
		}
		t.Logf("escort SAVE acknowledges notice kind %d through Escape", kind)
		if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap || f.live.world.Hash() != before {
			t.Fatal("escort notice acknowledgement changed GAME", err, app.Screen(), app.HeadlessMessage())
		}
	}
	text, kind, notice := f.LiveNotice()
	if err := app.HeadlessKey("f2"); err != nil || app.Screen() != ui.ScreenSave {
		t.Fatalf("escort F2 SAVE: error=%v screen=%s message=%q notice=%t kind=%d text=%q", err, app.Screen(), app.HeadlessMessage(), notice, kind, text)
	}
	dir := t.TempDir()
	if err := app.HeadlessSaveEdit(dir, name, ui.SaveSAV); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessSaveAction("save"); err != nil {
		t.Fatal(err)
	}
	if f.live.world.Hash() != before {
		t.Fatal("escort F2 SAVE changed World")
	}
	raw, err := os.ReadFile(filepath.Join(dir, name+".sav"))
	if err != nil {
		t.Fatal(err)
	}
	return dir, raw
}

func escortColdCheckpoint(t *testing.T, f *FrontEnd, app *ui.App, actor sim.EntityID, name string, lose func(*sim.ActorContinuation)) (*FrontEnd, *FrontEnd) {
	t.Helper()
	escortNativeContinuation(t, f)
	dir, wire := escortF2Save(t, f, app, name)
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatal("SAVE return to GAME", err, app.Screen())
	}
	cold, _ := castOrderSession(t, dir)
	if cold.live.world.Hash() != f.live.world.Hash() {
		logCurrentCarrierDiff(t, f.live.world, cold.live.world)
		t.Fatal("cold LOAD changed escort World hash")
	}
	doc, err := sav.DecodeDocumentData(wire)
	if err != nil {
		t.Fatal(err)
	}
	current, err := readCurrentActions(&doc)
	if err != nil || current == nil {
		t.Fatal("escort actions absent", err)
	}
	found := false
	for i := range current.Actions.Actors {
		if current.Actions.Actors[i].Entity == actor {
			lose(&current.Actions.Actors[i])
			found = true
		}
	}
	if !found {
		t.Fatal("escort action loss found no actor")
	}
	leaf, err := json.Marshal(current)
	if err != nil {
		t.Fatal(err)
	}
	if err := sav.SetNativeActions(&doc.State, leaf); err != nil {
		t.Fatal(err)
	}
	lost, err := sav.EncodeDocumentData(doc)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(wire, lost) {
		t.Fatal("escort loss changed no SAV bytes")
	}
	lossDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(lossDir, "escort loss.sav"), lost, 0o600); err != nil {
		t.Fatal(err)
	}
	loss, _ := castOrderSession(t, lossDir)
	if loss.live.world.Hash() == f.live.world.Hash() {
		t.Fatal("escort loss changed no decoded state")
	}
	return cold, loss
}

func escortActor(t *testing.T, f *FrontEnd, id sim.EntityID) sim.Entity {
	t.Helper()
	e, ok := f.live.entity(id)
	if !ok {
		t.Fatal("escort actor absent", id)
	}
	return e
}

func TestReleaseMissionEscortCloseReaimSAVColdLoadAndNative(t *testing.T) {
	f, app, follower, hero, x, y := installedEscort(t)
	escortPlace(t, f, hero, x+4, y)
	escortPlace(t, f, follower, x-4, y)
	for k := 0; k < 32 && escortActor(t, f, follower).EscortOrder != 1; k++ {
		f.live.tick()
	}
	before := escortActor(t, f, follower)
	if before.EscortOrder != 1 || !before.HasTarget {
		t.Fatal("installed Follow did not close", before)
	}
	if f.live.world.Tick()%16 == 6 {
		f.live.tick()
	}
	escortCentre(t, f, follower)
	cold, loss := escortColdCheckpoint(t, f, app, follower, "Escort closing", func(a *sim.ActorContinuation) { a.EscortOrder = 0 })
	for _, front := range []*FrontEnd{f, cold, loss} {
		escortPlace(t, front, hero, x+4, y+4)
	}
	charge := escortActor(t, f, hero)
	f.live.tick()
	cold.live.tick()
	loss.live.tick()
	closing, forgotten := escortActor(t, f, follower), escortActor(t, loss, follower)
	if !closing.HasTarget || closing.TargetX != charge.X || closing.TargetY != charge.Y {
		t.Fatal("centred escort retained the previous charge cell between actor passes", closing, charge)
	}
	if forgotten.TargetX == closing.TargetX && forgotten.TargetY == closing.TargetY {
		t.Fatal("close-order loss did not alter the next re-aim")
	}
	if f.live.world.Hash() != cold.live.world.Hash() {
		t.Fatal("cold LOAD changed the next escort re-aim")
	}
	for _, front := range []*FrontEnd{f, cold} {
		charge = escortActor(t, front, hero)
		for attempt := 0; ; attempt++ {
			actor := escortPlace(t, front, follower, charge.X-2, charge.Y)
			if max(releaseAbs32(actor.X-charge.X), releaseAbs32(actor.Y-charge.Y)) <= int32(actor.EscortRange) {
				break
			}
			if attempt == 31 {
				t.Fatal("could not place closing escort inside its stop distance")
			}
		}
		escortCentre(t, front, follower)
	}
	if f.live.world.Tick()%16 == 6 {
		f.live.tick()
		cold.live.tick()
	}
	f.live.tick()
	cold.live.tick()
	if e := escortActor(t, f, follower); e.HasTarget || e.EscortOrder != 1 {
		t.Fatal("closing escort did not stop inside the charge range", e)
	}
	for tick := range 48 {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("cold escort close/stop continuation diverged at tick %d", tick+1)
		}
	}
	t.Logf("mission 100 group 21 actor %d follows %d: F2 SAVE, cold LOAD and engine save keep close; one centred sub-tick re-aims and stops; 48 later hashes equal; loss changes re-aim", follower, hero)
}

func TestReleaseMissionEscortIdleTurnSAVColdLoadAndNative(t *testing.T) {
	f, app, follower, hero, x, y := installedEscort(t)
	for k := 0; k < 32; k++ {
		charge := escortPlace(t, f, hero, x, y)
		actor := escortPlace(t, f, follower, x+2, y)
		distance := max(releaseAbs32(actor.X-charge.X), releaseAbs32(actor.Y-charge.Y))
		if distance >= 2 && distance <= int32(actor.EscortRange) {
			break
		}
		if k == 31 {
			t.Fatal("could not place the installed escort inside its idle range")
		}
	}
	escortCentre(t, f, follower)
	for k := 0; k < 32 && escortActor(t, f, follower).EscortOrder != 2; k++ {
		f.live.tick()
	}
	before := escortActor(t, f, follower)
	if before.EscortOrder != 2 || before.HasTarget || before.HasAttackTarget || before.Owner == sim.SelfSlot {
		t.Fatal("installed AI Follow did not enter idle", before)
	}
	if f.live.world.Tick()%16 == 6 {
		f.live.tick()
	}
	escortCentre(t, f, follower)
	actions := f.live.world.Actions()
	for i := range actions.Actors {
		if actions.Actors[i].Entity == follower {
			actions.Actors[i].EscortTurnPending = true
		}
	}
	if err := f.live.world.RestoreActions(actions, nil); err != nil {
		t.Fatal(err)
	}
	before = escortActor(t, f, follower)
	cold, loss := escortColdCheckpoint(t, f, app, follower, "Escort idle", func(a *sim.ActorContinuation) { a.EscortTurnPending = false })
	f.live.tick()
	cold.live.tick()
	loss.live.tick()
	turned, forgotten := escortActor(t, f, follower), escortActor(t, loss, follower)
	if turned.EscortTurnPending || turned.Facing == before.Facing && turned.DesiredFacing == before.DesiredFacing {
		t.Fatal("idle escort did not consume its pending turn between actor passes", turned)
	}
	if turned.Facing == forgotten.Facing && turned.DesiredFacing == forgotten.DesiredFacing {
		t.Fatal("pending-turn loss did not alter the next idle turn")
	}
	for tick := range 48 {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("cold escort idle continuation diverged at tick %d", tick+1)
		}
	}
	t.Logf("mission 100 group 21 actor %d: installed idle order, fixture-centred motion and pending turn; F2 SAVE/cold LOAD and engine save keep state; forced turn consumes pending flag; 48 later hashes equal; loss changes turn", follower)
}

func TestReleasePlayerDefenderInstalledHealSAVColdLoadAndNative(t *testing.T) {
	f := releaseFront(t)
	f.Options = OptionsStore{}
	f.SetDeterministicFrames(true)
	mage := f.ChargenParty(ui.ChargenResult{Name: "Defender", Choices: []int{0, 1, 3}, Stats: []int{40, 30, 40, 40}})
	charge := f.ChargenParty(ui.ChargenResult{Name: "Charge", Choices: []int{0, 0, 0}, Stats: []int{40, 30, 30, 30}})
	charge[0].ID, charge[0].StartingHero, charge[0].PlayerCharacter = "charge", false, false
	party := append(mage, charge[0])
	app := f.App("defender heal")
	t.Cleanup(app.StopAudio)
	app.Layout(1024, 768)
	f.ConfigureSaveSeams(app, SaveStore{Dir: t.TempDir()}, OriginalStore{}, nil)
	if err := app.OpenMission(f.MissionOpenerWith(10, party)); err != nil {
		t.Fatal(err)
	}
	for k := 0; k < 16 && f.live.mission.open; k++ {
		if err := app.HeadlessKey("enter"); err != nil {
			t.Fatal(err)
		}
	}
	releasePauseMission(t, f, app)
	caster, target := f.live.mission.ids[0], f.live.mission.ids[1]
	me := escortActor(t, f, caster)
	heal, ok := f.live.world.Spell(6)
	resolved, known := sim.BookRuleFor(me, heal)
	if !ok || !heal.Restorative || !known || me.KnownSpells&(1<<6) == 0 || !me.Book.WirePresent(me.KnownSpells) || me.Mana < resolved.ManaCost {
		t.Fatal("installed generated mage does not carry the affordable spell 6 healing book", me.KnownSpells, me.Book, me.Mana, heal)
	}
	m := f.live.mission.state.Map
	var x, y, size int
	block := mapload.PassabilityWith(m, f.Table)
	for _, candidate := range []int{13, 11, 9, 7, 5} {
		if px, py, found := openGroundSquare(block, int(m.Width), int(m.Height), candidate); found {
			x, y, size = px, py, candidate
			break
		}
	}
	if size == 0 {
		t.Fatal("mission 10 has no open square for the defender placement")
	}
	for attempt := 0; ; attempt++ {
		protected := escortPlace(t, f, target, int32(x+size/2), int32(y+size/2))
		defender := escortPlace(t, f, caster, int32(x+size/2+2), int32(y+size/2))
		distance := max(releaseAbs32(defender.X-protected.X), releaseAbs32(defender.Y-protected.Y))
		if distance >= 2 && distance <= 3 {
			break
		}
		if attempt == 31 {
			t.Fatal("could not place the defender inside its protection range")
		}
	}
	if !f.live.world.SetAutoHealing(sim.SelfSlot, 0) {
		t.Fatal("could not set the full mana reserve")
	}
	for i := range f.live.fog.visible {
		f.live.fog.visible[i], f.live.fog.explored[i] = 1, 1
	}
	f.live.push()
	castOrderSelect(t, app, f.live, caster)
	if err := app.HeadlessKey("defend"); err != nil {
		t.Fatal(err)
	}
	px, py, err := app.HeadlessEntityPoint(uint32(target))
	if err != nil {
		t.Fatal(err)
	}
	for _, edge := range []string{"press", "release"} {
		if err := app.HeadlessPointer(edge, px, py); err != nil {
			t.Fatal(err)
		}
	}
	if len(f.live.pending) != 1 || f.live.pending[0].Kind != sim.KindGroupDefend || f.live.pending[0].Entity != caster || uint32(f.live.pending[0].X) != uint32(target) {
		t.Fatal("Defend did not queue the caster's protection order", f.live.pending)
	}
	f.live.tick()
	if e := escortActor(t, f, caster); e.ActorState != 8 || !e.HasEscortTarget || e.EscortTarget != target {
		t.Fatal("Defend did not install the caster's charge", e)
	}
	for f.live.world.Tick()%16 != 6 {
		f.live.tick()
	}
	protected := escortActor(t, f, target)
	woundedHP := (protected.MaxHP >> 1) - 1
	if err := f.live.world.HeadlessDamage(target, protected.HP-woundedHP); err != nil {
		t.Fatal(err)
	}
	f.live.tick()
	cast, admitted := castOrderBook(f.live.world, caster)
	if !admitted || cast.Spell != 6 || cast.Target != target || cast.Retained || escortActor(t, f, caster).HasAttackTarget {
		t.Fatal("defender did not admit its one-shot installed Heal before fighting", cast, admitted)
	}
	escortNativeContinuation(t, f)
	dir, _ := escortF2Save(t, f, app, "Defender heal")
	if err := app.HeadlessKey("escape"); err != nil || app.Screen() != ui.ScreenMap {
		t.Fatal("SAVE return to GAME", err, app.Screen())
	}
	cold, _ := castOrderSession(t, dir)
	if f.live.world.Hash() != cold.live.world.Hash() {
		t.Fatal("cold LOAD changed the defender's admitted Heal")
	}
	landed := false
	for tick := range 256 {
		f.live.tickWithCastSink(func(events []sim.CastEvent) {
			for _, event := range events {
				if event.Caster == caster && event.Target == target && event.Spell == 6 {
					landed = true
				}
			}
		})
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("cold defender Heal continuation diverged at tick %d", tick+1)
		}
		if landed {
			break
		}
	}
	if !landed || escortActor(t, f, target).HP <= woundedHP {
		t.Fatal("defender's installed Heal did not restore its charge")
	}
	for tick := range 33 {
		f.live.tick()
		cold.live.tick()
		if f.live.world.Hash() != cold.live.world.Hash() {
			t.Fatalf("cold defender post-Heal continuation diverged at tick %d", tick+1)
		}
	}
	t.Logf("mission 10 App Defend %d protects %d: generated mage's installed spell 6 book, full mana reserve and charge below half health; one-shot Heal saved mid-cast through F2/cold LOAD and engine save; restored charge and 33 later hashes equal; party and placement are fixtures", caster, target)
}
