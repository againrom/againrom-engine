package game

import (
	"errors"
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/formats/alm"
	"againrom/pkg/mapedit"
	"againrom/pkg/mod"
	"againrom/pkg/namefold"
	"againrom/pkg/ui"
)

// globalsRows is the save gate of the globals file: a file the game cannot
// read is refused, naming the file.
var globalsRows = modRowsPage{save: "editor.globals.err.save", rel: mod.GlobalsFile}

// globalLabel names a global where the page shows it.
func globalLabel(g mod.Global) string { return g.Name }

// globalUsers is where the open project names each global id: the open
// map as the editor holds it, every other map of the mod as its file or
// draft holds it, and the dialogues beside each. A place is named once per
// id. A map or dialogue the game cannot read names nothing.
func (s *mapInspectionEdits) globalUsers(w *editorWorkspaceState) map[int32][]string {
	out := map[int32][]string{}
	x := s.inspector
	if x.mod == nil || w == nil || w.Project == nil {
		return out
	}
	d := w.Project
	view := projectView(x.mod.files, d.Files)
	read := func(rel string) []byte {
		if b, drafted := view.Draft(rel); drafted {
			return b
		}
		b, _ := modDataFile(x.mod.dir, rel)
		return b
	}
	list, _ := s.mapList(d)
	own := namefold.ModTree(s.ownMap())
	for _, e := range list {
		for _, rel := range e.Files {
			var script alm.Script
			var dialogues map[string][]byte
			if s.model != nil && own != "" && namefold.ModTree(rel) == own {
				m, err := s.model.Map()
				if err != nil {
					continue
				}
				if script, err = m.Script(); err != nil {
					continue
				}
				dialogues = map[string][]byte{}
				for name, b := range w.dialogueFiles() {
					dialogues[name+".toml"] = b
				}
			} else {
				b := read(rel)
				if len(b) == 0 {
					continue
				}
				model, err := mapedit.New(b)
				if err != nil {
					continue
				}
				m, err := model.Map()
				if err != nil {
					continue
				}
				if script, err = m.Script(); err != nil {
					continue
				}
				prefix := namefold.ModTree(mod.MissionPlace{Map: rel}.Dialogs()) + "/"
				var entries []string
				spelled := map[string]string{}
				for _, name := range view.names {
					if e := path0(name); namefold.ModTree(name) == prefix+namefold.ModTree(e) {
						entries = append(entries, e)
						spelled[e] = name
					}
				}
				dialogues = mod.DialogueFiles(entries, mod.MaxFileBytes, func(e string) ([]byte, bool) {
					b := read(spelled[e])
					return b, len(b) > 0
				})
			}
			uses, loose := scriptGlobalUses(script, dialogues)
			for _, n := range loose {
				key := "editor.globals.use.action"
				if n.Condition {
					key = "editor.globals.use.condition"
				}
				if place := fmt.Sprintf(x.word(key), rel, n.Index); !slices.Contains(out[n.ID], place) {
					out[n.ID] = append(out[n.ID], place)
				}
			}
			for _, u := range uses {
				var place string
				if u.Trigger >= 0 {
					place = fmt.Sprintf(x.word("editor.globals.use.trigger"), rel, script.Triggers[u.Trigger].Name)
				} else {
					place = fmt.Sprintf(x.word("editor.globals.use.reply"), rel, u.File, u.Node, u.Reply)
				}
				if !slices.Contains(out[u.ID], place) {
					out[u.ID] = append(out[u.ID], place)
				}
			}
		}
	}
	return out
}

// path0 is the last element of a slash path.
func path0(name string) string {
	if i := strings.LastIndexByte(name, '/'); i >= 0 {
		return name[i+1:]
	}
	return name
}

// globalsFields is W's Globals page: the New button, then each global's
// name, number, starting value, note, users and removal.
func (s *mapInspectionEdits) globalsFields() []ui.InspectionField {
	x := s.inspector
	if !s.modPageHolds() {
		return nil
	}
	project := s.workspace.Project
	if project == nil {
		project = x.mod.document()
	}
	held, err := s.heldRows(project, nil, mod.GlobalsFile)
	if err != nil {
		return []ui.InspectionField{x.pageLine("error", "", err.Error())}
	}
	add := ui.InspectionField{Key: ui.GlobalNewKey, Label: x.word("editor.globals.add"), Meaning: x.word("editor.globals.add.help"), Action: true, Column: 1}
	out := []ui.InspectionField{add}
	users := s.globalUsers(s.workspace)
	lang := x.pageLanguage()
	for i, g := range held.Globals.Globals {
		key := "global." + strconv.Itoa(i) + "."
		start := len(out)
		name := workspaceText(key+"name", x.word("editor.globals.name"), g.Name)
		name.Meaning = x.word("editor.globals.name.help")
		initial := workspaceText(key+"initial", x.word("editor.globals.initial"), strconv.Itoa(int(g.Initial)))
		initial.Meaning = x.word("editor.globals.initial.help")
		note := workspaceText(key+"note", x.word("editor.globals.note"), g.Notes[lang])
		note.Meaning = x.word("editor.globals.note.help")
		used := x.word("editor.globals.users.none")
		if u := users[g.ID]; len(u) > 0 {
			used = fmt.Sprintf(x.word("editor.globals.users"), strings.Join(u, "; "))
		}
		out = append(out, name, x.pageLine(key+"id", "", fmt.Sprintf(x.word("editor.globals.id"), g.ID)), initial, note, x.pageLine(key+"users", "", used),
			ui.InspectionField{Key: key + "remove", Label: x.word("editor.globals.remove"), Meaning: x.word("editor.globals.remove.help"), Action: true, Column: 1})
		for k := start; k < len(out); k++ {
			out[k].Group = globalLabel(g)
		}
	}
	return out
}

