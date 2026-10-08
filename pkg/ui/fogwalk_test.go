package ui

// THE FOG GATE'S OWN STRUCTURAL FENCE (hotfix).
//
// The defect this file exists to prevent is not that four health bars leaked;
// it is WHY they could. fogGateEntity lives at a call site rather than at the
// snapshot, so every walk over v.entities was free to ignore it, and five of
// the seven did — silently, because a leak draws pixels rather than errors.
// entityLayer's own doc already argues that gating in one place keeps the
// sprite half and the square half from disagreeing; the same reasoning one
// tier up says every consumer of the snapshot in a DRAW path owes the gate.
//
// So this scan reads the package's own non-test source, finds every function
// that touches v.entities, and requires each one to appear in the table below
// with a verdict. A new walk fails here, by name, on the next `go test` — and
// its author has to say which kind it is rather than discovering a year later
// that it leaked.
//
// A "gated" verdict is CHECKED, NOT TAKEN: the scan requires such a function
// to call fogGateEntity too, so the table cannot claim a gate the code does
// not have.
//
// WHAT IT CANNOT DO, stated plainly rather than hoped over. It is LEXICAL, on
// internal/archtest's own determinism-scan precedent: it proves which
// functions name the field, not that a leak cannot arrive another way. Two
// gaps are known and left open on purpose. A function reaching the snapshot
// through a HELPER — numeralPlacements does, through entityByID — is invisible
// here, and is gated at the caller instead; a new caller of such a helper is
// not caught. And a "gated" function that asks fogGateEntity about the wrong
// entity, or on only one of two branches, passes: the scan sees the call, not
// its argument. Both are why this is a fence and not a proof.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"strings"
	"testing"
)

type fogWalkVerdict int

const (
	// fogWalkGated: the function turns snapshot entries into pixels, and asks
	// fogGateEntity before it does.
	fogWalkGated fogWalkVerdict = iota
	// fogWalkNotADraw: the function produces no map pixels from the snapshot
	// — it writes it, counts it, looks one entry up, or decides an input.
	// Each entry below says which, and why that is not a leak.
	fogWalkNotADraw
)

