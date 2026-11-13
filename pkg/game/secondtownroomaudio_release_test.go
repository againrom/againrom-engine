package game

import (
	"strings"
	"testing"
	"time"

	"againrom/pkg/audio"
	"againrom/pkg/ui"
)

// townAudioRun is one session over the installed second game: the shared
// delivery route with townAudioDevice, the receipts counted by phase.
type townAudioRun struct {
	*deliveryRun
	town   *townScreen
	seen   map[uint64]bool
	phase  string
	counts map[string]map[audio.DeliveryReason]int
}

func newTownAudioRun(t *testing.T) *townAudioRun {
	t.Helper()
	return &townAudioRun{deliveryRun: newDeliveryRun(t, true, true), seen: map[uint64]bool{}, counts: map[string]map[audio.DeliveryReason]int{}}
}

// use makes the run's clock the front end's and binds its town screen.
func (r *townAudioRun) use(f *FrontEnd, app *ui.App, s *townScreen, now *time.Time) {
	r.town = s
	r.deliveryRun.use(f, app, now)
}

func (r *townAudioRun) collect() {
	owner := ui.DeliveryOwner(r.front.SoundPlayer)
	if owner == nil {
		r.t.Fatal("no delivery owner")
	}
	for _, rc := range owner.Service.Snapshot().Receipts {
		if r.seen[rc.Sequence] {
			continue
		}
		r.seen[rc.Sequence] = true
		if r.counts[r.phase] == nil {
			r.counts[r.phase] = map[audio.DeliveryReason]int{}
		}
		r.counts[r.phase][rc.Reason]++
	}
}

func (r *townAudioRun) mark(phase string) { r.collect(); r.phase = phase }

func (r *townAudioRun) step(n int) {
	r.t.Helper()
	for i := 0; i < n; i++ {
		if err := r.app.HeadlessStep(); err != nil {
			r.t.Fatal(err)
		}
		r.now = r.now.Add(20 * time.Millisecond)
		if r.app.Screen() == ui.ScreenTown {
			if r.town.AtTownShop() {
				r.town.AdvanceShopInteriorAnimation()
			} else {
				r.town.AdvanceTownSquareAnimation()
			}
		}
		for _, d := range r.players {
			d.Pump()
		}
		r.collect()
	}
}

// keeps requires that the shown room's sound kept playing in the phase: it
// admitted sounds, stopped none and released its scope at most once, on the
// way in.
func (r *townAudioRun) keeps(phase string) {
	r.t.Helper()
	c := r.counts[phase]
	r.t.Logf("%s: admitted=%d completed=%d stopped=%d scope-destroyed=%d", phase,
		c[audio.DeliveryAdmitted], c[audio.DeliveryCompleted], c[audio.DeliveryStopped], c[audio.DeliveryScopeDestroyed])
	if c[audio.DeliveryAdmitted] == 0 {
		r.t.Errorf("%s: no sound was admitted", phase)
	}
	if c[audio.DeliveryScopeDestroyed] > 1 {
		r.t.Errorf("%s: the room's audio scope was destroyed %d times", phase, c[audio.DeliveryScopeDestroyed])
	}
	if c[audio.DeliveryStopped] > 0 {
		r.t.Errorf("%s: %d sounds were stopped while the room stayed shown", phase, c[audio.DeliveryStopped])
	}
}

