package game

import (
	"time"

	"againrom/pkg/ui"
)

// The horse, baba and dervish families of the town square (TOWN-439..447,
// TOWN-489..495). Everything here is presentation: it draws from its own
// generator and touches no simulation state or save.

// townCRT is the original's random source, the MSVC linear congruential
// generator returning 0..0x7fff (AI-RAND-058). Its seed is not known, so it is
// the front end's process-local presentation seed, taken once per town screen.
// raw replaces the source in tests.
type townCRT struct {
	state  uint32
	seeded bool
	raw    func() int
}

func (g *townCRT) next(seed time.Time) int {
	if g.raw != nil {
		return g.raw()
	}
	if !g.seeded {
		g.state, g.seeded = uint32(seed.UnixNano()), true
	}
	g.state = g.state*214013 + 2531011
	return int(g.state>>16) & 0x7fff
}

// scaled is (r*n)/0x7fff mod n (TOWN-491, TOWN-004).
func (g *townCRT) scaled(seed time.Time, n int) int {
	return (g.next(seed) * n / 0x7fff) % n
}

// quarter is (r*4)/0x7fff and 3 (TOWN-004).
func (g *townCRT) quarter(seed time.Time) int {
	return (g.next(seed) * 4 / 0x7fff) & 3
}

type townFamily struct {
	position, sheet int
	current         int // -1 draws frame 0 without an episode
	active          bool
	clock           time.Time
	delay           time.Duration
}

type townFamilies struct {
	entered              bool
	horse, baba, dervish townFamily
}

const (
	townFamilyHorseFrames  = 15
	townFamilyBabaFrames   = 31 // sheet A1; A2 has 32
	townFamilyDervishFrame = 30
	townHorse1Sound        = "town/horse1.wav"
	townHorse2Sound        = "town/horse2.wav"
)

// enterTownFamilies is the view's entry: positions are drawn, horse and baba
// start on sheet A1 with their entry delay (2000..3999 ms) and the dervish
// starts its revolution (TOWN-004, TOWN-439, TOWN-491).
func (t *townScreen) enterTownFamilies(now time.Time) {
	g := &t.townFamilyRand
	fam := &t.exterior.fam
	*fam = townFamilies{entered: true}
	seed := time.Unix(0, t.sound.ambientSeed())
	fam.horse.position = g.scaled(seed, ui.TownHorsePositions)
	fam.baba.position = g.quarter(seed)
	for {
		fam.dervish.position = g.quarter(seed)
		if fam.dervish.position != fam.baba.position {
			break
		}
	}
	fam.baba.current, fam.baba.clock = -1, now
	fam.baba.delay = time.Duration(2000+g.scaled(seed, 2000)) * time.Millisecond
	fam.horse.current, fam.horse.clock = -1, now
	fam.horse.delay = time.Duration(2000+g.scaled(seed, 2000)) * time.Millisecond
	fam.dervish.current, fam.dervish.active = 0, true
}

func (t *townScreen) townFamilySheetLen(f *townFamily, kind int) int {
	art := t.townExteriorArt()
	if art != nil {
		switch kind {
		case 0:
			if n := len(art.Horse[f.position][f.sheet]); n > 0 {
				return n
			}
		case 1:
			if n := len(art.Baba[f.position][f.sheet]); n > 0 {
				return n
			}
		default:
			if n := len(art.Dervish[f.position]); n > 0 {
				return n
			}
		}
	}
	switch kind {
	case 0:
		return townFamilyHorseFrames
	case 1:
		return townFamilyBabaFrames + f.sheet
	}
	return townFamilyDervishFrame
}

// stepTownFamilies runs once per admitted hub, in the original's order
// dervish, baba, horse (TOWN-492). A step advances one frame, refreshes the
// family clock and ends the episode at the sheet's terminal count.
func (t *townScreen) stepTownFamilies(now time.Time) {
	fam := &t.exterior.fam
	if !fam.entered {
		return
	}
	d := &fam.dervish
	if d.active {
		if art := t.townExteriorArt(); art == nil || len(art.Dervish[d.position]) == 0 {
			d.current = -1
		} else {
			d.current = (d.current + 1) % len(art.Dervish[d.position])
		}
	}
	for i, f := range []*townFamily{&fam.baba, &fam.horse} {
		if !f.active {
			continue
		}
		f.current++
		f.clock = now
		if f.current >= t.townFamilySheetLen(f, 1-i) {
			f.current, f.active = -1, false
		}
	}
}

// paintTownFamilies is the per-paint half: the strict delay test for horse and
// baba, then the horse sound gates (TOWN-445, TOWN-491, TOWN-494).
func (t *townScreen) paintTownFamilies(now time.Time) {
	fam := &t.exterior.fam
	if !fam.entered {
		return
	}
	g := &t.townFamilyRand
	if now.Sub(fam.baba.clock) > fam.baba.delay {
		fam.baba.clock, fam.baba.current, fam.baba.active = now, 0, true
		fam.baba.delay = time.Duration(2000+g.scaled(now, 5000)) * time.Millisecond
		fam.baba.sheet = g.scaled(now, ui.TownBabaSheets)
	}
	if now.Sub(fam.horse.clock) > fam.horse.delay {
		fam.horse.clock, fam.horse.current, fam.horse.active = now, 0, true
		fam.horse.delay = time.Duration(2000+g.scaled(now, 5000)) * time.Millisecond
		fam.horse.sheet = g.scaled(now, ui.TownHorseSheets)
	}
	art := t.townExteriorArt()
	h := fam.horse
	if art == nil || len(art.Horse[h.position][h.sheet]) == 0 {
		return
	}
	switch {
	case h.sheet == 2 && h.current == 1:
		t.exteriorSound(exteriorHorse1, townHorse1Sound)
	case h.current == 14, h.sheet == 1 && h.current == 8:
		t.exteriorSound(exteriorHorse2, townHorse2Sound)
	}
}

func (t *townScreen) townFamilyFrames(f *ui.TownExteriorFrame) {
	fam := &t.exterior.fam
	if !fam.entered || !t.exterior.present {
		return
	}
	show := func(x townFamily) ui.TownFamilyFrame {
		return ui.TownFamilyFrame{Visible: true, Position: x.position, Sheet: x.sheet, Frame: max(x.current, 0)}
	}
	f.Horse, f.Baba, f.Dervish = show(fam.horse), show(fam.baba), show(fam.dervish)
}
