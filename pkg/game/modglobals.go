package game

import (
	"fmt"
	"maps"
	"slices"

	"againrom/pkg/mapload"
	"againrom/pkg/mod"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/sim"
)

// modGlobalStore is the campaign's values of the mod globals, keyed by the
// mod that declares them, each list ascending by id (DIV-5108). It lives on
// the Town for one campaign run: a won mission's World gives its mod's values
// back, and a mission's entry seats them; a global with no value here is at
// its initial. The store is replaced whole and never edited in place, so a
// copied Town keeps its own.
type modGlobalStore map[string][]sim.ScriptGlobal

// modGlobals is the town's store; nil for no town.
func (t *Town) modGlobals() modGlobalStore {
	if t == nil {
		return nil
	}
	return t.globals
}

// with is the store with modID's values replaced by values; no values leaves
// the store as it is.
func (s modGlobalStore) with(modID string, values []sim.ScriptGlobal) modGlobalStore {
	if len(values) == 0 {
		return s
	}
	out := maps.Clone(s)
	if out == nil {
		out = modGlobalStore{}
	}
	out[modID] = slices.Clone(values)
	return out
}

// admitted is the store a LOAD keeps under the active mods: an active mod
// keeps the ids it declares now and drops the others, and a mod that is not
// active keeps its values untouched, so the next SAVE writes them back.
func (s modGlobalStore) admitted(ctx mapload.ModContext) modGlobalStore {
	var out modGlobalStore
	for _, id := range slices.Sorted(maps.Keys(s)) {
		values := s[id]
		if slices.ContainsFunc(ctx.Set.Mods, func(m mod.SetEntry) bool { return m.ID == id }) {
			values = slices.DeleteFunc(slices.Clone(values), func(g sim.ScriptGlobal) bool { return !ctx.Globals.Declares(id, g.ID) })
		}
		if len(values) == 0 {
			continue
		}
		if out == nil {
			out = modGlobalStore{}
		}
		out[id] = values
	}
	return out
}

// modMarkGlobal is one global's value in the mod mark.
type modMarkGlobal struct {
	ID    int32 `json:"id"`
	Value int32 `json:"value"`
}

// mark is the store in the mod mark's form; nil for an empty store, so a
// mark without globals keeps its bytes.
func (s modGlobalStore) mark() map[string][]modMarkGlobal {
	if len(s) == 0 {
		return nil
	}
	out := make(map[string][]modMarkGlobal, len(s))
	for id, values := range s {
		marked := make([]modMarkGlobal, len(values))
		for i, g := range values {
			marked[i] = modMarkGlobal{ID: g.ID, Value: g.Value}
		}
		out[id] = marked
	}
	return out
}

// markedGlobals reads the mod mark's globals back into a store. A mod key
// that is empty, a list longer than a mod may declare, or an id outside the
// global range or out of ascending order is refused.
func markedGlobals(m map[string][]modMarkGlobal) (modGlobalStore, error) {
	if len(m) == 0 {
		return nil, nil
	}
	out := make(modGlobalStore, len(m))
	for id, marked := range m {
		if id == "" || len(marked) == 0 || len(marked) > scriptprofile.LastGlobal {
			return nil, fmt.Errorf("the mod mark holds %d globals under mod %q", len(marked), id)
		}
		values := make([]sim.ScriptGlobal, len(marked))
		prior := int32(scriptprofile.FirstGlobal - 1)
		for i, g := range marked {
			if g.ID <= prior || g.ID > scriptprofile.LastGlobal {
				return nil, fmt.Errorf("the mod mark's global %d of mod %q is outside %d..%d or out of order",
					g.ID, id, scriptprofile.FirstGlobal, scriptprofile.LastGlobal)
			}
			prior = g.ID
			values[i] = sim.ScriptGlobal{ID: g.ID, Value: g.Value}
		}
		out[id] = values
	}
	return out, nil
}
