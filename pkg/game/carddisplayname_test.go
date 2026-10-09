package game

import (
	"reflect"
	"testing"

	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

func cardCaptionFixtureSubject(t *testing.T, f *FrontEnd, app *ui.App, id sim.EntityID) ui.PanelSubject {
	t.Helper()
	for _, draw := range f.live.entityDraws() {
		if draw.ID != uint32(id) {
			continue
		}
		f.live.view.SetEntities([]ui.MapEntity{draw})
		inspectionCentre(f.live, draw.Cell.X, draw.Cell.Y)
		if err := app.HeadlessSelectEntity(uint32(id)); err != nil {
			t.Fatal(err)
		}
		if subject, ok := f.live.view.InspectionPanel(); ok {
			return subject
		}
		t.Fatal("source fixture produced no selected card", id)
	}
	t.Fatal("source fixture has no projected entity", id)
	return ui.PanelSubject{}
}

func TestSourceDefinitionNamesReachTheCardAsInstalledCaptions(t *testing.T) {
	f := actorRegistryFront(t)
	f.SetDeterministicFrames(true)
	f.Words.UnitNames[35] = "Localized creature"
	f.Words.UnitNames[3] = "Localized person"
	raw := actorRegistrySave(f.Table.Units.EntryName(1), f.Table.Humans.EntryName(1))
	mission, town, err := f.RestoreOriginal(raw)
	if err != nil || town {
		t.Fatalf("original LOAD: town=%t err=%v", town, err)
	}
	app := f.App("source card captions")
	app.Layout(1024, 768)
	if err := app.OpenMission(mission); err != nil {
		t.Fatal(err)
	}
	if err := app.HeadlessKey("0"); err != nil {
		t.Fatal(err)
	}
	for i := range f.live.fog.visible {
		f.live.fog.visible[i], f.live.fog.explored[i] = 1, 1
	}
	f.live.push()
	manifest := cloneActorManifest(f.live.mission.state.ActorManifest)
	hash := f.live.world.Hash()
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.Class == 0 {
			continue
		}
		want := f.live.actorNames[e.ID]
		if e.SourceBinding.ActorClass() == 1 {
			if want != f.Table.Units.EntryName(1) {
				t.Fatal("source Unit has no internal definition name", e.ID)
			}
			want = f.Words.UnitNames[e.TypeID]
		} else if e.TypeID == 3 {
			if want != f.Table.Humans.EntryName(1) {
				t.Fatal("source Human has no internal definition name", e.ID)
			}
			want = f.Words.UnitNames[e.TypeID]
		}
		subject := cardCaptionFixtureSubject(t, f, app, e.ID)
		got, present := ui.PanelSubjectName(subject)
		if !present || got != want {
			t.Errorf("source class %d type %d card=%q present=%t, want %q", e.SourceBinding.ActorClass(), e.TypeID, got, present, want)
		}
	}
	if f.live.world.Hash() != hash || !reflect.DeepEqual(f.live.mission.state.ActorManifest, manifest) {
		t.Fatal("card projection changed World or ordinary saved Name")
	}
	var unit sim.EntityID
	for _, e := range f.live.world.Entities() {
		if e.SourceBinding.ActorClass() == 1 {
			unit = e.ID
			break
		}
	}
	custom := f.live.chars[unit]
	original := custom
	custom.Name = "Explicit creature name"
	f.live.chars[unit] = custom
	subject := cardCaptionFixtureSubject(t, f, app, unit)
	if got, present := ui.PanelSubjectName(subject); !present || got != custom.Name {
		t.Fatalf("explicit creature name=%q present=%t, want %q", got, present, custom.Name)
	}
	f.live.chars[unit] = original
	actor, found := f.live.entity(unit)
	if !found || f.live.actorNames[unit] != f.Table.Units.EntryName(1) {
		t.Fatal("missing-caption control lost its internal definition key")
	}
	words := f.live.view.Words()
	words.UnitNames[actor.TypeID] = ""
	f.live.view.SetWords(words)
	subject = cardCaptionFixtureSubject(t, f, app, unit)
	if got, present := ui.PanelSubjectName(subject); present || got != "" {
		t.Fatalf("missing installed caption exposes name=%q present=%t", got, present)
	}
	if f.live.world.Hash() != hash || !reflect.DeepEqual(f.live.mission.state.ActorManifest, manifest) {
		t.Fatal("missing-caption projection changed World or ordinary saved Name")
	}
}

func TestSourceRowZeroDefinitionNamesReachTheCardAsInstalledCaptions(t *testing.T) {
	for _, tc := range []struct {
		name, key string
		face      int32
	}{
		{"type and face", "Row zero creature", 3},
		{"raised ghost", "Ghost", 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := actorRegistryFront(t)
			f.SetDeterministicFrames(true)
			parameters := make([]int32, 41)
			for i := range parameters {
				parameters[i] = -1
			}
			parameters[29], parameters[30] = 35, tc.face
			f.Table.Units = dbCollection{{}, {name: tc.key, params: parameters}}
			const caption = "Localized row-zero creature"
			f.Words.UnitNames[35] = caption
			body, actors := actorRegistryBody1111(tc.key)
			body[actors[0].off+16] = 0
			mission, town, err := f.RestoreOriginal(savedContainer(body))
			if err != nil || town {
				t.Fatalf("row-zero original LOAD: town=%t err=%v", town, err)
			}
			app := f.App("row-zero card caption")
			app.Layout(1024, 768)
			if err := app.OpenMission(mission); err != nil {
				t.Fatal(err)
			}
			if err := app.HeadlessKey("0"); err != nil {
				t.Fatal(err)
			}
			var actor sim.Entity
			for _, e := range f.live.world.Entities() {
				if e.SourceBinding.ActorClass() == 1 {
					actor = e
					break
				}
			}
			if actor.SourceBinding.Class == 0 || actor.SourceBinding.TokenRow != 0 || actor.TypeID != 35 || actor.SourceBinding.Face != 3 || f.live.actorNames[actor.ID] != tc.key {
				t.Fatal("row-zero actor lost its literal source name or binding", actor)
			}
			manifest := cloneActorManifest(f.live.mission.state.ActorManifest)
			character := f.live.chars[actor.ID]
			hash := f.live.world.Hash()
			subject := cardCaptionFixtureSubject(t, f, app, actor.ID)
			if got, present := ui.PanelSubjectName(subject); !present || got != caption {
				t.Fatalf("row-zero source card=%q present=%t, want %q", got, present, caption)
			}
			if f.live.actorNames[actor.ID] != tc.key || f.live.chars[actor.ID] != character || !reflect.DeepEqual(f.live.mission.state.ActorManifest, manifest) || f.live.world.Hash() != hash {
				t.Fatal("row-zero card projection changed ordinary source metadata or World")
			}
		})
	}
}
