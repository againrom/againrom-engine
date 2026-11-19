package game

import (
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) modFields(out []ui.InspectionField) []ui.InspectionField {
	x, p := s.inspector, s.workspace.Package
	campaign := p.Kind == mod.MissionCampaign || p.Kind == mod.MissionReplace
	fixed := s.fileMissionFixed(s.workspace) != nil
	for i := range out {
		f := &out[i]
		f.Label = x.modFieldLabel(f.Key)
		// The mod's own fields stand left under Mod, the chosen mission's
		// right under Mission.
		f.Column, f.Group = 1, x.word("editor.mod.source")
		if _, manifest := mod.ManifestKeyNamed(f.Key); manifest || f.Key == "folder" || f.Key == "id" || f.Key == "title" {
			f.Column, f.Group = 0, x.word("editor.mod.window")
		}
		switch f.Key {
		case "kind":
			if p.Kind == mod.MissionCampaign {
				f.ReadOnly, f.Text, f.TextValue = true, true, x.word("editor.mod.campaign")
				f.Lines = []ui.InspectionLine{{Text: x.inspectionUTF8(f.Label + ": " + f.TextValue)}}
				break
			}
			f.Text = false
			f.Value = 0
			if p.Kind == mod.MissionReplace {
				f.Value = 1
			}
			f.Choices = []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.mod.guestmode"))}, {Value: 1, Label: x.inspectionUTF8(x.word("editor.mod.replace"))}}
		case "after":
			if x.archives == nil || x.archives.Containers == nil {
				break
			}
			c, e := LoadCampaign(x.archives.Containers)
			if e == nil {
				for _, n := range c.Main {
					if _, _, ok := c.opensTown(n); ok {
						f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(n), Label: x.inspectionUTF8(fmt.Sprintf("%s %d", x.word("editor.mod.campaign"), n))})
					}
				}
			}
			if d := s.workspace.Project; d != nil {
				for _, m := range d.Missions {
					if !m.Off && m.Kind == mod.MissionGuest && m.Number != p.Number {
						f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(m.Number), Label: fmt.Sprintf("%s (%d)", m.Key, m.Number)})
					}
				}
			}
		case "guest":
			for _, choice := range x.npcChoices(65535, false) {
				if x.archives == nil || x.archives.Containers == nil {
					break
				}
				c, e := LoadCampaign(x.archives.Containers)
				if e == nil {
					if chapter, ok := c.Chapters[p.Chapter]; ok {
						for _, id := range chapter.AddHero {
							if int64(id) == choice.Value {
								f.Choices = append(f.Choices, choice)
							}
						}
					}
				}
			}
		}
		if f.Key == "chapter" || f.Key == "id" && x.mod != nil || fixed && (f.Key == "kind" || f.Key == "number") || campaign && f.Key == "after" || campaign && p.Guest == 0 && len(p.Party) == 0 && (f.Key == "guest" || f.Key == "gold") || len(p.Party) > 0 && (f.Key == "guest" || f.Key == "chapter") {
			f.ReadOnly = true
			f.Lines = []ui.InspectionLine{{Text: x.inspectionUTF8(f.Label+": ") + fieldLabel(*f)}}
		}
	}
	landing := int64(0)
	if len(p.Party) > 0 {
		landing = 1
	}
	if campaign && p.Guest == 0 && len(p.Party) == 0 {
		landing = 2
	}
	{
		choices := []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.mod.companion"))}, {Value: 1, Label: x.inspectionUTF8(x.word("editor.mod.party"))}}
		if campaign {
			choices = append(choices, ui.InspectionChoice{Value: 2, Label: x.inspectionUTF8(x.word("editor.graph.mainparty"))})
			auto := ""
			if p.AutoGetMission != nil {
				auto = strconv.Itoa(*p.AutoGetMission)
			}
			out = append(out, ui.InspectionField{Key: "AutoGetMission", Label: x.word("editor.graph.successor"), Meaning: x.word("editor.graph.flowhint"), Text: true, TextValue: auto, Column: 1})
		}
		out = append(out, ui.InspectionField{Key: "landing", Label: x.word("editor.mod.landing"), Value: landing, Column: 1, Choices: choices})
		if campaign && s.campaignLocked() != nil {
			for i := range out {
				if f := &out[i]; f.Key == "AutoGetMission" || f.Key == "landing" {
					f.ReadOnly = true
					f.Lines = []ui.InspectionLine{{Text: x.inspectionUTF8(f.Label+": ") + fieldLabel(*f)}}
				}
			}
		}
		if !campaign {
			out = append(out, x.heroField("class", p.Class), x.heroField("sex", p.Sex))
		}
		if len(p.Party) == 0 && p.Guest != 0 {
			out = append(out, x.modLine("carry", x.word("editor.mod.carryjoin")))
		}
	}
	off := "editor.mod.offno"
	if d := s.workspace.Project; d != nil && d.Selected >= 0 && d.Selected < len(d.Missions) && d.Missions[d.Selected].Off {
		off = "editor.mod.offyes"
	}
	out = append(out, x.modLine("off", x.word(off)))
	for _, key := range []string{missionTitleKey, missionBriefKey} {
		if f, ok := s.missionTextField(key); ok {
			out = append(out, f)
		}
	}
	for n, g := range p.Party {
		prefix := fmt.Sprintf("party.%d.", n)
		characters := s.modCharacters()
		var choices []ui.InspectionChoice
		value := int64(0)
		for i, c := range characters {
			name := x.inspectionUTF8(c.Name)
			if name == "" {
				name = c.Key
			}
			for _, npc := range x.npcChoices(65535, false) {
				if npc.Value == int64(c.NPC) {
					name = npc.Label
					break
				}
			}
			choices = append(choices, ui.InspectionChoice{Value: int64(i), Label: name + " (" + c.Key + ")"})
			if c.Key == g.Character {
				value = int64(i)
			}
		}
		label := fmt.Sprintf("%s %d", x.word("editor.mod.member"), n+1)
		if n == 0 {
			label += " / " + x.word("editor.mod.leader")
		}
		out = append(out, ui.InspectionField{Key: prefix + "character", Label: label, Value: value, Choices: choices, Column: 1})
		levels := []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.mod.default"))}}
		for level := 1; level <= 4; level++ {
			levels = append(levels, ui.InspectionChoice{Value: int64(level), Label: strconv.Itoa(level)})
		}
		out = append(out, ui.InspectionField{Key: prefix + "level", Label: x.word("editor.mod.level"), Value: int64(g.Level), Choices: levels, Column: 1, Parent: prefix})
		for _, cell := range []struct {
			key   string
			value *string
		}{{"weapon", g.Weapon}, {"shield", g.Shield}} {
			out = append(out, s.partyItemField(prefix, cell.key, cell.key, cell.value))
		}
		armour := int64(0)
		if g.Armour != nil {
			armour = 1
		}
		out = append(out, ui.InspectionField{Key: prefix + "armour", Label: x.word("editor.mod.armour"), Value: armour, Column: 1, Parent: prefix, Choices: []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.mod.template"))}, {Value: 1, Label: x.inspectionUTF8(x.word("editor.mod.override"))}}})
		if g.Armour != nil {
			for cell := 0; cell < 8; cell++ {
				name := ""
				if cell < len(g.Armour) {
					name = g.Armour[cell]
				}
				out = append(out, s.partyItemField(prefix, "armour"+strconv.Itoa(cell), "armour", &name))
			}
		}
		for _, action := range []string{"up", "down", "remove"} {
			out = append(out, ui.InspectionField{Key: prefix + action, Label: x.modFieldLabel(action), Column: 1, Parent: prefix, Action: true})
		}
	}
	if len(p.Party) > 0 && len(p.Party) < mod.MaxGuestParty {
		out = append(out, ui.InspectionField{Key: "party.0.add", Label: x.word("editor.mod.addmember"), Column: 1, Action: true})
	}
	for _, action := range []string{"add", "up", "down", "remove"} {
		out = append(out, ui.InspectionField{Key: action, Label: x.modFieldLabel(action), Action: true})
	}
	output, _ := x.publishFolder()
	f := workspaceText("output", x.word("editor.mod.output"), output)
	f.Group = x.word("editor.mod.window")
	out = append(out, f)
	// The working and publish folders are chosen in the system folder
	// window: each row carries its Choose button.
	for _, field := range []string{"folder", "output"} {
		out = append(out, ui.InspectionField{Key: field + ".pick", Label: x.word("editor.pick.choose"), Action: true, Of: field, Meaning: x.word("editor.pick.folder.help")})
	}
	for i := range out {
		if out[i].Meaning == "" {
			out[i].Meaning = x.modFieldHelp(out[i].Key)
		}
		if out[i].Group == "" {
			out[i].Group = x.word("editor.mod.source")
			if out[i].Column == 0 {
				out[i].Group = x.word("editor.mod.window")
			}
		}
	}
	return s.missionBlock(out)
}

// missionBlock leads the Mission block with the mission choice, its chain
// line with Move up and Move down, and, while the open map is no mission of
// the mod, the button that adds it: the actions a mission has stand beside
// it, not in the window's button row.
func (s *mapInspectionEdits) missionBlock(out []ui.InspectionField) []ui.InspectionField {
	x := s.inspector
	chosen := false
	if d := s.workspace.Project; d != nil && d.Selected >= 0 && d.Selected < len(d.Missions) && !d.Missions[d.Selected].Off {
		chosen = true
	}
	summary := s.ModSummary()
	head := []ui.InspectionField{}
	if i := slices.IndexFunc(out, func(f ui.InspectionField) bool { return f.Key == "source" }); i >= 0 {
		head = append(head, out[i])
		out = slices.Delete(out, i, i+1)
	}
	if summary != "" {
		chain := ui.InspectionField{Key: ui.ModChainKey, ReadOnly: true, Text: true, TextValue: summary, Column: 1, Group: x.word("editor.mod.source"),
			Lines: []ui.InspectionLine{{Text: x.inspectionUTF8(summary)}}}
		if chosen {
			chain.Value = 1
		}
		head = append(head, chain)
	}
	if i := slices.IndexFunc(out, func(f ui.InspectionField) bool { return f.Key == "add" }); i >= 0 && !chosen {
		out[i].Column, out[i].Group = 1, x.word("editor.mod.source")
		head = append(head, out[i])
		out = slices.Delete(out, i, i+1)
	}
	at := slices.IndexFunc(out, func(f ui.InspectionField) bool { return f.Column == 1 })
	if at < 0 {
		at = len(out)
	}
	return slices.Insert(out, at, head...)
}

// modFieldHelp is the (?) help of the Missions page field key, the word
// "editor.mod.help.<key>" with the key's numbers dropped ("party.0.level"
// reads "party.level"); "" when the table has none.
func (x *MapInspector) modFieldHelp(key string) string {
	var parts []string
	for _, p := range strings.Split(key, ".") {
		if _, err := strconv.Atoi(p); err != nil {
			parts = append(parts, p)
		}
	}
	id := "editor.mod.help." + strings.Join(parts, ".")
	if h := x.word(id); h != id {
		return h
	}
	return ""
}

func fieldLabel(f ui.InspectionField) string {
	if f.Text {
		return f.TextValue
	}
	for _, c := range f.Choices {
		if c.Value == f.Value {
			return c.Label
		}
	}
	return strconv.FormatInt(f.Value, 10)
}

func (s *mapInspectionEdits) partyItemField(prefix, key, kind string, current *string) ui.InspectionField {
	x := s.inspector
	f := ui.InspectionField{Key: prefix + key, Label: x.modFieldLabel(kind), Column: 1, Parent: prefix, Value: -1, Choices: []ui.InspectionChoice{{Value: -1, Label: x.inspectionUTF8(x.word("editor.mod.template"))}, {Value: 0, Label: x.inspectionUTF8(x.word("editor.mod.empty"))}}}
	if kind == "armour" {
		f.Label = fmt.Sprintf("%s %s", f.Label, strings.TrimPrefix(key, "armour"))
		f.Choices = f.Choices[1:]
	}
	if current != nil {
		f.Value = -2
		if *current == "" {
			f.Value = 0
		}
	}
	for _, item := range x.recordCatalog("item") {
		name, ok := s.partyItemName(kind, item.Value)
		if !ok {
			continue
		}
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: item.Value, Label: item.Label})
		if current != nil && *current == name {
			f.Value = item.Value
		}
	}
	if f.Value == -2 {
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: -2, Label: x.inspectionUTF8(*current)})
	}
	return f
}

