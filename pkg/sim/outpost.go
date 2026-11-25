package sim

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"slices"
)

// Outpost is one type-10 Outpost (R2-ENGINE-510): the computer Group of Owner
// it watches, the cell and radius its members come back over, the reload and
// the countdown in container passes, whether the countdown runs, and each
// member with the template it is re-created from.
type Outpost struct {
	Owner, Group uint32
	X, Y         int32
	Radius       uint8
	Reload       int32
	Countdown    int32
	Counting     bool
	Members      []PassiveBody
}

// OutpostMap is what a map gives the type-10 pass: whether the pass runs
// (type-0 dword 0x84 above 1, R2-ENGINE-509), whether the build keeps the
// members' spread as the radius (R2-ENGINE-511), and the entries in file order.
type OutpostMap struct {
	Runs, Spread bool
	Entries      []OutpostEntry
}

// OutpostEntry is the part of a type-10 entry the pass carries: the group
// number it matches, the reload percent and the flag bits (R2-ASSET-120).
type OutpostEntry struct {
	Group   uint32
	Percent int32
	Flags   uint32
}

const (
	// outpostScale is S of R2-ENGINE-1819: the constructor's 100.
	outpostScale = 100
	// outpostPercent is the reload percent of a group no entry names.
	outpostPercent = 120
	// outpostKeepsSpread is the entry flag whose absence sets the radius to 0.
	outpostKeepsSpread = 2
	// outpostLoadedReload is the reload a loaded Outpost holds (R2-SESSION-503).
	outpostLoadedReload = 120
)

// DeclareOutposts runs the type-10 pass over the World's Group registry: one
// Outpost for every Group of a computer owner that has a listed member, at
// the members' centre, with each member's template; matching entries apply
// in file order (R2-ENGINE-510, R2-ENGINE-1816). A game without Outposts and
// a map whose gate is shut build none.
func (w *World) DeclareOutposts(in OutpostMap) {
	w.outposts = nil
	if !w.game.outposts || !in.Runs {
		return
	}
	for _, g := range w.savedGroups.Groups {
		owner := g.Owner.Owner
		if owned, human := w.slotControl(owner); g.Owner.Class != 1 || !owned || human {
			continue
		}
		var members []int
		for _, m := range g.Members {
			if i := indexOfEntity(w.entities, m.Entity); m.Bound && i >= 0 && w.scriptCountsMember(w.entities[i]) {
				members = append(members, i)
			}
		}
		if len(members) == 0 {
			continue
		}
		o := Outpost{Owner: owner, Group: g.Selector, Reload: outpostPercent * outpostScale / 100}
		o.X, o.Y = groupCentroid(w.entities, members, subCellMean)
		if in.Spread {
			o.Radius = memberExtent(w.entities, members)
		}
		for _, e := range in.Entries {
			if e.Group != o.Group {
				continue
			}
			o.Reload = e.Percent * outpostScale / 100
			if e.Flags&outpostKeepsSpread == 0 {
				o.Radius = 0
			}
		}
		o.Countdown = o.Reload
		for _, i := range members {
			t := w.passiveHoldings(i)
			body := passiveEntity(w.entities[i])
			t.Body = &body
			o.Members = append(o.Members, t)
		}
		w.outposts = append(w.outposts, o)
	}
}

// memberExtent is the larger of the members' x and y extents, held to a byte.
func memberExtent(ents []Entity, members []int) uint8 {
	lo, hi := cellOf(&ents[members[0]]), cellOf(&ents[members[0]])
	for _, i := range members {
		c := cellOf(&ents[i])
		lo.x, lo.y, hi.x, hi.y = min(lo.x, c.x), min(lo.y, c.y), max(hi.x, c.x), max(hi.y, c.y)
	}
	return uint8(min(max(hi.x-lo.x, hi.y-lo.y), 255))
}

// Outposts returns the World's Outposts as fresh copies.
func (w *World) Outposts() []Outpost {
	out := make([]Outpost, len(w.outposts))
	for i, o := range w.outposts {
		out[i] = o.clone()
	}
	return out
}