// fogWalkTable is every function in this package that names v.entities, with
// what it does with it. IT IS THE WHOLE CONTRACT: a function missing from it
// fails the scan, and so does one whose verdict the code contradicts.
var fogWalkTable = map[string]struct {
	verdict fogWalkVerdict
	why     string
}{
	// --- draws, gated ---
	"entityLayer":          {fogWalkGated, "the sprite and the square; the gate's original and only call site"},
	"inspectionPanel":      {fogWalkGated, "read-only hover stats use the live unit visibility gate"},
	"Inspection":           {fogWalkNotADraw, "tagged subject lookup; hover is gated by the draw list and the selected fallback by visiblePanelSelection"},
	"statusBars":           {fogWalkGated, "the health and mage mana bars — the health bar was the leak the owner reported"},
	"minimapMarks":         {fogWalkGated, "the minimap dot"},
	"shotScreenRects":      {fogWalkGated, "the shot mark"},
	"ingestDamage":         {fogWalkGated, "the damage numeral's creation half"},
	"stepSound":            {fogWalkGated, "the blow grunt -- the numeral's other half, and a positional sound carries the direction too"},
	"attackTargetRect":     {fogWalkGated, "the attack cursor's outline; the PRESS behind it is deliberately not gated, see cursor.go"},
	"selectionScreenRects": {fogWalkGated, "the selection rim over a selected enemy that walked into the dark"},
	"pathScreenSegments":   {fogWalkGated, "the ordered route of a selected unit — where he is AND where he is going"},
	"spellEffectPasses":    {fogWalkGated, "the spell effect ring: a spell landing on a unit in the dark shows nothing, on the selection rim's own terms"},
	"topEntityAtCell": {fogWalkGated,
		"the minimap's own cell hit test (story 1034, `AI-MINIMAP-124`): it names which entity a minimap press attacks, and an enemy in the dark is not one -- the widget does not draw him either (minimapMarks, above)"},
	"castVictimAt": {fogWalkGated,
		"the unit a cast cursor may stand over: it asks fogGateEntity before it reports a hit, so a unit in the fog changes no cursor and no click arm"},
	"spellCasterHolds": {fogWalkNotADraw,
		"filters the already-selected units down to those that cast the selected spell; the selection is units the player holds and sees, and it emits no map pixel"},
	"hoverMask": {fogWalkGated,
		"the mission map's own hover hit test (story 1034): it asks fogGateEntity before it reports a hit at all, so a unit in the fog changes no cursor and no click arm -- the gate hoverHostilityCursor carried, moved to the one function that now names the snapshot for every cursor arm"},

	// --- no map pixels ---
	"beginSoundEntry":        {fogWalkNotADraw, "queues stage-one audio subjects; playHurt applies the existing visibility and placement gates when consumed"},
	"RestoreSaveApplication": {fogWalkNotADraw, "validates saved selection identities before restoring viewer state; emits no map pixels"},
	"SelectedUnits":          {fogWalkNotADraw, "returns the already-selected accepted ID population for book composition; emits no map pixels"},
	"quickSpell":             {fogWalkNotADraw, "checks selected ownership before arming Cast; emits no map pixels"},
	"issuePlayerRetreat":     {fogWalkNotADraw, "checks the owned selection and queues immediate Retreat; draws no map pixels"},
	"ArmItemCast":            {fogWalkNotADraw, "validates the owned selected actor; emits no map pixels"},
	"releaseItemCast":        {fogWalkNotADraw, "dispatches a target from the fog-gated hoverMask; emits no map pixels"},
	"HeadlessEntityPoint":    {fogWalkNotADraw, "read-only test point lookup through the production pick rectangle; sends no event or pixel"},
	"selectionSummary": {fogWalkNotADraw,
		"this build's `view+0x144` (story 1034): a flags word derived from the SELECTION -- units the player already holds and can already see. It emits no map pixel and reports nothing about an unselected entity, so there is no unit in the fog for it to reveal"},
	"missionHoverCursor": {fogWalkNotADraw,
		"which cursor picture the pointer wears (story 1034). Every entity fact it reads comes through hoverMask, which is gated above; this function itself only names a cursor"},
	"gestureCursorAt": {fogWalkNotADraw,
		"the cursor a map CLICK was made under, which is what turns it into an order (story 1034). Its entity facts come through hoverMask and through the selection, both gated; it emits no map pixel. That a click may TARGET a unit the fog hides is command's own question, named there and in the hotfix ledger"},
	"selectAllOwnedUnits": {fogWalkNotADraw,
		"the E key's whole-army selection (story 1034). It takes OWNED units only, so every entity it can reach is one the local participant owns and therefore sees; it emits no map pixel and names no unit on screen. A selection is not a draw -- what the selection overlay paints is the overlay's own question, gated where it draws"},
	"groupKey": {fogWalkNotADraw,
		"the digit row's group assign and recall (story 1034). It reads the snapshot only to filter a stored id list down to ids still present, and to average the recalled members' own cells for the Alt centring. Every id in a group was put there by an owned selection, so there is no unit in the fog for it to reveal, and it emits no map pixel"},
	"SetEntities": {fogWalkNotADraw, "the writer of the snapshot; there is nothing yet to gate"},
	"EntityMarkers": {fogWalkNotADraw,
		"a COUNT for the debug readout, not a per-unit glyph; it reveals no cell and names no unit"},
	"entityByID": {fogWalkNotADraw,
		"a lookup helper; its one caller, numeralPlacements, carries the gate — a SECOND caller would not be caught here"},
	"marked": {fogWalkNotADraw,
		"the ids the two debug keys act on; an input set, and the debug keys are a developer's surface"},
	"command": {fogWalkNotADraw,
		"the press's own decision. Gating input would change what the player can TARGET rather than what he can see, and whether the original refuses that is not decoded — a story's question, named in the hotfix ledger"},
	"armAttack": {fogWalkNotADraw, "whether the attack cursor may arm at all; input, on command's own terms"},
	"armDefend": {fogWalkNotADraw, "owned-selection input gate for Defend; emits no map pixel and names no unselected actor"},
	"toggleAutocastAt": {fogWalkNotADraw,
		"the autocast key's own decision. It emits no map pixel and reveals no cell -- it reads the selection's ownership through canArmAttack, which is armAttack's own gate, and writes a setting on a unit the player already has selected"},
	"inventoryEligible": {fogWalkNotADraw,
		"whether the inventory window may open for the selection — a PANEL rather than a map glyph, and blanking a panel mid-inspection is a rule this hotfix does not make; named in the hotfix ledger as a standing exposure"},
	"dollInventoryActive": {fogWalkNotADraw,
		"whether invisible inventory rectangles reserve input under the selected subject's HUD doll; it emits no map pixel"},
	"dollSubject":       {fogWalkNotADraw, "the selected unit's HUD picture source; it reveals no map cell"},
	"panelSubject":      {fogWalkNotADraw, "the info panel's subject, on inventoryEligible's own terms"},
	"characterPaneView": {fogWalkNotADraw, "counts the presented selection for fixed character-pane text and emits no map-position pixel"},
	"readoutSubjectOf":  {fogWalkNotADraw, "the debug readout's counts and its selected-unit line; a developer's box, off by default"},
	"SelectedUnit": {fogWalkNotADraw,
		"the entity id the panel and the spellbook both describe -- an id lookup, not a glyph; pkg/game's per-tick push is its one caller across the seam"},
	"headlessSelectEntityOnce": {fogWalkNotADraw,
		"a production hit-test input adapter for a no-window scenario; it selects through command and emits no map pixels"},
	"HeadlessDollSlotPoint": {fogWalkNotADraw,
		"a scenario's doll-slot pixel oracle; it counts the selected entities only to say WHY a frame drew no doll box, and emits no map pixel"},
	"composeCommandPanel": {fogWalkNotADraw,
		"whether the command panel composes active or its head-only picture, through commandPanelActive -- armAttack's own ownership check restated. It draws a fixed UI panel, not a per-entity map glyph, and reveals no cell"},
	"pressCommandPanelCell": {fogWalkNotADraw,
		"whether a panel press may arm or issue at all, on commandPanelActive/armAttack's own terms; input, not a map pixel"},
	"castKey": {fogWalkNotADraw,
		"whether the C key may arm Cast mode for the selection, on armAttack's own ownership terms; input, not a map pixel"},
	"commandHoverPresent": {fogWalkNotADraw,
		"whether the hovered cell's label may show at all, on commandPanelActive's own terms; a tooltip over a fixed UI panel, not a map glyph, and reveals no cell"},
	"tooltipTarget": {fogWalkNotADraw, "resolves fixed HUD help under the same command-panel ownership gate; does not emit actor glyphs or reveal map cells"},
}