// heroField is the class or sex field over the install's hero choices; with
// none it is one read-only line saying so.
func (x *MapInspector) heroField(key, word string) ui.InspectionField {
	choices, ok := x.editorTables().heroChoices[key]
	if !ok {
		return x.modLine(key, x.word("editor.tables.none"))
	}
	f := ui.InspectionField{Key: key, Label: x.modFieldLabel(key), Column: 1}
	for i, c := range choices {
		f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(i), Label: x.inspectionUTF8(x.word(c[1]))})
		if c[0] == word {
			f.Value = int64(i)
		}
	}
	return f
}

// modLine is a read-only line of the mission page: key's label and text.
func (x *MapInspector) modLine(key, text string) ui.InspectionField {
	label := x.modFieldLabel(key)
	return ui.InspectionField{Key: key, Label: label, Column: 1, ReadOnly: true, Text: true, TextValue: text, Lines: []ui.InspectionLine{{Text: x.inspectionUTF8(label + ": " + text)}}}
}

func (x *MapInspector) modFieldLabel(key string) string {
	labels := map[string]string{
		"folder": x.word("editor.mod.folder"), "id": x.word("editor.mod.id"),
		"title": x.word("editor.mod.title"), "source": x.word("editor.mod.source"),
		"kind": x.word("editor.mod.kind"), "number": x.word("editor.mod.number"),
		"after": x.word("editor.mod.after"), "guest": x.word("editor.mod.guest"),
		"chapter": x.word("editor.mod.chapter"), "gold": x.word("editor.mod.gold"),
		"weapon": x.word("editor.mod.weapon"), "shield": x.word("editor.mod.shield"),
		"armour": x.word("editor.mod.armour"), "add": x.word("editor.mod.add"),
		"up": x.word("editor.mod.up"), "down": x.word("editor.mod.down"),
		"remove": x.word("editor.mod.remove"),
	}
	if label, ok := labels[key]; ok {
		return label
	}
	return x.word("editor.mod." + key)
}

