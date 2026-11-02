package ui

// Palette is the bar palette a placeable record lists in: a structure kind
// of the Add window's kind step or an object class of the object brush. The
// host decides it from the game's class data; each record lists in exactly
// one palette. PaletteOwn is the palette of the record's own kind:
// Structures for a structure kind, Objects for an object class.
type Palette uint8

const (
	PaletteOwn        Palette = iota
	PaletteStructures         // what can be selected, destroyed or switched, bridges, and what the data does not settle
	PaletteObjects            // decoration, no trees
	PaletteTrees              // trees
)

// in is the palette p names for a record whose own kind lists in own.
func (p Palette) in(own Palette) Palette {
	if p == PaletteOwn {
		return own
	}
	return p
}

// tree reports whether object class o lists in the Trees palette.
func (o BrushObject) tree() bool {
	return o.Code != 0 && o.Palette.in(PaletteObjects) == PaletteTrees
}

// brushObjectRows are the object brush's rows in its mode: the clear row,
// then the tree classes in the Trees palette's window (Forest mode, or its
// one-kind mode) and every other class in the Objects palette's one-kind mode.
func (e *MapEditor) brushObjectRows() []BrushObject {
	return e.objectRowsIn(e.brush.forest.on || e.brush.forest.trees)
}

// objectRowsIn are the clear row and the object classes of the Trees
// palette, or of the Objects palette when trees is false.
func (e *MapEditor) objectRowsIn(trees bool) []BrushObject {
	h := e.brushHost()
	if h == nil {
		return nil
	}
	var out []BrushObject
	for _, o := range h.BrushObjects() {
		if o.Code == 0 || o.tree() == trees {
			out = append(out, o)
		}
	}
	return out
}

// PaletteHolds is what bar palette p lists on the open document, as its
// window shows it: the values of its structure kinds and the codes of its
// object classes. A headless driver reads it; PaletteOwn names no palette.
func (e *MapEditor) PaletteHolds(p Palette) (kinds []int64, objects []int) {
	if p == PaletteOwn {
		return nil, nil
	}
	if p != PaletteTrees {
		if _, ok := e.addHost(); ok {
			for _, c := range e.placeIn(KindStructure, "kind", p) {
				kinds = append(kinds, c.Value)
			}
		}
	}
	if p != PaletteStructures {
		for _, o := range e.objectRowsIn(p == PaletteTrees) {
			if o.Code != 0 {
				objects = append(objects, o.Code)
			}
		}
	}
	return kinds, objects
}

// emptyPaletteNote is the plain words of a palette that holds no record in
// one of its categories, empty when it holds one in each: Structures with no
// structure kind; Objects with no decoration object class or no decoration
// structure kind; Trees with no tree class.
func (e *MapEditor) emptyPaletteNote() []string {
	var out []string
	switch {
	case e.listKind() == "records" && e.filter == 2:
		if _, ok := e.addHost(); ok && len(e.placeShown()) == 0 {
			out = append(out, e.word("editor.palette.nostructures"))
		}
	case e.forestOn():
		if len(e.forestKinds()) == 0 {
			out = append(out, e.word("editor.palette.notrees"))
		}
	case e.listKind() == "brush" && e.brush.mode == brushObjects && e.brush.forest.trees:
		if len(e.brushObjectRows()) < 2 {
			out = append(out, e.word("editor.palette.notrees"))
		}
	case e.listKind() == "brush" && e.brush.mode == brushObjects:
		if len(e.brushObjectRows()) < 2 {
			out = append(out, e.word("editor.palette.noobjects"))
		}
		if _, ok := e.addHost(); ok && len(e.placeShown()) == 0 {
			out = append(out, e.word("editor.palette.nodecoration"))
		}
	}
	return out
}