// TestEveryWalkOverTheEntitySnapshotIsClassified is the fence itself.
func TestEveryWalkOverTheEntitySnapshotIsClassified(t *testing.T) {
	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", func(fi fs.FileInfo) bool {
		return !strings.HasSuffix(fi.Name(), "_test.go")
	}, 0)
	if err != nil {
		t.Fatalf("parsing this package's own source: %v", err)
	}
	pkg, ok := pkgs["ui"]
	if !ok {
		t.Fatal(`no package "ui" parsed out of "."`)
	}

	found := map[string]bool{} // function name -> calls fogGateEntity
	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			walks, gates := false, false
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch e := n.(type) {
				case *ast.SelectorExpr:
					if id, ok := e.X.(*ast.Ident); ok && id.Name == "v" && e.Sel.Name == "entities" {
						walks = true
					}
				case *ast.CallExpr:
					if sel, ok := e.Fun.(*ast.SelectorExpr); ok && sel.Sel.Name == "fogGateEntity" {
						gates = true
					}
				}
				return true
			})
			if walks {
				found[fn.Name.Name] = found[fn.Name.Name] || gates
			}
		}
	}

	for name, gates := range found {
		entry, known := fogWalkTable[name]
		if !known {
			t.Errorf("%s walks the entity snapshot and is not in fogWalkTable.\n"+
				"Decide what it is and say so there: does it turn entries into MAP PIXELS?\n"+
				"If it does, it must call fogGateEntity — an enemy in the fog shows NO indicator\n"+
				"(owner, 2026-08-08). If it does not, add it with fogWalkNotADraw and say why.", name)
			continue
		}
		if entry.verdict == fogWalkGated && !gates {
			t.Errorf("%s is declared gated in fogWalkTable (%q) but calls no fogGateEntity",
				name, entry.why)
		}
	}
	for name := range fogWalkTable {
		if _, ok := found[name]; !ok {
			t.Errorf("fogWalkTable names %s, which no longer walks the entity snapshot — "+
				"delete the row so the table stays the whole truth", name)
		}
	}
}