// editGlobals is one draft step of the globals file in the project w.
func (s *mapInspectionEdits) editGlobals(w *editorWorkspaceState, edit func(*mod.GlobalData) error) error {
	return kindFiles{s: s, w: w}.editRows(nil, []string{mod.GlobalsFile}, func(d *mod.Data) error { return edit(&d.Globals) })
}

// setGlobalsField makes one edit of the Globals page a draft step.
func (s *mapInspectionEdits) setGlobalsField(key, value string) (*ui.InspectionDocument, error) {
	return s.modPageEdit(key, value, s.globalsEdit)
}

func (s *mapInspectionEdits) globalsEdit(key, value string) (*ui.InspectionDocument, error) {
	x := s.inspector
	gone := x.modGone("editor.globals.err.gone", key)
	if key == ui.GlobalNewKey {
		return s.editKind(func(k kindFiles) error {
			named := s.globalUsers(k.w)
			return s.editGlobals(k.w, func(d *mod.GlobalData) error { return s.addGlobal(d, named) })
		})
	}
	kind, i, field, ok := modRowKey(key)
	if !ok || kind != "global" {
		return nil, gone
	}
	return s.editKind(func(k kindFiles) error {
		var users map[int32][]string
		if field == "remove" {
			users = s.globalUsers(k.w)
		}
		return s.editGlobals(k.w, func(d *mod.GlobalData) error {
			if i >= len(d.Globals) {
				return gone
			}
			return s.editGlobal(d, i, field, value, users, gone)
		})
	})
}

// addGlobal appends a global of the first number no declaration and no map
// names, called global<number>. A number a map names is never given again
// while the map names it.
func (s *mapInspectionEdits) addGlobal(d *mod.GlobalData, named map[int32][]string) error {
	x := s.inspector
	if len(d.Globals) >= mod.MaxGlobals {
		return fmt.Errorf(x.word("editor.globals.err.full"), mod.MaxGlobals)
	}
	for id := int32(mod.FirstGlobal); id <= mod.LastGlobal; id++ {
		if _, taken := named[id]; taken || slices.ContainsFunc(d.Globals, func(g mod.Global) bool { return g.ID == id }) {
			continue
		}
		name := "global" + strconv.Itoa(int(id))
		if slices.ContainsFunc(d.Globals, func(g mod.Global) bool { return g.Name == name }) {
			continue
		}
		d.Globals = append(d.Globals, mod.Global{Mod: x.mod.id, File: mod.GlobalsFile, ID: id, Name: name})
		return nil
	}
	return fmt.Errorf(x.word("editor.globals.err.full"), mod.MaxGlobals)
}

// editGlobal makes the edit of field of global i. A global a map or a reply
// uses is not removed: the refusal names where it is used.
func (s *mapInspectionEdits) editGlobal(d *mod.GlobalData, i int, field, value string, users map[int32][]string, gone error) error {
	x := s.inspector
	g := &d.Globals[i]
	switch field {
	case "remove":
		if u := users[g.ID]; len(u) > 0 {
			return fmt.Errorf(x.word("editor.globals.err.used"), g.Name, strings.Join(u, "; "))
		}
		d.Globals = slices.Delete(d.Globals, i, i+1)
	case "name":
		if !mod.ValidGlobalName(value) {
			return errors.New(x.word("editor.globals.err.name"))
		}
		if slices.ContainsFunc(d.Globals, func(o mod.Global) bool { return o.ID != g.ID && o.Name == value }) {
			return fmt.Errorf(x.word("editor.globals.err.twice"), value)
		}
		g.Name = value
	case "initial":
		n, err := strconv.ParseInt(value, 10, 32)
		if err != nil || n < math.MinInt32 || n > math.MaxInt32 {
			return fmt.Errorf(x.word("editor.globals.err.initial"), math.MinInt32, math.MaxInt32)
		}
		g.Initial = int32(n)
	case "note":
		notes := map[string]string{}
		for k, v := range g.Notes {
			notes[k] = v
		}
		if lang := x.pageLanguage(); value == "" {
			delete(notes, lang)
		} else {
			notes[lang] = value
		}
		if len(notes) == 0 {
			notes = nil
		}
		g.Notes = notes
	default:
		return gone
	}
	return nil
}

// modGlobalList is the globals the open mod declares, the project's drafts
// over its folder's file; nil outside a mod.
func (x *MapInspector) modGlobalList() []mod.Global {
	m := x.mod
	if m == nil {
		return nil
	}
	if m.loaded != nil {
		return m.loaded.Data.Globals.Of(m.id)
	}
	if src, err := modDataFile(m.dir, mod.GlobalsFile); err == nil && src != nil {
		if d, err := mod.ParseGlobals(m.id, mod.GlobalsFile, src); err == nil {
			return d.Globals
		}
	}
	return nil
}
