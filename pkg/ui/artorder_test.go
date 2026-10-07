package ui

import (
	"fmt"
	"image"
	"image/color"
	"reflect"
	"testing"

	"againrom/pkg/render/terrain"
)

func TestCellArtCompositionKeepsActorDepthAgainstTreesAndBuildings(t *testing.T) {
	red, blue := color.RGBA{200, 40, 10, 255}, color.RGBA{20, 80, 240, 255}
	for _, building := range []bool{false, true} {
		for _, tc := range []struct {
			name           string
			actor, scenery image.Point
			category       terrain.UnitCategory
			frontActor     bool
		}{
			{"behind-later-row", image.Pt(4, 3), image.Pt(4, 4), terrain.UnitOrdinary, false},
			{"in-front-of-earlier-row", image.Pt(4, 5), image.Pt(4, 4), terrain.UnitOrdinary, true},
			{"later-column", image.Pt(4, 4), image.Pt(5, 4), terrain.UnitOrdinary, true},
			{"earlier-column", image.Pt(4, 4), image.Pt(3, 4), terrain.UnitOrdinary, false},
			{"air-above-later-row", image.Pt(4, 3), image.Pt(4, 4), terrain.UnitAir, true},
			{"alternate-before-earlier-row", image.Pt(4, 5), image.Pt(4, 4), terrain.UnitAlternate, false},
		} {
			t.Run(fmt.Sprintf("building%v/%s", building, tc.name), func(t *testing.T) {
				art := unitArt(64, 96, 32, 80, 64, 96)
				art.Frames[0].Palette[5] = red
				scenery := &terrain.StaticClass{Width: 64, Height: 96, CenterX: 32, CenterY: 80, Frame: art.Frames[0], Frames: art.Frames}
				v, target := occlusionViewer(t, scenery, tc.scenery)
				p := v.staticPlacements()[0]
				if building {
					// A declared synthetic non-flat structure strip with the same
					// overlap. This exercises the production structure arm.
					v.showStaticArt = false
					v.showStructureArt = true
					class := &terrain.StructureClass{TileWidth: 2, TileHeight: 1, FullHeight: 3}
					v.structuresFlat = []terrain.StructurePlacement{{StructureID: 7, Cell: tc.scenery, TopLeft: p.TopLeft, Frame: p.Frame, Class: class}}
					v.structureInfo = map[uint32]*terrain.StructureClass{7: class}
					v.SetStructures([]MapStructure{{ID: 7, Cell: tc.scenery, Health: 10, MaxHealth: 10}})
					v.planeOrder = terrain.PlaneOrder(v.structuresFlat, v.staticsFlat)
				}
				body := unitArt(64, 64, 32, 48, 64, 64)
				body.Frames[0].Palette[5] = blue
				e := withFrame(tc.actor, body)
				e.ID = 11
				e.DrawCategory = tc.category
				v.SetEntities([]MapEntity{e})
				target.bodies[body.Frames[0]] = "actor"
				v.drawArt(target)
				// Explicit rectangles from the fixture canvases. Compare every
				// shared opaque pixel, not a sorter/helper result.
				actorRect := image.Rect(tc.actor.X*32-16, tc.actor.Y*32-32, tc.actor.X*32+48, tc.actor.Y*32+32)
				sceneryRect := image.Rect(tc.scenery.X*32-16, tc.scenery.Y*32-64, tc.scenery.X*32+48, tc.scenery.Y*32+32)
				shared := actorRect.Intersect(sceneryRect).Intersect(target.pixels.Rect)
				if shared.Empty() {
					t.Fatal("fixture has no opaque overlap")
				}
				want := red
				if tc.frontActor {
					want = blue
				}
				for y := shared.Min.Y; y < shared.Max.Y; y++ {
					for x := shared.Min.X; x < shared.Max.X; x++ {
						if got := target.pixels.RGBAAt(x, y); got != want {
							t.Fatalf("(%d,%d)=%v want foreground=%v; calls=%v", x, y, got, want, target.calls)
						}
					}
				}
				// Inspection must follow the composed opaque body at this pixel.
				// A foreground tree has no card and still hides the actor's card.
				at := shared.Min.Add(image.Pt(shared.Dx()/2, shared.Dy()/2))
				got, found := v.inspectionAt(at.X, at.Y)
				wantSubject, wantFound := InspectionSubject{}, false
				if tc.frontActor {
					wantSubject, wantFound = InspectionSubject{Kind: InspectionUnit, ID: 11}, true
				} else if building {
					wantSubject, wantFound = InspectionSubject{Kind: InspectionStructure, ID: 7}, true
				}
				if found != wantFound || found && got != wantSubject {
					t.Fatalf("inspection at composed pixel %v=(%v,%v), want (%v,%v)", at, got, found, wantSubject, wantFound)
				}
			})
		}
	}
}

func TestCellArtCompositionPreservesNativeCorpseSackLiveTie(t *testing.T) {
	v, target := newSpellPassFixture(t)
	corpse, live := v.entities[0], v.entities[1]
	corpse.Life = LifeDead
	corpse.Translucent, live.Translucent = false, false
	sack := unitArt(32, 32, 16, 16, 32, 32).Frames[0]
	sack.Palette[5] = color.RGBA{30, 180, 40, 255}
	v.sacks = []MapSack{{Cell: image.Pt(4, 4)}}
	v.sackFrames = []*terrain.StaticFrame{sack}
	target.bodies[sack] = "sack"
	v.SetEntities([]MapEntity{live, corpse}) // reverse input is deliberate
	v.drawArt(target)
	want := []string{"shadow body1", "body1", "sack", "shadow body2", "body2"}
	if !reflect.DeepEqual(target.calls, want) {
		t.Fatalf("calls=%v want=%v", target.calls, want)
	}
	if got := target.pixels.RGBAAt(144, 144); got != (color.RGBA{0, 120, 240, 255}) {
		t.Fatalf("live body lost same-cell tie: %v", got)
	}
	v.SetEntities([]MapEntity{corpse})
	target.calls = nil
	v.drawArt(target)
	if got := target.pixels.RGBAAt(144, 144); got != (color.RGBA{30, 180, 40, 255}) {
		t.Fatalf("sack hidden by corpse: %v", got)
	}
}
