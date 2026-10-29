package game

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"againrom/pkg/mod"
	"againrom/pkg/ui"
)

func (s *mapInspectionEdits) modFields(out []ui.InspectionField) []ui.InspectionField {
	x, p := s.inspector, s.workspace.Package
	if p.Kind == mod.MissionReplace {
		p.Party = nil
	}
	for i := range out {
		f := &out[i]
		f.Label = x.modFieldLabel(f.Key)
		f.Column = 1
		switch f.Key {
		case "folder", "id", "title", "source":
			f.Column = 0
		case "kind":
			f.Text = false
			f.Value = 0
			if p.Kind == mod.MissionReplace {
				f.Value = 1
			}
			f.Choices = []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.mod.guestmode"))}, {Value: 1, Label: x.inspectionUTF8(x.word("editor.mod.replace"))}}
		case "after":
			c, e := LoadCampaign(x.archives.Containers)
			if e == nil {
				for _, n := range c.Main {
					if town, ok := c.NextMission(n); ok && town.Town {
						if _, auto := c.AutoAdvance(n); !auto {
							f.Choices = append(f.Choices, ui.InspectionChoice{Value: int64(n), Label: x.inspectionUTF8(fmt.Sprintf("%s %d", x.word("editor.mod.campaign"), n))})
						}
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
		if f.Key == "chapter" || f.Key == "id" && x.mod != nil || p.Kind == mod.MissionReplace && (f.Key == "after" || f.Key == "guest" || f.Key == "gold") || len(p.Party) > 0 && (f.Key == "guest" || f.Key == "chapter") {
			f.ReadOnly = true
			f.Lines = []ui.InspectionLine{{Text: x.inspectionUTF8(f.Label+": ") + fieldLabel(*f)}}
		}
	}
	landing := int64(0)
	if len(p.Party) > 0 {
		landing = 1
	}
	if p.Kind != mod.MissionReplace {
		out = append(out, ui.InspectionField{Key: "landing", Label: x.word("editor.mod.landing"), Value: landing, Column: 1, Choices: []ui.InspectionChoice{{Value: 0, Label: x.inspectionUTF8(x.word("editor.mod.companion"))}, {Value: 1, Label: x.inspectionUTF8(x.word("editor.mod.party"))}}})
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
	out = append(out, workspaceText("output", x.word("editor.mod.output"), x.modOutput))
	return out
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
	return labels[key]
}

func (s *mapInspectionEdits) modDestination(w *editorWorkspaceState) (int, bool) {
	c, err := LoadCampaign(s.inspector.archives.Containers)
	if err != nil {
		return 0, false
	}
	m := w.Package.mission(nil, nil)
	d := mod.MissionData{Missions: []mod.Mission{m}}
	if p := w.Project; p != nil && p.Selected >= 0 {
		m, err = p.selected(w.Package)
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

func (s *mapInspectionEdits) ModSummary() string {
	if m := s.inspector.mod; m != nil && m.contextIssue != "" {
		return s.inspector.word("editor.mod.readonly") + " " + m.contextIssue
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
	if s.inspector.modOutput == "" {
		return ""
	}
	return filepath.Join(s.inspector.modOutput, mod.PackedName(s.workspace.Package.ID))
}
