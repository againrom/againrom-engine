package game

import (
	"bytes"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/scriptprofile"
	"againrom/pkg/ui"
)

// The map's interactive dialogues. The workspace holds each dialogue file's
// bytes; the project's drafts and every save follow it as they follow the
// labels. A reply's condition and outcome are nodes of the map's script that
// no trigger names: the dialogue file names them, so a write keeps them and
// a rewrite deletes the ones it leaves unnamed.

const maxDialogueFileBytes = 1 << 20

// dialogueFiles is the live dialogue files of w by name.
func (w *editorWorkspaceState) dialogueFiles() map[string][]byte {
	if w == nil {
		return nil
	}
	var out map[string][]byte
	for name, b := range w.Dialogues {
		if b == nil {
			continue
		}
		if out == nil {
			out = map[string][]byte{}
		}
		out[name] = bytes.Clone(b)
	}
	return out
}

// modDiskBytes is the file rel of the open mod's folder, nil when there is none.
func (s *mapInspectionEdits) modDiskBytes(rel string) []byte {
	if s.inspector == nil || s.inspector.mod == nil {
		return nil
	}
	b, _, err := mod.ReadInside(s.inspector.mod.dir, rel)
	if err != nil {
		return nil
	}
	return b
}

// syncDialogueDrafts puts each dialogue of dialogues into the project's
// drafts under folder, a draft that equals the file in the mod folder left
// out, and a removed dialogue as a removal when the mod folder holds it.
func (s *mapInspectionEdits) syncDialogueDrafts(d *editorModDocument, folder string, dialogues map[string][]byte) {
	for name, b := range dialogues {
		rel := path.Join(folder, name+".toml")
		disk := s.modDiskBytes(rel)
		switch {
		case b == nil && disk != nil:
			d.Files[rel] = nil
		case b == nil:
			delete(d.Files, rel)
		case bytes.Equal(disk, b):
			delete(d.Files, rel)
		case !bytes.Equal(d.Files[rel], b):
			d.Files[rel] = bytes.Clone(b)
		}
	}
}

// openDialogues reads the dialogue files beside the open map into w: the
// folder next to a map file on the host, then the project's drafts over it.
func (s *mapInspectionEdits) openDialogues(w *editorWorkspaceState) {
	x := s.inspector
	var folder string
	if p, ok := mapHostPath(ui.ParseMapSource(s.source), x.archives); ok && s.source != "" {
		folder = mod.MissionPlace{Map: p}.Dialogs()
	}
	if folder != "" {
		if entries, err := os.ReadDir(folder); err == nil {
			for _, e := range entries {
				name, isTOML := strings.CutSuffix(e.Name(), ".toml")
				if e.IsDir() || !isTOML || !mod.ValidDialogueName(name) {
					continue
				}
				if info, err := e.Info(); err != nil || info.Size() > maxDialogueFileBytes {
					continue
				}
				if b, err := os.ReadFile(filepath.Join(folder, e.Name())); err == nil {
					if w.Dialogues == nil {
						w.Dialogues = map[string][]byte{}
					}
					w.Dialogues[name] = b
				}
			}
		}
	}
	d := w.Project
	if d == nil || d.Selected < 0 || d.Selected >= len(d.Missions) {
		return
	}
	prefix := mod.MissionPlace{Map: d.Missions[d.Selected].Map}.Dialogs() + "/"
	for rel, b := range d.Files {
		name, isTOML := strings.CutSuffix(strings.TrimPrefix(rel, prefix), ".toml")
		if !strings.HasPrefix(rel, prefix) || !isTOML || !mod.ValidDialogueName(name) {
			continue
		}
		if w.Dialogues == nil {
			w.Dialogues = map[string][]byte{}
		}
		w.Dialogues[name] = bytes.Clone(b)
	}
}

// dialogueHeld is the dialogues a read or a write sees: the stage's while a
// change is under way, else the workspace's.
func (s *mapInspectionEdits) dialogueHeld() map[string][]byte {
	if s.stage != nil && s.stage.next != nil {
		return s.stage.next.Dialogues
	}
	if s.prepareWorkspace() != nil || s.workspace == nil {
		return nil
	}
	return s.workspace.Dialogues
}

// DialogueNames are the dialogues of the map, in name order.
func (s *mapInspectionEdits) DialogueNames() []string {
	var out []string
	for name, b := range s.dialogueHeld() {
		if b != nil {
			out = append(out, name)
		}
	}
	slices.Sort(out)
	return out
}

