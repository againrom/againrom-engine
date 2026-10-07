package ui

import (
	"reflect"
	"slices"
	"testing"
	"time"
)

type gestureReply struct {
	gesture VoiceGesture
	ids     []uint32
}

// TestEachGestureAsksItsVoiceOnce drives each map gesture through the App
// with unit 4 selected: one command reply naming gesture and unit.
func TestEachGestureAsksItsVoiceOnce(t *testing.T) {
	foe, own := 4, 5
	for _, tc := range []struct {
		name string
		do   func(a *App, v *Viewer)
		want VoiceGesture
	}{
		{"move click", func(a *App, v *Viewer) { atPress(a, v, atEmptyCol, atEmptyRow) }, VoiceMove},
		{"attack aimed at ground", func(a *App, v *Viewer) {
			atArm(a, v)
			atPress(a, v, atEmptyCol, atEmptyRow)
		}, VoiceMove},
		{"attack aimed at a unit", func(a *App, v *Viewer) {
			atArm(a, v)
			atPress(a, v, foe-1, 3)
		}, VoiceAttack},
		{"swarm", func(a *App, v *Viewer) {
			x, y := cellPoint(v, atEmptyCol, atEmptyRow)
			in := atFrame(x, y)
			in.March = true
			a.step(in, atAt)
			atPress(a, v, atEmptyCol, atEmptyRow)
		}, VoiceSwarm},
		{"patrol", func(a *App, v *Viewer) {
			x, y := cellPoint(v, atEmptyCol, atEmptyRow)
			in := atFrame(x, y)
			in.Patrol = true
			a.step(in, atAt)
			atPress(a, v, atEmptyCol, atEmptyRow)
		}, VoicePatrol},
		{"defend", func(a *App, v *Viewer) {
			x, y := cellPoint(v, atEmptyCol, atEmptyRow)
			in := atFrame(x, y)
			in.Defend = true
			a.step(in, atAt)
			atPress(a, v, own, 3)
		}, VoiceDefend},
		{"guard key", func(a *App, v *Viewer) { a.step(appInput{Guard: true}, atAt) }, VoiceGuard},
		{"stand ground key", func(a *App, v *Viewer) { a.step(appInput{StandGround: true}, atAt) }, VoiceStandGround},
		{"retreat key", func(a *App, v *Viewer) { a.step(appInput{PlayerRetreat: true}, atAt) }, VoiceRetreat},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v := mkOnMap(t)
			a.flow.order = func(uint32, int, int) {}
			a.flow.attack = func(uint32, uint32, uint32, int, int, bool) {}
			a.flow.march = func(uint32, bool, int, int) {}
			a.flow.stance = func(uint32, bool) {}
			v.defendSink = func(uint32, uint32) {}
			v.playerRetreatSink = func([]uint32) {}
			var got []gestureReply
			v.SetCommandAcknowledgment(func(g VoiceGesture, ids []uint32, _ time.Time) {
				got = append(got, gestureReply{g, slices.Clone(ids)})
			})
			v.sel = selection{4}
			tc.do(a, v)
			if want := []gestureReply{{tc.want, []uint32{4}}}; !reflect.DeepEqual(got, want) {
				t.Fatalf("replies %v, want %v", got, want)
			}
		})
	}
}

// TestACastAsksNoVoice delivers a cast and asks for no reply.
func TestACastAsksNoVoice(t *testing.T) {
	for _, tc := range []struct {
		name     string
		col, row int
	}{{"at ground", atEmptyCol, atEmptyRow}, {"at a unit", 4, 3}} {
		t.Run(tc.name, func(t *testing.T) {
			a, v := mkOnMap(t)
			strikes, replies := 0, 0
			a.flow.attack = func(uint32, uint32, uint32, int, int, bool) { strikes++ }
			v.SetCommandAcknowledgment(func(VoiceGesture, []uint32, time.Time) { replies++ })
			v.sel = selection{4}
			v.selectedSpell, v.spellArmed = 7, true
			v.spellbook = []SpellEntry{{ID: 7, PointTarget: tc.col == atEmptyCol}}
			atPress(a, v, tc.col, tc.row)
			if strikes != 1 || replies != 0 {
				t.Fatalf("%d casts delivered and %d replies asked, want 1 and 0", strikes, replies)
			}
		})
	}
}