// A town shown after another in one process keeps the sound of its square,
// inn, inn talk and shop: the hidden scenes of the town shown earlier hold
// the same room names and must not release the shown room's audio.
func TestReleaseLaterTownKeepsItsRoomSounds(t *testing.T) {
	r := newTownAudioRun(t)
	f := secondGameFront(t)
	app := f.App("later town sound")
	app.Layout(640, 480)
	s := f.TownScreen().(*townScreen)
	app.SetTown(s)
	r.use(f, app, s, nil)
	if err := app.HeadlessActivate("new game"); err != nil {
		t.Fatal(err)
	}
	r.step(200)
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	r.step(200)
	if err := app.HeadlessKey("escape"); err != nil {
		t.Fatal(err)
	}
	r.step(50)
	secondShopActivate(t, f, app, "SHOP")
	if !s.AtTownShop() {
		t.Fatal("town 1's shop is not shown")
	}
	s.CloseTip()
	r.step(100)
	secondShopActivate(t, f, app, "EXIT")
	if !s.AtTownSquare() {
		t.Fatal("town 1's square is not shown after its shop")
	}
	c := f.Town.second
	c.current, c.available, c.room = partyPlace{}, []partyPlace{{2, 2}}, secondTownSquare
	c.bank[768] = 30
	if err := app.HeadlessActivate("town 2"); err != nil {
		t.Fatal(err)
	}
	if c.current != (partyPlace{2, 2}) || !s.AtTownSquare() {
		t.Fatal("town 2 is not shown")
	}
	r.mark("square")
	r.step(600)
	r.keeps("square")
	secondShopActivate(t, f, app, "SHOP")
	if !s.AtTownShop() {
		t.Fatal("the shop is not shown")
	}
	s.CloseTip()
	r.mark("shop")
	r.step(600)
	r.keeps("shop")
	secondShopActivate(t, f, app, "EXIT")
	if !s.AtTownSquare() {
		t.Fatal("the square is not shown after the shop")
	}
	if err := app.HeadlessActivate("TAVERN"); err != nil {
		t.Fatal(err)
	}
	r.mark("inn")
	r.step(600)
	r.keeps("inn")
	talk := ""
	for _, row := range s.Rows() {
		if strings.HasPrefix(row.Text, "TALK ") && talk == "" {
			talk = row.Text
		}
	}
	if talk == "" {
		t.Fatal("the inn offers no talk")
	}
	if err := app.HeadlessActivate(talk); err != nil {
		t.Fatal(err)
	}
	r.mark("talk")
	r.step(400)
	r.keeps("talk")
}

// Loading a save of the town shown earlier keeps that town's square sound.
func TestReleaseLoadOfTheEarlierTownKeepsItsSquareSound(t *testing.T) {
	r := newTownAudioRun(t)
	dir := t.TempDir()
	f0, app0 := secondTownNew(t, false, false)
	secondTownNamedSave(t, f0, app0, dir, "Earlier town")
	app0.StopAudio()
	f, app, s, now := secondTownTwoOf(t, secondGameFront(t))
	r.use(f, app, s, now)
	r.mark("town 2")
	r.step(300)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	_, list, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	token := localOriginalSaveToken(secondSaveFile(t, dir, "Earlier town"))
	label := ""
	for _, row := range list() {
		if row.Name == token {
			label = row.Label
		}
	}
	if label == "" {
		t.Fatal("the save is absent from the LOAD list")
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(label); err != nil {
		t.Fatal(err)
	}
	if app.Screen() != ui.ScreenTown || !s.AtTownSquare() {
		t.Fatal("the load did not show the square", app.Screen())
	}
	if d, _, _ := s.composedRoom(); d.Town != "rom2-town1" {
		t.Fatal("the load did not show town 1:", d.Town)
	}
	r.mark("loaded")
	r.step(600)
	r.keeps("loaded")
}

// A save loaded from the main menu of a fresh process keeps its square sound.
func TestReleaseFreshLoadKeepsItsSquareSound(t *testing.T) {
	r := newTownAudioRun(t)
	dir := t.TempDir()
	f0, app0, _, _ := secondTownTwoOf(t, secondGameFront(t))
	secondTownNamedSave(t, f0, app0, dir, "Fresh town")
	app0.StopAudio()
	f := secondGameFront(t)
	app := f.App("fresh load sound")
	app.Layout(640, 480)
	s := f.TownScreen().(*townScreen)
	app.SetTown(s)
	r.use(f, app, s, nil)
	r.step(30)
	f.ConfigureSaveSeams(app, SaveStore{Dir: dir}, OriginalStore{}, nil)
	_, list, _ := f.SaveSeams(SaveStore{Dir: dir}, OriginalStore{}, nil)
	token := localOriginalSaveToken(secondSaveFile(t, dir, "Fresh town"))
	label := ""
	for _, row := range list() {
		if row.Name == token {
			label = row.Label
		}
	}
	if label == "" {
		t.Fatal("the save is absent from the LOAD list")
	}
	if err := headlessOpenLoad(app); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessActivate(label); err != nil {
		t.Fatal(err)
	}
	r.mark("loaded")
	r.step(600)
	r.keeps("loaded")
}