func (o Outpost) clone() Outpost {
	o.Members = slices.Clone(o.Members)
	for i := range o.Members {
		o.Members[i] = o.Members[i].Clone()
	}
	return o
}

// AdoptLoadedOutposts gives every Outpost the state a load builds: no member
// or template, group 0, reload 120 and a stopped countdown (R2-SESSION-503).
// Owner and cell stay; the countdown a load leaves is Unknown (DIV-5185).
func (w *World) AdoptLoadedOutposts() {
	if len(w.outposts) == 0 {
		return
	}
	loaded := make([]Outpost, len(w.outposts))
	for i, o := range w.outposts {
		loaded[i] = Outpost{Owner: o.Owner, X: o.X, Y: o.Y, Reload: outpostLoadedReload}
	}
	w.outposts = loaded
}

// setOutpostCountdown is action 31 (R2-ENGINE-1701, R2-ENGINE-1820): every
// Outpost of every owner whose group number is group takes countdown,
// unscaled. A countdown that has not started still waits for the members.
func (w *World) setOutpostCountdown(group uint32, countdown int32) {
	w.outposts = slices.Clone(w.outposts)
	for i := range w.outposts {
		if w.outposts[i].Group == group {
			w.outposts[i].Countdown = countdown
		}
	}
}

// outpostPass is slot 5 of each Outpost, once per container pass
// (R2-ENGINE-1817): the countdown starts once no member stands below decay
// stage 3, and from then counts down each pass; when the value before the
// step is 0 or below and the Group lists no member, the members come back
// (R2-ENGINE-1818). An Outpost with no member does nothing (DIV-5185).
func (w *World) outpostPass() {
	if len(w.outposts) == 0 {
		return
	}
	w.outposts = slices.Clone(w.outposts)
	for k := range w.outposts {
		if len(w.outposts[k].Members) != 0 && w.outpostDue(&w.outposts[k]) {
			w.returnOutpostMembers(k)
		}
	}
}

func (w *World) outpostDue(o *Outpost) bool {
	if !o.Counting {
		for _, m := range o.Members {
			if i := indexOfEntity(w.entities, m.ID); i >= 0 && w.entities[i].Decay < decayDarkStage {
				return false
			}
		}
		o.Counting = true
	}
	before := o.Countdown
	o.Countdown--
	if before > 0 {
		return false
	}
	if g := w.outpostGroup(o.Owner, o.Group); g != nil {
		for _, m := range g.Members {
			if i := indexOfEntity(w.entities, m.Entity); m.Bound && i >= 0 && w.scriptCountsMember(w.entities[i]) {
				return false
			}
		}
	}
	return true
}

// outpostGroup is owner's Group with the selector, nil when none holds it.
func (w *World) outpostGroup(owner, selector uint32) *SavedGroup {
	for i := range w.savedGroups.Groups {
		g := &w.savedGroups.Groups[i]
		if g.Owner.Class == 1 && g.Owner.Owner == owner && g.Selector == selector {
			return g
		}
	}
	return nil
}

// returnOutpostMembers re-creates each member from its template through the
// one actor builder and re-forms the Group; a member still standing, or one
// no cell takes, stays as it is. Any member placed restarts the countdown.
func (w *World) returnOutpostMembers(k int) {
	o := w.outposts[k]
	var placed []EntityID
	for _, m := range o.Members {
		if id, ok := w.recreateOutpostMember(o, m); ok {
			placed = append(placed, id)
		}
	}
	if len(placed) == 0 {
		return
	}
	w.outposts[k].Counting, w.outposts[k].Countdown = false, w.outposts[k].Reload
	w.reformOutpostGroup(o.Owner, o.Group, placed)
}

