package game

import (
	"reflect"
	"testing"

	"againrom/pkg/data"
)

// TestReleaseFallenHeroBodyIsTheForcedDyingBodyOfItsDirectory resolves every
// hero body of the installed roster directories and requires the fallen body
// to be the forced dying body's composed sheet, which differs from the class
// record's own corpse sheet (the mercenary's).
func TestReleaseFallenHeroBodyIsTheForcedDyingBodyOfItsDirectory(t *testing.T) {
	f := releaseFront(t)
	units, err := LoadUnits(f.Archives.Containers)
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, dir := range []string{data.HeroDirHeroes, data.HeroDirHeroesLight} {
		for _, body := range []data.HeroBody{data.BodySwordsman, data.BodyAxeman, data.BodyClubman, data.BodyArcher, data.BodyMage} {
			LoadHeroBody(f.Archives.Containers, units, dir, body)
			live := units.Bodies[data.HeroBodyKey(dir, body)]
			if live == nil {
				continue
			}
			dying := data.HeroBodyName(data.BodyUnarmed, false, body == data.BodyMage, true)
			LoadHeroBody(f.Archives.Containers, units, dir, dying)
			want := units.Bodies[data.HeroBodyKey(dir, dying)]
			if want == nil || live.Corpse == nil {
				t.Fatalf("%s/%s: missing fallen body", dir, body)
			}
			if !reflect.DeepEqual(live.Corpse.Frames, want.Frames) {
				t.Errorf("%s/%s: fallen frames are not the %s sheet", dir, body, dying)
			}
			id, _ := data.HeroBodyClass(body)
			if own := units.Classes[id].Corpse; own != nil && reflect.DeepEqual(own.Frames, live.Corpse.Frames) {
				t.Errorf("%s/%s: fallen body is still the class record's corpse sheet", dir, body)
			}
			checked++
		}
	}
	if checked == 0 {
		t.Fatal("no installed hero body resolved")
	}
	t.Logf("checked %d hero bodies", checked)
}