func (s *mapInspectionEdits) modDestination(w *editorWorkspaceState) (int, bool) {
	c, err := LoadCampaign(s.inspector.archives.Containers)
	if err != nil {
		return 0, false
	}
	m := w.Package.mission(nil, nil, mod.MissionPlace{})
	if w.Package.Kind == mod.MissionCampaign || w.Package.Kind == mod.MissionReplace {
		copy := cloneWorkspace(w)
		s.syncModDocument(copy, copy.Pages, copy.Labels, false)
		set, _ := s.inspector.modMissions(s.inspector.mod.description(), copy.Project.Missions)
		if flow, err := c.withModMissions(set); err == nil {
			towns := flow.missionTowns[w.Package.Number]
			if len(towns) > 0 {
				return towns[0], true
			}
		}
		return 0, false
	}
	d := mod.MissionData{Missions: []mod.Mission{m}}
	if p := w.Project; p != nil && p.Selected >= 0 {
		m, err = p.selected(w.Package, s.inspector.word)
		if err != nil {
			return 0, false
		}
		d.Missions = append([]mod.Mission(nil), p.Missions...)
		d.Missions[p.Selected] = m
	}
	d, _, err = mod.MergeMissions(d)
	if err != nil {
		return 0, false
	}
	chain := d.Chain(m)
	if len(chain) == 0 {
		return 0, false
	}
	town, ok := c.NextMission(chain[0].After)
	return town.Mission, ok && town.Town
}

