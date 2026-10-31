package game

import (
	"fmt"
	"image"
	"math"
	"slices"

	"againrom/pkg/base"
	"againrom/pkg/data"
	"againrom/pkg/mapedit"
	"againrom/pkg/mapload"
	"againrom/pkg/ui"
)

func (x *MapInspector) npcFields(m *mapedit.Editor, index int, value int64) (int64, []mapedit.FieldSet, error) {
	if !x.table.ClassPlacementFields() || value < -1 || value > math.MaxUint16 {
		return 0, nil, fmt.Errorf("NPC record is outside this placement's native key domain")
	}
	view, err := m.Map()
	if err != nil {
		return 0, nil, err
	}
	u := view.AuthoredUnits[index]
	if mapload.Resolve(u, x.table).Arm == mapload.ArmUnits {
		return 0, nil, fmt.Errorf("a creature placement cannot take an NPC record")
	}
	if value == -1 && u.Flags&1 == 0 {
		return int64(u.ClassSubID), nil, nil
	}
	probe := u
	flag := int64(0)
	if value >= 0 {
		if _, ok := x.table.NPCCatalog.Entry(int32(value)); !ok {
			return 0, nil, fmt.Errorf("NPC record %d does not exist", value)
		}
		probe.Flags |= 1
		probe.ClassSubID, flag = uint16(value), 1
	} else {
		probe.Flags &^= 1
		probe.ClassSubID = 0
	}
	if !mapload.Resolve(probe, x.table).Found() {
		return 0, nil, fmt.Errorf("NPC choice %d resolves to no Human definition", value)
	}
	if probe.Flags == u.Flags {
		return int64(probe.ClassSubID), nil, nil
	}
	return int64(probe.ClassSubID), []mapedit.FieldSet{{Key: "npcflag", Value: flag}}, nil
}

func (x *MapInspector) npcChoices(maxID int64, roles bool) []ui.InspectionChoice {
	var choices []ui.InspectionChoice
	if x.table == nil {
		return choices
	}
	for _, e := range x.table.NPCCatalog.Records() {
		id := int64(e.ID)
		if id < 0 || id > maxID {
			continue
		}
		choices = append(choices, ui.InspectionChoice{Value: id, Label: fmt.Sprintf("%d: %s", id, x.inspectionUTF8(x.npcName(e, roles)))})
	}
	slices.SortFunc(choices, func(a, b ui.InspectionChoice) int { return int(a.Value - b.Value) })
	return choices
}

// npcPortraits are the faces of the NPC records the choices name, by choice
// value: each the picture a dialogue shows when that NPC speaks (SpeakerFace
// over the game's speaker records), cut as the dialogue pane cuts it. A choice
// whose record has no face has no entry. Each face is composed once per
// inspector.
func (x *MapInspector) npcPortraits(choices []ui.InspectionChoice) map[int64]*image.RGBA {
	if x == nil || x.table == nil || x.archives == nil {
		return nil
	}
	if x.recordPictures == nil {
		x.recordPictures = map[string]map[int64]*image.RGBA{}
	}
	cache := x.recordPictures["speaker"]
	if cache == nil {
		cache = map[int64]*image.RGBA{}
		x.recordPictures["speaker"] = cache
	}
	var r *speakerResolver
	out := map[int64]*image.RGBA{}
	for _, c := range choices {
		pic, tried := cache[c.Value]
		if !tried {
			if r == nil {
				r = &speakerResolver{src: x.archives.artSource(), units: x.units, npcFaces: speakerFaces(x.table),
					portraits: map[string]*image.RGBA{}, figurePics: map[figureCacheKey]*image.RGBA{}}
			}
			if c.Value >= 0 && c.Value <= math.MaxInt32 {
				if face, window, ok := r.SpeakerFace(int(c.Value)); ok {
					pic = ui.SpeakerPortrait(face, window)
				}
			}
			cache[c.Value] = pic
		}
		if pic != nil {
			out[c.Value] = pic
		}
	}
	return out
}

// npcName is NPC record e's name in UTF-8. With roles, as the message popup
// names a speaker, a speaker whose person depends on the first hero is its
// role's word; otherwise the record's name, its npcnames.txt line, or its
// section.
func (x *MapInspector) npcName(e data.NPCEntry, roles bool) string {
	id := int64(e.ID)
	if roles {
		if role, ok := x.speakerRole(id); ok {
			return role
		}
	}
	name := e.Name
	line := id
	if !roles {
		line--
	}
	if name == "" && line >= 0 && line < int64(len(x.table.NPCNames)) {
		name = x.table.NPCNames[line]
	}
	if name == "" {
		name = e.Section
	}
	return x.messageUTF8([]byte(name))
}

// speakerLabel is speaker id's label in UTF-8 as the message popup builds
// it, false when the install has no such speaker.
func (x *MapInspector) speakerLabel(id int64) (string, bool) {
	if role, ok := x.speakerRole(id); ok {
		return role, true
	}
	if x.table == nil || x.table.NPCCatalog == nil || id < 0 || id > math.MaxInt32 {
		return "", false
	}
	e, ok := x.table.NPCCatalog.Entry(int32(id))
	if !ok {
		return "", false
	}
	return x.npcName(e, true), true
}

// heroLabels builds the hero band's labels from the words format and
// roleFormat, each text passed through encode. On a first-game install heroes
// 1 to 6 name speakers 21 to 26 (TRIG-HEROTPL-076), and each carries that
// speaker's label; on a second-game install the label stays a number.
func (x *MapInspector) heroLabels(format, roleFormat string, encode func(string) string) ui.HeroLabels {
	h := ui.HeroLabels{Format: encode(format), RoleFormat: encode(roleFormat)}
	if x == nil {
		return h
	}
	var profile base.Profile
	if x.archives != nil {
		profile = x.archives.Base.Profile
	}
	if !profile.HasFirstData() {
		return h
	}
	h.Roles = map[int64]string{}
	for n := int64(1); n <= 6; n++ {
		if label, ok := x.speakerLabel(20 + n); ok {
			h.Roles[n] = encode(label)
		}
	}
	return h
}

// editorHeroLabels are the hero band's labels in the editor's words and the
// font's bytes.
func (x *MapInspector) editorHeroLabels() ui.HeroLabels {
	return x.heroLabels(x.word("editor.param.hero"), x.word("editor.param.herorole"), x.inspectionUTF8)
}
