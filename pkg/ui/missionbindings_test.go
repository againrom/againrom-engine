package ui

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestMissionBindingsReadTheSamePhysicalAndNamedKeys(t *testing.T) {
	for _, profile := range []string{"", "pickup-p"} {
		for _, key := range []struct {
			name string
			key  ebiten.Key
		}{{"p", ebiten.KeyP}, {"f", ebiten.KeyF}} {
			for _, ctrl := range []bool{false, true} {
				var physical, named appInput
				physical.Viewer.Ctrl, named.Viewer.Ctrl = ctrl, ctrl
				physical.readMissionKeys(missionKeyProfiles[profile], onlyKey(key.key), ctrl)
				named.missionKeyNamed(missionKeyProfiles[profile], key.name)
				if !reflect.DeepEqual(physical, named) {
					t.Fatalf("%q %s Ctrl %v: physical %+v, named %+v", profile, key.name, ctrl, physical, named)
				}
				wantPatrol := key.name == "p" && profile == ""
				wantGrab := key.name == "f" && !ctrl || key.name == "p" && profile == "pickup-p" && !ctrl
				wantReply := key.name == "p" && profile == "pickup-p" && !ctrl
				if physical.Patrol != wantPatrol || physical.Grab != wantGrab || physical.GrabReply != wantReply {
					t.Fatalf("%q %s Ctrl %v: patrol/grab/reply %v/%v/%v, want %v/%v/%v", profile, key.name, ctrl,
						physical.Patrol, physical.Grab, physical.GrabReply, wantPatrol, wantGrab, wantReply)
				}
				var held appInput
				held.readMissionKeys(missionKeyProfiles[profile], func(ebiten.Key) bool { return false }, ctrl)
				if held.Patrol || held.Grab || held.GrabReply {
					t.Fatal("a frame without a new edge repeated an action")
				}
			}
		}
	}
}

func TestMissionPickupBindingsDispatchThroughApp(t *testing.T) {
	for _, tc := range []struct {
		name, profile, key string
		selected           selection
		grabs, replies     int
		patrol             bool
	}{
		{"second P", "pickup-p", "p", selection{4, 9}, 2, 1, false},
		{"second P empty", "pickup-p", "p", nil, 0, 0, false},
		{"second P foreign", "pickup-p", "p", selection{7}, 0, 0, false},
		{"second F", "pickup-p", "f", selection{4}, 1, 0, false},
		{"first P", "", "p", selection{4}, 0, 0, true},
		{"first F", "", "f", selection{4}, 1, 0, false},
		{"second Patrol alias", "pickup-p", "patrol", selection{4}, 0, 0, true},
		{"second Alt P", "pickup-p", "alt-p", selection{4}, 0, 0, false},
		{"second Alt F", "pickup-p", "alt-f", selection{4}, 0, 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			a, v := mkOnMap(t)
			a.SetMissionKeys(tc.profile)
			v.sel = tc.selected
			grabs := 0
			a.flow.grab = func(id uint32, x, y int, action GrabAction) {
				grabs++
				if tc.key == "p" {
					if action != GrabCollect || !slices.Contains(tc.selected, id) {
						t.Errorf("collect payload: %d %d %d %v", id, x, y, action)
					}
				} else if id != 0 || x != 0 || y != 0 || action != GrabUnderfoot {
					t.Errorf("key changed the F payload: %d %d %d %v", id, x, y, action)
				}
			}
			var got []gestureReply
			v.SetCommandAcknowledgment(func(g VoiceGesture, ids []uint32, _ time.Time) {
				got = append(got, gestureReply{g, append([]uint32(nil), ids...)})
			})
			if err := a.HeadlessKey(tc.key); err != nil {
				t.Fatal(err)
			}
			if grabs != tc.grabs || len(got) != tc.replies || (v.aimed == commandPatrol) != tc.patrol {
				t.Fatalf("grabs %d replies %v mode %v; want %d/%d patrol %v", grabs, got, v.aimed, tc.grabs, tc.replies, tc.patrol)
			}
			if tc.replies != 0 && !reflect.DeepEqual(got, []gestureReply{{VoicePickup, []uint32(tc.selected)}}) {
				t.Fatalf("reply %v, want one pickup over %v", got, tc.selected)
			}
		})
	}
}

func TestMissionPickupReplyObeysTheExistingInputGates(t *testing.T) {
	for _, gate := range []string{"control", "menu", "no grab", "no reply"} {
		t.Run(gate, func(t *testing.T) {
			a, v := mkOnMap(t)
			a.SetMissionKeys("pickup-p")
			v.sel = selection{4}
			grabs, replies := 0, 0
			a.flow.grab = func(uint32, int, int, GrabAction) { grabs++ }
			v.SetCommandAcknowledgment(func(VoiceGesture, []uint32, time.Time) { replies++ })
			in := a.headlessIdleInput()
			in.missionKeyNamed(a.missionKeys, "p")
			switch gate {
			case "control":
				in.Viewer.Ctrl, in.AttackHeld = true, true
			case "menu":
				v.menuUp = true
			case "no grab":
				a.flow.grab = nil
			case "no reply":
				v.SetCommandAcknowledgment(nil)
			}
			a.step(in, atAt)
			wantGrabs := 0
			if gate == "no reply" {
				wantGrabs = 1
			}
			if grabs != wantGrabs || replies != 0 || v.aimed == commandPatrol {
				t.Fatalf("grabs %d replies %d mode %v, want %d/0/no Patrol", grabs, replies, v.aimed, wantGrabs)
			}
		})
	}
}