// modContextRefusal is the read-only line of a mod whose records did not load,
// empty for any other mod and with no mod open.
func (x *MapInspector) modContextRefusal() string {
	if x.mod == nil || x.mod.contextIssue == "" {
		return ""
	}
	return x.word("editor.mod.readonly") + " " + x.mod.contextIssue
}

func (s *mapInspectionEdits) ModSummary() string {
	if refusal := s.inspector.modContextRefusal(); refusal != "" {
		return refusal
	}
	w := s.workspace
	if w == nil || w.Project == nil {
		return ""
	}
	d, _, err := mod.MergeMissions(mod.MissionData{Missions: w.Project.Missions})
	if err != nil {
		return s.inspector.word("editor.mod.brokenchain")
	}
	if w.Project.Selected < 0 || w.Project.Selected >= len(w.Project.Missions) {
		return s.inspector.word("editor.mod.unregistered")
	}
	m := w.Project.Missions[w.Project.Selected]
	if m.Kind == mod.MissionReplace {
		return s.inspector.word("editor.mod.replace")
	}
	chain := d.Chain(m)
	var labels []string
	if len(chain) > 0 {
		labels = append(labels, fmt.Sprintf("%s %d", s.inspector.word("editor.mod.campaign"), chain[0].After))
	}
	for _, m := range chain {
		labels = append(labels, fmt.Sprintf("%s (%d)", m.Key, m.Number))
	}
	if town, ok := s.modDestination(w); ok {
		labels = append(labels, fmt.Sprintf("%s %d", s.inspector.word("editor.mod.chapter"), town))
	}
	return strings.Join(labels, " -> ")
}

func (s *mapInspectionEdits) ModOutput() string {
	output, _ := s.inspector.publishFolder()
	if output == "" || s.workspace == nil {
		return ""
	}
	return filepath.Join(output, mod.PackedName(s.workspace.Package.ID))
}
