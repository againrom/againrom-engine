package mod

import (
	"strings"
	"testing"
)

// layerRecords parses one file of a kind for each mod, in load order.
func companionLayers(t *testing.T, order Precedence, files ...[2]string) CompanionData {
	t.Helper()
	d := CompanionData{Order: order}
	for _, f := range files {
		j, err := ParseCompanions(f[0], CompanionsFile, []byte(f[1]))
		if err != nil {
			t.Fatal(err)
		}
		d.Joins = append(d.Joins, j.Joins...)
	}
	return d
}

const talkOnly = "[[join]]\ncompanion = 22\nchapter = 30\ntalk = 5\n"

// Two mods that change one field of one join condition: with load-after the
// later wins and the override is named; without an order the merge refuses;
// off switches the condition off.
func TestCompanionLayersFollowLoadAfter(t *testing.T) {
	ordered := companionLayers(t, Precedence{"b": {"a"}}, [2]string{"a", goodJoin}, [2]string{"b", talkOnly})
	got, overrides, err := MergeCompanions(ordered)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Joins) != 1 || got.Joins[0].Talk != 5 || got.Joins[0].Key != "mage" || got.Joins[0].Building != BuildingTavern || got.Joins[0].Mod != "b" {
		t.Fatalf("merged %+v", got.Joins)
	}
	if len(overrides) != 1 || overrides[0].String() != `companion 22 of chapter 30 talk: mod "b" (data/companions.toml:4) overrides mod "a"` {
		t.Fatalf("overrides %v", overrides)
	}

	unordered := companionLayers(t, nil, [2]string{"a", goodJoin}, [2]string{"b", talkOnly})
	_, _, err = MergeCompanions(unordered)
	want := `mod "b": data/companions.toml:1: companion 22 of chapter 30 is added by mod "a" and named by mod "b", and neither loads after the other: add load-after to one of them`
	if err == nil || err.Error() != want {
		t.Fatalf("unordered: %v, want %q", err, want)
	}

	off := companionLayers(t, Precedence{"b": {"a"}}, [2]string{"a", goodJoin}, [2]string{"b", "[[join]]\ncompanion = 22\nchapter = 30\noff = true\n"})
	got, _, err = MergeCompanions(off)
	if err != nil || len(got.Joins) != 0 {
		t.Fatalf("off: %v %+v", err, got.Joins)
	}
	_, _, err = MergeCompanions(companionLayers(t, nil, [2]string{"b", "[[join]]\ncompanion = 22\nchapter = 30\noff = true\n"}))
	if err == nil || !strings.Contains(err.Error(), "companion 22 of chapter 30 is switched off, but no layer holds it") {
		t.Fatalf("off with no layer: %v", err)
	}
}

func bodyLayers(t *testing.T, order Precedence, files ...[2]string) BodyData {
	t.Helper()
	d := BodyData{Order: order}
	for _, f := range files {
		b, err := ParseBodies(f[0], BodiesFile, []byte(f[1]))
		if err != nil {
			t.Fatal(err)
		}
		d.Bodies = append(d.Bodies, b...)
	}
	return d
}

// Two mods that supply one body: with load-after the later sheet replaces
// the earlier and the override is named; without an order the merge
// refuses; off switches the body off.
func TestBodySheetLayersFollowLoadAfter(t *testing.T) {
	wider := strings.Replace(goodBody, "[96, 80]", "[100, 80]", 1)
	got, overrides, err := MergeBodySheets(bodyLayers(t, Precedence{"b": {"a"}}, [2]string{"a", goodBody}, [2]string{"b", wider}))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Width != 100 || got[0].Mod != "b" {
		t.Fatalf("merged %+v", got)
	}
	if len(overrides) != 1 || overrides[0].String() != `body clubman2h sheet: mod "b" (data/bodies.toml:1) overrides mod "a"` {
		t.Fatalf("overrides %v", overrides)
	}

	_, _, err = MergeBodySheets(bodyLayers(t, nil, [2]string{"a", goodBody}, [2]string{"b", wider}))
	want := `mod "b": data/bodies.toml:1: body clubman2h is added by mod "a" and named by mod "b", and neither loads after the other: add load-after to one of them`
	if err == nil || err.Error() != want {
		t.Fatalf("unordered: %v, want %q", err, want)
	}

	off := "[[body]]\nname = \"clubman2h\"\noff = true\n"
	got, _, err = MergeBodySheets(bodyLayers(t, Precedence{"b": {"a"}}, [2]string{"a", goodBody}, [2]string{"b", off}))
	if err != nil || len(got) != 0 {
		t.Fatalf("off: %v %+v", err, got)
	}
	if _, err := ParseBodies("b", BodiesFile, []byte("[[body]]\nname = \"clubman2h\"\noff = true\ndirections = 8\n")); err == nil ||
		!strings.Contains(err.Error(), "data/bodies.toml:1: [[body]] \"clubman2h\" switches the body off and sets other keys") {
		t.Fatalf("off with keys: %v", err)
	}
}
