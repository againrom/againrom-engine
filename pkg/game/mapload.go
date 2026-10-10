package game

import (
	"image"

	"againrom/pkg/base"
	"againrom/pkg/formats/alm"
	"againrom/pkg/mapload"
	"againrom/pkg/render/terrain"
	"againrom/pkg/ui"
)

// Markers selects which diagnostic placement-marker overlays a loaded map shows:
// the placed structures of 0008 (type-4 records), the placed units of 0009
// (type-6 records) and the static-object cells of 0017 (the type-3 grid),
// independently.
//
// It is a PARAMETER of the load path rather than something a caller switches on
// afterwards, and that is the whole point (0010 DD33). The shipped defect this
// type exists to close was exactly a post-hoc setter: the overlays worked, the
// standalone viewer turned them on inside its own main, and the game — which
// goes through the same load path — silently drew terrain and nothing else for
// five stories. A caller can still ask for neither; what it can no longer do is
// fail to answer the question.
//
// THREE FIELDS, ONE QUESTION. The game reaches all three through the single
// -markers switch 0010 DD33 gave it, so a story that adds a glyph adds a
// field here and nothing to that command line; the developer tools keep
// offering the three independently, which is what a diagnostic surface is
// for.
type Markers struct {
	Objects bool
	Units   bool

	// Statics is the static-object layer's own cross: one glyph per cell whose
	// placement byte resolves to a drawable frame.
	//
	// Unlike the other two it carries NO cell list out of this package, and
	// that asymmetry is the contract rather than an omission. Which cells are
	// marked is the question "which cells resolve", answerable only from the
	// loaded object classes, so the viewer takes it from the placement list it
	// built; WHERE each mark then goes is decided from the cell alone by the
	// marker geometry, reading no class field and no frame size. MarkerCells is
	// therefore untouched by this field — see its own doc for why that
	// independence is the instrument and not a tidiness question.
	Statics bool
}

// StaticLayer is a map's static-object layer as the load path takes it: the
// loaded object-class bundle, and whether its sprites paint.
//
// It rides beside Markers for the same reason Markers itself exists — the
// bundle is a PARAMETER of loading a map, never something handed to a viewer
// afterwards. ui.NewViewerWithStatics offers no setter at all, so a front-end
// that does not pass a bundle here is a front-end that draws bare ground, and
// the way to find that out is to read one call site rather than to look for a
// call that is missing.
//
// The zero value is exactly the pre-story load path: no bundle, no
// placements, no object draw, and a viewer byte-for-byte the one that
// shipped.
//
// ART AND THE CROSS ARE TWO SWITCHES, and only one of them is here. Either
// must be usable without the other, yet both need the bundle — the cross's
// cells come out of the built placement list — so a caller asking for the
// cross alone still supplies Set and leaves Art off: the lists are built,
// every resolving cell is marked, and no sprite is painted. The cross itself
// is Markers.Statics, because it is a diagnostic glyph and belongs with the
// other two; Art is here, because it is the map's own content.
//
// AnimGate selects the cycle-capable subset. AnimGateTiles keeps the decoded
// four-corner test for tools that inspect it. AnimGateAll admits every class
// with a non-zero period; the game uses that population and the viewer applies
// current fog visibility at draw time. The zero value remains AnimGateTiles for
// older callers.
type StaticLayer struct {
	Set      *terrain.StaticSet
	Art      bool
	AnimGate bool
}

// StructureLayer is a map's STRUCTURE layer as the load path takes it: the
// loaded structure-class bundle, and whether its art paints.
//
// It rides beside StaticLayer for that type's own reason, and it is a SECOND
// PARAMETER rather than a field of it: the two layers share no class, no sheet
// selector and no placement arithmetic — an object is a sprite anchored by a
// canvas, a structure is a rectangle of tile-sized frames anchored by a cell — so
// a single struct carrying both would be one parameter two loaders fill and two
// builders read, and its zero value would no longer say which of them is off.
//
// The zero value is exactly the pre-story load path: no bundle, no entries,
// no structure draw, and a viewer byte-for-byte the one that shipped.
//
// It carries NO marker switch, and that asymmetry with StaticLayer is the
// contract. The cross a placed structure gets is the type-4 marker overlay
// Markers.Objects has drawn since 0008: it derives its cell from the record
// alone and MUST keep doing so, because art standing away from its own cross
// is the instrument that catches a misplaced building. Giving this layer a
// marker of its own would be a second cross derived from this story's own
// geometry, which can only ever agree with itself.
type StructureLayer struct {
	Set *terrain.StructureSet
	Art bool
}