// ReadDialogue is dialogue name; every node a reply names is kept as it stands.
func (s *mapInspectionEdits) ReadDialogue(name string) (ui.DialogueData, error) {
	b := s.dialogueHeld()[name]
	if b == nil {
		return ui.DialogueData{}, wordErr("editor.dialogue.err.missing", name)
	}
	f, err := mod.ParseDialogue(name+".toml", b)
	if err != nil {
		return ui.DialogueData{}, err
	}
	var out ui.DialogueData
	for _, n := range f.Nodes {
		node := ui.DialogueNodeData{ID: n.ID, Name: n.Name, Text: maps.Clone(n.Text), Speaks: n.Speaks, Speaker: n.Speaker, Next: n.Next}
		for k, v := range n.Variants {
			if node.Variants == nil {
				node.Variants = map[string]map[string]string{}
			}
			node.Variants[k] = maps.Clone(v)
		}
		for _, r := range n.Replies {
			reply := ui.DialogueReplyData{Text: maps.Clone(r.Text), Next: r.Next, Once: r.Once, Mark: r.Mark, Disabled: r.Disabled}
			for _, c := range r.When {
				reply.Pairs = append(reply.Pairs, ui.WizardPair{Left: ui.WizardNode{ID: uint32(c.Left), Keep: true},
					Right: ui.WizardNode{ID: uint32(c.Right), Keep: true}, Cmp: uint32(c.Cmp)})
			}
			for _, id := range r.Do {
				reply.Actions = append(reply.Actions, ui.WizardNode{ID: uint32(id), Keep: true})
			}
			node.Replies = append(node.Replies, reply)
		}
		out.Nodes = append(out.Nodes, node)
	}
	return out, nil
}

// dialogueRefs counts the node ids the dialogue files name: conditions by
// the pairs, actions by the outcomes.
func dialogueRefs(files map[string][]byte, skip string) (conds, acts map[uint32]int) {
	conds, acts = map[uint32]int{}, map[uint32]int{}
	for name, b := range files {
		if name == skip || b == nil {
			continue
		}
		f, err := mod.ParseDialogue(name+".toml", b)
		if err != nil {
			continue
		}
		for _, n := range f.Nodes {
			for _, r := range n.Replies {
				for _, c := range r.When {
					conds[uint32(c.Left)]++
					conds[uint32(c.Right)]++
				}
				for _, id := range r.Do {
					acts[uint32(id)]++
				}
			}
		}
	}
	return conds, acts
}

// dialogueGone are the nodes file f names.
func dialogueGone(f mod.Dialogue) []goneNode {
	var out []goneNode
	for _, n := range f.Nodes {
		for _, r := range n.Replies {
			for _, c := range r.When {
				out = append(out, goneNode{true, uint32(c.Left)}, goneNode{true, uint32(c.Right)})
			}
			for _, id := range r.Do {
				out = append(out, goneNode{false, uint32(id)})
			}
		}
	}
	return out
}

// dialogueVariables is the variable count a once mark must be below.
func dialogueVariables(m *mapedit.Editor) int {
	if m.Extended() {
		return m.ScriptVariables()
	}
	return scriptprofile.FirstRegisters
}

// toFile is d as a dialogue file with ids for its nodes' references already
// written: pairs and actions are the ids their nodes have now.
func toDialogue(name string, d ui.DialogueData, pairs [][][]mod.DialogueCondition, outs [][][]int) mod.Dialogue {
	f := mod.Dialogue{File: name + ".toml"}
	for i, n := range d.Nodes {
		node := mod.DialogueNode{ID: n.ID, Name: n.Name, Text: mod.DialogueText(maps.Clone(n.Text)), Speaks: n.Speaks, Speaker: n.Speaker, Next: n.Next}
		for k, v := range n.Variants {
			if node.Variants == nil {
				node.Variants = map[string]mod.DialogueText{}
			}
			node.Variants[k] = mod.DialogueText(maps.Clone(v))
		}
		for k, r := range n.Replies {
			reply := mod.DialogueReply{Text: mod.DialogueText(maps.Clone(r.Text)), Next: r.Next, Once: r.Once, Mark: r.Mark, Disabled: r.Disabled}
			if pairs != nil {
				reply.When, reply.Do = pairs[i][k], outs[i][k]
			}
			node.Replies = append(node.Replies, reply)
		}
		f.Nodes = append(f.Nodes, node)
	}
	return f
}

