package game

import (
	"errors"
	"strings"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/mod"
)

// SetModCharacters applies the character edits the mods make to the definition
// rows of the people the game builds, and records the names the panels show.
// It runs once, after SetMods and before any game is opened.
//
// An edit names definition rows (a row, or every tier of a family) and changes
// the class and figure columns of those rows only, takes equipment off their
// cells, and gives them a display name. Everything that builds a person from a
// row reads the edited row, so a tavern candidate, a hired squad, a saved party
// and a mission's placement of the row agree. A refusal names the mod, the file
// and the line.
func (f *FrontEnd) SetModCharacters(chars mod.CharacterData) error {
	if chars.Empty() {
		return nil
	}
	t := f.Table
	if t == nil || t.Humans == nil {
		return errors.New("the install has no definition rows for the mods to edit")
	}
	base := t.Humans
	edits := map[int]data.RowEdit{}
	var named []mapload.ModCharacter
	for _, c := range chars.Rows {
		rows := characterRows(base, c.Target)
		if len(rows) == 0 {
			return itemFileError(c.Mod, c.File, c.Line, "no definition row is named %q, nor is %q a family of tier rows", c.Target, c.Target)
		}
		var names []string
		for _, i := range rows {
			name := base.EntryName(i)
			names = append(names, name)
			edit := edits[i]
			if edit.Params == nil {
				edit.Params = map[int]int32{}
			}
			if edit.Cells == nil {
				edit.Cells = map[int]string{}
			}
			if len(base.EntryParams(i)) < data.HumanSlotsRequired {
				return itemFileError(c.Mod, c.File, c.Line, "the row %q carries no class and figure columns to edit", name)
			}
			if c.Kind != "" {
				like, ok := characterKindRow(base, c.Kind, name)
				if !ok {
					return itemFileError(c.Mod, c.File, c.KindLine, "no definition row is named %q, nor %q followed by the tier of %q", c.Kind, c.Kind, name)
				}
				p := base.EntryParams(like)
				if len(p) < data.HumanSlotsRequired {
					return itemFileError(c.Mod, c.File, c.KindLine, "the row %q carries no class and figure columns to take", base.EntryName(like))
				}
				for _, slot := range []int{data.HumanSlotTypeID, data.HumanSlotFace, data.HumanSlotGender} {
					edit.Params[slot] = p[slot]
				}
			}
			if c.Face != 0 {
				edit.Params[data.HumanSlotFace] = c.Face
			}
			for _, g := range c.Strip {
				switch g {
				case mod.StripWeapon:
					edit.Cells[data.HumanCellWeapon] = ""
				case mod.StripShield:
					edit.Cells[data.HumanCellShield] = ""
				case mod.StripArmour:
					for cell := data.HumanCellArmour; cell < data.HumanCellCount; cell++ {
						edit.Cells[cell] = ""
					}
				}
			}
			edits[i] = edit
		}
		if c.Name != "" {
			// A panel draws the install's own byte alphabet, not UTF-8.
			shown, err := encodeSaveLabel(c.Name, f.textSelector())
			if err != nil {
				return itemFileError(c.Mod, c.File, c.NameLine, "the name %q cannot be drawn by this install: %v", c.Name, err)
			}
			named = append(named, mapload.ModCharacter{Mod: c.Mod, Rows: names, Name: shown})
		}
	}
	edited := data.NewEditedRows(base, edits)
	t.Humans = edited
	f.Humans = edited
	t.Mods.Characters = append(t.Mods.Characters[:0:0], named...)
	return nil
}

// characterRows are the rows a target names: the row of that name, else every
// row named the target followed by "_" and a tier number.
func characterRows(c data.Collection, target string) []int {
	var exact, family []int
	for i := 1; i < c.Len(); i++ {
		name := c.EntryName(i)
		switch {
		case name == target:
			exact = append(exact, i)
		case strings.HasPrefix(name, target+"_") && allDigits(name[len(target)+1:]):
			family = append(family, i)
		}
	}
	if len(exact) != 0 {
		return exact
	}
	return family
}

// characterKindRow is the row a kind names for the row `of`: the row of that
// name, else the kind followed by the tier number `of` ends with, else the first
// row named the kind followed by digits.
func characterKindRow(c data.Collection, kind, of string) (int, bool) {
	first, tiered := 0, 0
	tier := ""
	if at := strings.LastIndexByte(of, '_'); at >= 0 && allDigits(of[at+1:]) {
		tier = of[at+1:]
	}
	for i := 1; i < c.Len(); i++ {
		name := c.EntryName(i)
		switch {
		case name == kind:
			return i, true
		case tier != "" && name == kind+tier && tiered == 0:
			tiered = i
		case strings.HasPrefix(name, kind) && allDigits(name[len(kind):]) && len(name) > len(kind) && first == 0:
			first = i
		}
	}
	if tiered != 0 {
		return tiered, true
	}
	return first, first != 0
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// modCharacterName is the display name a mod gives the person built from the
// definition row of that name.
func (in *InstallResources) modCharacterName(row string) (string, bool) {
	if in == nil || in.Table == nil {
		return "", false
	}
	return in.Table.Mods.CharacterName(row)
}
