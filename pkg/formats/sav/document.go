package sav

import "fmt"

// document retains the exact envelope index and shared archive objects.
type document struct {
	players, dead             []*Record
	world                     *WorldHalf
	buildings, effects, sacks []*Record
	priorItems                map[*Record]bool
	objects                   map[uint16]*Record
	deadRoots                 DocumentDeadRootLocation
	// trailer is world+0x118's 400-byte block (SAV-790, TrailerBody).
	trailer    TrailerBody
	trailerOff int

	// end is the stream's natural, UNPADDED length (w.p at the read's very
	// end, before SAV-DECPAD-238's alignment byte is considered). A caller
	// that resizes something ahead of the object stream — Head.MapName is
	// the one this package has (SetMapName) — needs it to tell whether the
	// old body carried that alignment byte, which len(body) alone cannot say
	// (it is always even, padded or not).
	end int
}

// exactDocument follows the published envelope under one archive state.
// It returns a document even without a world; empty terrain is still a world.
func (f *File) exactDocument() (*document, bool, error) {
	if f == nil {
		return nil, false, fmt.Errorf("sav: document requires a save")
	}
	w := &walker{b: f.Body, p: f.Head.End, next: 1,
		classes: map[uint16]string{}, objects: map[uint16]*Record{}}
	players, err := w.playerRecords(f.Head.PlayerCount)
	if err != nil {
		return nil, false, err
	}
	deadRoots := DocumentDeadRootLocation{CountOff: w.p}
	dead, err := w.groundCountedList("dead actors", "Unit", &deadRoots.RefOffs)
	if err != nil {
		return nil, false, err
	}
	if w.p >= len(w.b) {
		return nil, false, fmt.Errorf("sav: missing world selector at %d", w.p)
	}
	present := w.b[w.p] != 0
	w.p++
	doc := &document{players: players, dead: dead, deadRoots: deadRoots, priorItems: make(map[*Record]bool)}
	if present {
		var err error
		buildingsOff := w.p
		doc.buildings, err = w.groundCountedList("Buildings", "Building")
		if err != nil {
			return nil, true, err
		}
		buildingsEnd := w.p
		doc.effects, err = w.groundCountedList("SpellEffects", "SpellEffect")
		if err != nil {
			return nil, true, err
		}
		doc.world, err = w.worldHalf()
		if err != nil {
			return nil, true, err
		}
		doc.world.BuildingsOff, doc.world.BuildingsEnd = buildingsOff, buildingsEnd
		for _, record := range w.objects {
			if groundClass(record.Class, "Item") {
				doc.priorItems[record] = true
			}
		}
		doc.sacks, err = w.groundCountedList("Sacks", "Sack")
		if err != nil {
			return nil, true, err
		}
	}
	marker, err := w.u32()
	if err != nil {
		return nil, present, err
	}
	if marker == 0xbadface1 {
		if err := w.skip("trailer global", 4); err != nil {
			return nil, present, err
		}
	}
	doc.trailerOff = w.p
	trailer, err := w.readTrailerBody()
	if err != nil {
		return nil, present, err
	}
	doc.trailer = trailer
	doc.end = w.p
	// The codec aligns an odd logical endpoint by one unconstrained byte.
	if len(w.b) != w.p+(w.p&1) {
		return nil, present, fmt.Errorf("sav: document ends at %d, decoded size %d", w.p, len(w.b))
	}
	doc.objects = w.objects
	return doc, present, nil
}