// dialogueIssueWords is the word key of each check's refusal.
var dialogueIssueWords = map[mod.DialogueRule]string{
	mod.RuleNoNode:      "editor.dialogue.check.nonode",
	mod.RuleReplies:     "editor.dialogue.check.five",
	mod.RuleNextMissing: "editor.dialogue.check.next",
	mod.RuleNameEmpty:   "editor.dialogue.check.nameempty",
	mod.RuleNameTwice:   "editor.dialogue.check.nametwice",
	mod.RuleNoText:      "editor.dialogue.check.notext",
	mod.RuleMarkPast:    "editor.dialogue.check.mark",
}

// DialogueIssues are the file checks of d against this map.
func (s *mapInspectionEdits) DialogueIssues(d ui.DialogueData) []ui.DialogueIssue {
	variables := 0
	if s.model != nil {
		variables = dialogueVariables(s.model)
	}
	var out []ui.DialogueIssue
	f := toDialogue("", d, nil, nil)
	for _, i := range mod.CheckDialogue(f, variables, f.NamedNodes()) {
		out = append(out, ui.DialogueIssue{Rule: dialogueIssueWords[i.Rule], Node: i.Node, Reply: i.Reply, Other: i.Other})
	}
	return out
}

// FreeMark is the highest script variable below the variable count that no
// node addresses, no check result takes and no dialogue reply marks, -1 when
// none is left.
func (s *mapInspectionEdits) FreeMark(d ui.DialogueData) int {
	if s.model == nil {
		return -1
	}
	_, sc, err := wizardScript(s.model)
	if err != nil {
		return -1
	}
	used := variableRegisters(sc)
	for _, b := range s.dialogueHeld() {
		if f, err := mod.ParseDialogue("", b); err == nil {
			for _, n := range f.Nodes {
				for _, r := range n.Replies {
					if r.Once {
						used[uint32(r.Mark)] = true
					}
				}
			}
		}
	}
	for _, n := range d.Nodes {
		for _, r := range n.Replies {
			if r.Once {
				used[uint32(r.Mark)] = true
			}
		}
	}
	for v := dialogueVariables(s.model) - 1; v >= len(sc.Conditions); v-- {
		if !used[uint32(v)] {
			return v
		}
	}
	return -1
}

// DialogueUsers are the names of the triggers whose actions start the
// dialogue, a trigger with no name by its number.
func (s *mapInspectionEdits) DialogueUsers(name string) []string {
	if s.model == nil {
		return nil
	}
	_, sc, err := wizardScript(s.model)
	if err != nil {
		return nil
	}
	return dialogueUsers(sc, name)
}

func dialogueUsers(sc alm.Script, name string) []string {
	starts := map[uint32]bool{}
	for _, a := range sc.Actions {
		if a.InteractiveMessage() && a.Label == name {
			starts[a.ID] = true
		}
	}
	var out []string
	for i, t := range sc.Triggers {
		if slices.ContainsFunc(t.Acts, func(id uint32) bool { return starts[id] }) {
			label := t.Name
			if label == "" {
				label = "#" + strconv.Itoa(i+1)
			}
			out = append(out, label)
		}
	}
	return out
}

// WriteDialogue writes dialogue name and every node its replies name as one
// edit; the file reads back equal through the loader.
func (s *mapInspectionEdits) WriteDialogue(name string, d ui.DialogueData) (_ *ui.InspectionDocument, said error) {
	defer s.worded(&said)
	return s.writeDialogue(nil, name, d)
}

// WriteDialogueFor writes the dialogue as WriteDialogue does and the owner's
// trigger as the plan holds it, whose action starts the dialogue, as one edit.
func (s *mapInspectionEdits) WriteDialogueFor(owner ui.InspectionMessageOwner, name string, d ui.DialogueData) (_ *ui.InspectionDocument, said error) {
	defer s.worded(&said)
	plan := owner.Trigger
	if plan == nil || owner.Kind != ui.KindTrigger || owner.Action < 0 || owner.Action >= len(plan.Actions) {
		return nil, wordErr("editor.message.unavailable")
	}
	return s.writeDialogue(plan, name, d)
}

func (s *mapInspectionEdits) writeDialogue(plan *ui.WizardTrigger, name string, d ui.DialogueData) (*ui.InspectionDocument, error) {
	if !mod.ValidDialogueName(name) {
		return nil, wordErr("editor.dialogue.err.name")
	}
	if issues := s.DialogueIssues(d); len(issues) != 0 {
		return nil, wordErr("editor.dialogue.err.refused")
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		st, err := s.stage.workspace()
		if err != nil {
			return false, err
		}
		mark := m.Mark()
		if err := writeDialogueInto(m, st, name, d); err != nil {
			return false, err
		}
		if plan != nil {
			if err := writeWizardTrigger(m, *plan); err != nil {
				return false, err
			}
		}
		m.Squash(mark)
		return m.Mark() != mark, nil
	})
}

