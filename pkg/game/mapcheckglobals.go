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

// globalUse is one place a script names a global: a trigger (Trigger >= 0)
// or reply Reply of node Node in dialogue file File.
type globalUse struct {
	ID      int32
	Trigger int
	File    string
	Node    int
	Reply   int
}

// globalNode is a global node no trigger and no reply names.
type globalNode struct {
	Condition bool
	Index     int
	ID        int32
}

// scriptGlobalUses is every global the script's triggers and the dialogues'
// replies name, each place once per global, and the global nodes none of
// them names. It is the one reader of what names a global: the map check
// and the Globals page's users ask it.
func scriptGlobalUses(script alm.Script, dialogues map[string][]byte) (uses []globalUse, loose []globalNode) {
	table := func(condition bool, nodes []alm.ScriptNode) map[uint32]int32 {
		out := map[uint32]int32{}
		for _, n := range nodes {
			if scriptprofile.AddressesGlobal(condition, n.Opcode) {
				out[n.ID] = int32(n.Value[0])
			}
		}
		return out
	}
	checks, actions := table(true, script.Conditions), table(false, script.Actions)
	if len(checks) == 0 && len(actions) == 0 {
		return nil, nil
	}
	type nodeKey struct {
		condition bool
		id        uint32
	}
	named := map[nodeKey]bool{}
	name := func(condition bool, node uint32) (int32, bool) {
		t := actions
		if condition {
			t = checks
		}
		id, ok := t[node]
		if ok {
			named[nodeKey{condition, node}] = true
		}
		return id, ok
	}
	for i, t := range script.Triggers {
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
			uses = append(uses, globalUse{ID: id, Trigger: i})
		}
	}
	files := make([]string, 0, len(dialogues))
	for file := range dialogues {
		files = append(files, file)
	}
	slices.Sort(files)
	for _, file := range files {
		d, err := mod.ParseDialogue(file, dialogues[file])
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
					uses = append(uses, globalUse{ID: id, Trigger: -1, File: file, Node: n.ID, Reply: r + 1})
				}
			}
		}
	}
	for _, side := range []struct {
		condition bool
		nodes     []alm.ScriptNode
		table     map[uint32]int32
	}{{true, script.Conditions, checks}, {false, script.Actions, actions}} {
		for i, n := range side.nodes {
			if id, ok := side.table[n.ID]; ok && !named[nodeKey{side.condition, n.ID}] {
				loose = append(loose, globalNode{side.condition, i, id})
			}
		}
	}
	return uses, loose
}

// globalUndeclared: DIV-5108. A node naming a global its mod does not
// declare reads 0 and writes nothing; the check refuses it at each trigger
// and reply that names the node, and at the node when none does.
func (c *mapCheck) globalUndeclared() []mapHit {
	g := c.globals
	if g == nil || c.scriptErr != nil {
		return nil
	}
	uses, loose := scriptGlobalUses(c.script, g.dialogues)
	var out []mapHit
	for _, u := range uses {
		switch {
		case g.declared.Declares(g.mod, u.ID):
		case u.Trigger >= 0:
			out = append(out, mapHit{Kind: ui.KindTrigger, Index: u.Trigger, Text: fmt.Sprintf(c.x.word("editor.message.globaltrigger"), u.ID, g.mod)})
		default:
			out = append(out, mapHit{Text: fmt.Sprintf(c.x.word("editor.message.globalreply"), u.File, u.Node, u.Reply, u.ID, g.mod)})
		}
	}
	for _, n := range loose {
		if g.declared.Declares(g.mod, n.ID) {
			continue
		}
		kind := ui.KindAction
		if n.Condition {
			kind = ui.KindCondition
		}
		out = append(out, mapHit{Kind: kind, Index: n.Index, Text: fmt.Sprintf(c.x.word("editor.message.globalnode"), n.ID, g.mod)})
	}
	return out
}
