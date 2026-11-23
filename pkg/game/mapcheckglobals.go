package game

import (
	"fmt"
	"slices"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mod"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// modMapGlobals is what the global rule reads of a mod's mission: the mod,
// the ids it declares and the mission's dialogue files, whose replies name
// script nodes too.
type modMapGlobals struct {
	mod       string
	declared  mod.GlobalData
	dialogues map[string][]byte
}

// modMapGlobals is the global rule's view of mission m, nil when the
// inspector's table does not run m's mod and so cannot say what it
// declares; the launch and the staged package's admission run it then.
func (x *MapInspector) modMapGlobals(m mod.Mission) *modMapGlobals {
	if x == nil || x.table == nil || m.Mod == "" {
		return nil
	}
	mods := x.table.Mods
	if !slices.ContainsFunc(mods.Set.Mods, func(e mod.SetEntry) bool { return e.ID == m.Mod }) {
		return nil
	}
	return &modMapGlobals{mod: m.Mod, declared: mods.Globals, dialogues: m.DialogueFiles}
}

// globalUndeclared: DIV-5108. A node naming a global its mod does not
// declare reads 0 and writes nothing; the check refuses it at each trigger
// and reply that names the node, and at the node when none does.
func (c *mapCheck) globalUndeclared() []mapHit {
	g := c.globals
	if g == nil || c.scriptErr != nil {
		return nil
	}
	undeclared := func(condition bool, nodes []alm.ScriptNode) map[uint32]int32 {
		out := map[uint32]int32{}
		for _, n := range nodes {
			if id := int32(n.Value[0]); scriptprofile.AddressesGlobal(condition, n.Opcode) && !g.declared.Declares(g.mod, id) {
				out[n.ID] = id
			}
		}
		return out
	}
	checks, actions := undeclared(true, c.script.Conditions), undeclared(false, c.script.Actions)
	if len(checks) == 0 && len(actions) == 0 {
		return nil
	}
	type nodeKey struct {
		condition bool
		id        uint32
	}
	used := map[nodeKey]bool{}
	name := func(condition bool, node uint32) (int32, bool) {
		table := actions
		if condition {
			table = checks
		}
		id, ok := table[node]
		if ok {
			used[nodeKey{condition, node}] = true
		}
		return id, ok
	}
	var out []mapHit
	for i, t := range c.script.Triggers {
		var ids []int32
		for k := range t.Pairs() {
			left, right, _ := t.Pair(k)
			for _, node := range []uint32{left, right} {
				if id, ok := name(true, node); ok && !slices.Contains(ids, id) {
					ids = append(ids, id)
				}
			}
		}
		for _, node := range t.Acts {
			if id, ok := name(false, node); ok && !slices.Contains(ids, id) {
				ids = append(ids, id)
			}
		}
		for _, id := range ids {
			out = append(out, mapHit{Kind: ui.KindTrigger, Index: i, Text: fmt.Sprintf(c.x.word("editor.message.globaltrigger"), id, g.mod)})
		}
	}
	files := make([]string, 0, len(g.dialogues))
	for file := range g.dialogues {
		files = append(files, file)
	}
	slices.Sort(files)
	for _, file := range files {
		d, err := mod.ParseDialogue(file, g.dialogues[file])
		if err != nil {
			continue
		}
		for _, n := range d.Nodes {
			for r, reply := range n.Replies {
				var ids []int32
				for _, w := range reply.When {
					for _, node := range []int{w.Left, w.Right} {
						if id, ok := name(true, uint32(node)); ok && !slices.Contains(ids, id) {
							ids = append(ids, id)
						}
					}
				}
				for _, node := range reply.Do {
					if id, ok := name(false, uint32(node)); ok && !slices.Contains(ids, id) {
						ids = append(ids, id)
					}
				}
				for _, id := range ids {
					out = append(out, mapHit{Text: fmt.Sprintf(c.x.word("editor.message.globalreply"), file, n.ID, r+1, id, g.mod)})
				}
			}
		}
	}
	for _, side := range []struct {
		condition bool
		kind      ui.RecordKind
		nodes     []alm.ScriptNode
		table     map[uint32]int32
	}{{true, ui.KindCondition, c.script.Conditions, checks}, {false, ui.KindAction, c.script.Actions, actions}} {
		for i, n := range side.nodes {
			id, ok := side.table[n.ID]
			if ok && !used[nodeKey{side.condition, n.ID}] {
				out = append(out, mapHit{Kind: side.kind, Index: i, Text: fmt.Sprintf(c.x.word("editor.message.globalnode"), id, g.mod)})
			}
		}
	}
	return out
}