// MarkerCells is the ONE conversion from a decoded map's placement records to
// the anchor cells the two diagnostic overlays mark.
//
// Each cell comes from its own record's stored anchor and from nothing else
// — terrain.AnchorCell of that record's (X, Y), which is the cell the
// marker's cross is then centred in. It reads no class, no sprite sheet, no
// frame size and no other record, and it MUST stay that way.
//
// That independence is a measuring instrument, not a style preference. Object
// art is placed from an anchor built out of the class canvas and the drawn
// frame; when that placement is wrong, the art stands away from the cross and a
// human sees it at once — which is how a two-tile error was caught the last time
// this engine was built. Route both through one shared "where does this stand"
// helper and a wrong anchor moves the marker with the art: the screen stays
// self-consistent, and the disagreement stops being representable at all. The
// duplication here is the discriminating power.
func MarkerCells(m *alm.Map) (objects, units []image.Point) {
	objects = make([]image.Point, 0, len(m.Objects))
	for _, o := range m.Objects {
		col, row := terrain.AnchorCell(o.X, o.Y)
		objects = append(objects, image.Point{X: col, Y: row})
	}
	units = make([]image.Point, 0, len(m.Units))
	for _, u := range m.Units {
		col, row := terrain.AnchorCell(u.X, u.Y)
		units = append(units, image.Point{X: col, Y: row})
	}
	return objects, units
}

// MapView is a loaded map ready to show: the viewer that draws it, the decoded
// map behind it, and the title the window carries.
//
// The decoded map comes back because a caller may need more of it than the
// viewer does — the standalone viewer reports counts and builds diagnostic
// overlays from it — and re-decoding to get at that would be both wasteful and a
// second chance to disagree about what the file says.
type MapView struct {
	Viewer *ui.Viewer
	Map    *alm.Map
	Title  string
}

// LoadMapViewer decodes a map and builds the viewer that draws it.
//
// This is the ONE place an asset root plus a map becomes a running viewer. Both
// entry points call it — the standalone developer viewer and the game front-end
// — which is what makes "the two must not diverge" a property of there being a
// single function rather than a convention someone has to remember. A change to
// how a map becomes a viewer is observable in both or in neither.
//
// The title is the map's own recorded name when it has one, and fallbackTitle
// otherwise. Most campaign maps record no name, so the fallback is the normal
// case rather than the exceptional one; the caller supplies it because only the
// caller knows what the map was called where it came from — a file name for a
// loose map, an entry name for an archived one.
//
// markers selects the diagnostic placement overlays, and they are wired HERE
// rather than in either caller (DD33): the conversion from placement records to
// anchor cells is one function with one contract, so the game and the developer
// viewer cannot mark different cells, and a front-end cannot inherit the load
// path while missing the overlays that go with it. Which overlays to show is
// still the caller's call — the developer tool takes it from -objects/-units,
// the game from -markers — because that is presentation, not loading.
//
// layer is the map's static-object bundle and its art switch, and it is a
// PARAMETER of this one entry point rather than a second one: a
// LoadMapViewerStatics twin would let a caller keep the older, narrower call
// and silently draw no objects, which is the shipped defect the marker
// parameter above already exists to close, reintroduced one story later. Its
// zero value gives back the viewer that shipped before this story, so a
// caller with no bundle says so in the call.
//
// Errors come back unwrapped. Each caller labels them with its own source,
// since this function takes bytes and has no path to name; that is what lets
// the standalone viewer keep printing `decode <path>: <err>` exactly as it
// always has while the front-end says something appropriate to a picker row.
// structures is the map's structure bundle and its art switch, and it is a
// PARAMETER of this same entry point for the reason layer is: a second,
// narrower door is how a caller ends up drawing no buildings while every
// test stays green. Its zero value gives back the viewer that shipped before
// this story, so a caller with no bundle says so in the call.
func LoadMapViewer(tiles *terrain.Tileset, data []byte, fallbackTitle string, markers Markers, layer StaticLayer,
	structures StructureLayer) (*MapView, error) {
	return LoadMapViewerFor("", tiles, data, fallbackTitle, markers, layer, structures)
}