// writeDialogueInto writes the nodes of d into m and the file into st.
func writeDialogueInto(m *mapedit.Editor, st *editorWorkspaceState, name string, d ui.DialogueData) error {
	_, sc, err := wizardScript(m)
	if err != nil {
		return err
	}
	condRefs, actRefs := wizardRefs(sc)
	otherConds, otherActs := dialogueRefs(st.Dialogues, name)
	for id, n := range otherConds {
		condRefs[id] += n
	}
	for id, n := range otherActs {
		actRefs[id] += n
	}
	planned := map[[2]uint32]int{}
	var gone []goneNode
	if old := st.Dialogues[name]; old != nil {
		if f, err := mod.ParseDialogue(name+".toml", old); err == nil {
			gone = dialogueGone(f)
			for _, g := range gone {
				family := uint32(0)
				if g.condition {
					family = 1
				}
				planned[[2]uint32{family, g.id}] += 0x100
				if g.condition {
					condRefs[g.id]++
				} else {
					actRefs[g.id]++
				}
			}
		}
	}
	for _, n := range d.Nodes {
		for _, r := range n.Replies {
			for _, p := range r.Pairs {
				planned[[2]uint32{1, p.Left.ID}]++
				planned[[2]uint32{1, p.Right.ID}]++
			}
			for _, a := range r.Actions {
				planned[[2]uint32{0, a.ID}]++
			}
		}
	}
	pairs := make([][][]mod.DialogueCondition, len(d.Nodes))
	outs := make([][][]int, len(d.Nodes))
	for i, n := range d.Nodes {
		pairs[i], outs[i] = make([][]mod.DialogueCondition, len(n.Replies)), make([][]int, len(n.Replies))
		for k, r := range n.Replies {
			if len(r.Pairs) > mod.MaxDialogueWhen || len(r.Actions) > mod.MaxDialogueDo {
				return wordErr("editor.dialogue.err.rows")
			}
			for j, p := range r.Pairs {
				var ids [2]uint32
				for side, n := range []ui.WizardNode{p.Left, p.Right} {
					if ids[side], err = wizardNode(m, true, n, condRefs, planned); err != nil {
						return wordErr("editor.err.condition", j+1, err)
					}
				}
				pairs[i][k] = append(pairs[i][k], mod.DialogueCondition{Left: int(ids[0]), Right: int(ids[1]), Cmp: int(p.Cmp)})
			}
			for j, a := range r.Actions {
				id, err := wizardNode(m, false, a, actRefs, planned)
				if err != nil {
					return wordErr("editor.err.action", j+1, err)
				}
				outs[i][k] = append(outs[i][k], int(id))
			}
		}
	}
	f := toDialogue(name, d, pairs, outs)
	b, err := mod.WriteDialogue(f)
	if err != nil {
		return err
	}
	if st.Dialogues == nil {
		st.Dialogues = map[string][]byte{}
	}
	st.Dialogues[name] = b
	return dropDialogueNodes(m, st, gone)
}

// dropDialogueNodes deletes the nodes of gone that no trigger and no
// dialogue of st names now.
func dropDialogueNodes(m *mapedit.Editor, st *editorWorkspaceState, gone []goneNode) error {
	conds, acts := dialogueRefs(st.Dialogues, "")
	return dropNodes(m, gone, func(g goneNode) bool {
		if g.condition {
			return conds[g.id] != 0
		}
		return acts[g.id] != 0
	})
}

// RemoveDialogue deletes dialogue name and the nodes only it names; a
// dialogue a trigger starts is refused, naming the triggers.
func (s *mapInspectionEdits) RemoveDialogue(name string) (_ *ui.InspectionDocument, said error) {
	defer s.worded(&said)
	if users := s.DialogueUsers(name); len(users) != 0 {
		return nil, wordErr("editor.dialogue.err.used", name, strings.Join(users, ", "))
	}
	return s.change(func(m *mapedit.Editor) (bool, error) {
		st, err := s.stage.workspace()
		if err != nil {
			return false, err
		}
		old := st.Dialogues[name]
		if old == nil {
			return false, nil
		}
		var gone []goneNode
		if f, err := mod.ParseDialogue(name+".toml", old); err == nil {
			gone = dialogueGone(f)
		}
		st.Dialogues[name] = nil
		mark := m.Mark()
		if err := dropDialogueNodes(m, st, gone); err != nil {
			return false, err
		}
		m.Squash(mark)
		return m.Mark() != mark, nil
	})
}
