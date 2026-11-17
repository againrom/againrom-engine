package game

import (
	"testing"

	"againrom/pkg/mod"
)

// heldCharacters is the open project's characters as the page reads them.
func heldCharacters(s *mapInspectionEdits) (mod.CharacterData, error) {
	d, err := s.heldRows(s.workspace.Project, s.characterNames, mod.CharactersFile)
	return d.Characters, err
}

// heldTexts is the text of each language of the project d.
func heldTexts(s *mapInspectionEdits, d *editorModDocument) (map[string]mod.Strings, error) {
	read, err := s.heldRows(d, nil, textFiles()...)
	return read.Strings, err
}

// A saved mod page's kind files keep their bytes through a reopen and a second
// save with no edit: the page's write rule (the kind's writer, then put or drop)
// is the same at each save.
func TestModPagesKeepTheirKindBytesAcrossAReopen(t *testing.T) {
	t.Run("terrain", func(t *testing.T) {
		s, dir := terrainPage(t)
		setTerrain(t, s, "newground", groundPNG(t, "Glacier.png", squaresPicture(3, 2)))
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		first := tree(t, dir)
		if first[mod.TerrainFile] == "" {
			t.Fatalf("the saved terrain file is empty")
		}
		s = openInW(t, s, dir)
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		if got := tree(t, dir); got[mod.TerrainFile] != first[mod.TerrainFile] {
			t.Fatalf("the terrain file changed on a reopen:\n%q\nwant\n%q", got[mod.TerrainFile], first[mod.TerrainFile])
		}
	})
	t.Run("items", func(t *testing.T) {
		x, s, dir := itemsPage(t)
		setItems(t, s, itemsNewArmour, installValue(t, x, "Chain Mail"))
		setItems(t, s, "item.armour.name.en", "Bone Mail")
		setItems(t, s, "item.armour.name.ru", "Костяная броня")
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		first := tree(t, dir)
		s = openInW(t, s, dir)
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		got := tree(t, dir)
		for _, rel := range []string{mod.ItemsFile, mod.StringsFile("en"), mod.StringsFile("ru")} {
			if got[rel] != first[rel] {
				t.Fatalf("%s changed on a reopen:\n%q\nwant\n%q", rel, got[rel], first[rel])
			}
		}
	})
	t.Run("texts", func(t *testing.T) {
		s, dir := terrainPage(t)
		a := newText(t, s, "A text to show later")
		setTexts(t, s, "en:"+a, "one")
		setTexts(t, s, "ru:"+a, "один")
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		first := tree(t, dir)
		s = openInW(t, s, dir)
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		got := tree(t, dir)
		for _, rel := range []string{mod.StringsFile("en"), mod.StringsFile("ru")} {
			if got[rel] != first[rel] {
				t.Fatalf("%s changed on a reopen:\n%q\nwant\n%q", rel, got[rel], first[rel])
			}
		}
	})
	t.Run("blank strings", func(t *testing.T) {
		// A Russian game reads the name from its Russian text.
		x, s, dir := charactersPage(t, "ru")
		setItems(t, s, itemsNewArmour, installValue(t, x, "Chain Mail"))
		setItems(t, s, "item.armour.name.ru", "Костяная броня")
		setItems(t, s, "item.armour.name.en", "Bone Mail")
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		if tree(t, dir)[mod.StringsFile("en")] == "" {
			t.Fatal("the saved English strings file is empty")
		}
		setItems(t, s, "item.armour.name.en", "")
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		if got, ok := tree(t, dir)[mod.StringsFile("en")]; ok {
			t.Fatalf("an items edit that takes out the last English caption keeps the strings file %q", got)
		}
	})
	t.Run("characters", func(t *testing.T) {
		_, s, dir := charactersPage(t, "en")
		addCharacter(t, s, "new")
		setCharacter(t, s, "character.0.name.en", "Visitor")
		setCharacter(t, s, "character.0.face", "7")
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		first := tree(t, dir)
		if first[mod.CharactersFile] == "" {
			t.Fatalf("the saved characters file is empty")
		}
		s = openInW(t, s, dir)
		if _, err := s.SaveIntoMod(); err != nil {
			t.Fatal(err)
		}
		if got := tree(t, dir); got[mod.CharactersFile] != first[mod.CharactersFile] {
			t.Fatalf("the characters file changed on a reopen:\n%q\nwant\n%q", got[mod.CharactersFile], first[mod.CharactersFile])
		}
	})
}