// recreateOutpostMember removes the member's body, takes its unit off the
// dead list and builds it again from the template under its own id: around
// the Outpost at its radius, else at the member's start cell (DIV-5184).
func (w *World) recreateOutpostMember(o Outpost, m PassiveBody) (EntityID, bool) {
	if i := indexOfEntity(w.entities, m.ID); i >= 0 {
		if w.entities[i].Decay < decayDarkStage || !w.remove([]EntityID{m.ID}) {
			return 0, false
		}
	}
	w.takePooledDead(m.ID)
	t := m.Clone()
	body := *t.Body
	body.Group, body.CommandGroup = o.Group, 0
	x, y, r := body.X, body.Y, int32(0)
	if o.Radius != 0 {
		x, y, r = o.X, o.Y, int32(o.Radius)
	}
	id, err := w.createActor(body, t.Carried, t.Worn, x, y, outpostPlacement(x, y, r), true,
		actorCreationProfile{reuse: m.ID, ungrouped: true})
	return id, err == nil
}

// outpostPlacement is the positioning search around (x, y) at radius r
// (R2-ENGINE-531), then the free cell nearest (x, y) when it finds none.
func outpostPlacement(x, y, r int32) actorSearch {
	return func(n *World, i int) (int32, int32, bool) {
		if px, py, ok := secondPlacement(x, y, r)(n, i); ok {
			return px, py, true
		}
		return nearestFreeCell(x, y)(n, i)
	}
}

// reformOutpostGroup puts the returned members in owner's Group with the
// selector, built by the mission-start Group builder and keeping the
// Group's identity; a Group the registry lost is appended to the owner's.
func (w *World) reformOutpostGroup(owner, selector uint32, ids []EntityID) {
	w.savedGroups = cloneSavedGroups(w.savedGroups)
	var members []int
	for _, id := range ids {
		if i := indexOfEntity(w.entities, id); i >= 0 {
			members = append(members, i)
		}
	}
	g, orders := w.constructedGroup(owner, selector, members, w.scriptNamedGroups(), false)
	if old := w.outpostGroup(owner, selector); old != nil {
		g.ID, g.ContainerID, g.OwnerID, g.Authored, g.Reference = old.ID, old.ContainerID, old.OwnerID, old.Authored, old.Reference
		g.Members = append(slices.Clone(old.Members), g.Members...)
		*old = g
	} else {
		if !w.allocSavedGroup(&g, w.uniqueSavedPlayerSlot(owner), false) {
			return
		}
		w.appendSavedCommandGroup(g)
	}
	for _, o := range orders {
		if i := indexOfEntity(w.entities, o.Entity); i >= 0 {
			*w.ensureSavedOrder(i) = o
		}
	}
}

// outpostFormVersion wraps only Worlds that hold an Outpost. Older forms hold
// none.
const outpostFormVersion byte = 146

type outpostContinuation struct {
	Outposts []Outpost
}

func (w *World) appendOutposts(b []byte) []byte {
	if len(w.outposts) == 0 {
		return b
	}
	ws := beginWrapper(b)
	raw, _ := json.Marshal(outpostContinuation{Outposts: w.outposts})
	return endWrapper(append(b, raw...), ws, outpostFormVersion)
}

func (w *World) unmarshalOutposts(data []byte, proof formProof) error {
	fail := func() error { return fmt.Errorf("malformed outpost section") }
	env, ok := readWrapper(data, outpostFormVersion)
	size := uint64(len(env.section()))
	if !ok || size == 0 || size > maxCarriedResumeBytes {
		return fail()
	}
	var c outpostContinuation
	d := json.NewDecoder(bytes.NewReader(env.section()))
	d.DisallowUnknownFields()
	if d.Decode(&c) != nil || d.Decode(new(any)) != io.EOF || len(c.Outposts) == 0 {
		return fail()
	}
	for _, o := range c.Outposts {
		for _, m := range o.Members {
			if m.Body == nil || m.Body.ID != m.ID {
				return fail()
			}
		}
	}
	next := *w
	if err := next.unmarshalForm(env.base(), proof); err != nil {
		return err
	}
	next.outposts = c.Outposts
	if !next.marshalsTo(data, proof) {
		return fail()
	}
	*w = next
	return nil
}