// LoadMapViewerFor is LoadMapViewer reading the map the way game g's files are
// laid out; the empty game is the first.
func LoadMapViewerFor(g base.Game, tiles *terrain.Tileset, data []byte, fallbackTitle string, markers Markers, layer StaticLayer,
	structures StructureLayer) (*MapView, error) {
	open := alm.Open
	if g.Edition().SecondMaps {
		open = alm.OpenROM2
	}
	m, err := open(data)
	if err != nil {
		return nil, err
	}

	title := m.Name
	if title == "" {
		title = fallbackTitle
	}

	// The altitudes ride along with the tile words: they are part of the map,
	// so this one load path hands both to the viewer and the viewer decides
	// what to do with them. Both entry points come through here, so the game
	// front-end gets a displaced terrain by construction rather than by
	// remembering to ask for one.
	//
	// The object grid rides here for exactly that reason too, and it is set
	// UNCONDITIONALLY: it costs a bundle-less load nothing, since
	// StaticPlacements is the field's one reader and it answers an empty list
	// for a nil bundle whatever the grid holds. Gating it on the bundle would
	// make the map layers a viewer sees depend on what the caller loaded rather
	// than on what the map contains. The block plane rides with them for the
	// same reason, and it is the reason this function imports pkg/mapload at
	// all. It is DERIVED rather than decoded, and this is the tier where that
	// can happen: the rule that derives it may not be imported by the viewer's
	// own tier, and the one value it needs from the map is the map this
	// function has just opened. So the margin the viewer dims and the margin
	// the simulation refuses to walk into are one derivation called twice over
	// one decoded map, not two rules kept in agreement — Passability is the
	// same call pkg/mapload's own world builder makes, and there is no second
	// place a depth could drift.
	//
	// It is set UNCONDITIONALLY, as the object grid above is: a viewer decides
	// what to do with a layer, and which layers it sees must depend on what the
	// map contains rather than on what the caller happened to ask for.
	//
	// The type-4 placement records ride with them, and UNCONDITIONALLY for the
	// object grid's reason: they cost a bundle-less load one slice of the map's
	// own records, StructurePlacements is the field's one reader and it answers
	// an empty list for a nil bundle whatever the records hold. Gating them on
	// the bundle would make the map layers a viewer sees depend on what the
	// caller loaded rather than on what the map contains.
	viewer, err := ui.NewViewerWithStatics(title, terrain.Grid{
		Width:      m.Width,
		Height:     m.Height,
		Tiles:      terrain.RenderTileWords(m.Tiles),
		Altitudes:  m.Altitudes,
		Overlay:    m.Overlay,
		Block:      mapload.Passability(m),
		Structures: StructureRecords(m.Objects),
	}, tiles, layer.Set, layer.Art, markers.Statics, layer.AnimGate, structures.Set, structures.Art)
	if err != nil {
		return nil, err
	}

	// Cells are derived once and handed over per overlay. An overlay that was
	// not asked for is left at its zero state rather than set to "off with
	// cells", so a viewer built with Markers{} is the viewer that shipped
	// before this parameter existed.
	//
	// The static-object cross is deliberately absent from this block: it was
	// passed to the constructor above and it needs no cells from here, because
	// the cells it marks are the ones the viewer's own placement list resolved.
	// MarkerCells is left with the two record-derived overlays it has always
	// had.
	if markers.Objects || markers.Units {
		objects, units := MarkerCells(m)
		if markers.Objects {
			viewer.SetObjects(true, objects)
		}
		if markers.Units {
			viewer.SetUnits(true, units)
		}
	}
	return &MapView{Viewer: viewer, Map: m, Title: title}, nil
}
