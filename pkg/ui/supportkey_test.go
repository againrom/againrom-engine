package ui

import (
	"reflect"
	"slices"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
)

func TestSecondGameLKeySendsTheSupportOrderToTheSelection(t *testing.T) {
	for _, tc := range []struct {
		profile, key string
		want         bool
	}{
		{"pickup-p", "l", true},
		{"pickup-p", "ctrl-l", false},
		{"", "l", false},
		{"", "ctrl-l", false},
	} {
		t.Run(tc.profile+"/"+tc.key, func(t *testing.T) {
			a, v := mkOnMap(t)
			a.SetMissionKeys(tc.profile)
			type stood struct {
				id    uint32
				order StanceOrder
			}
			var orders []stood
			a.flow.stance = func(id uint32, order StanceOrder) { orders = append(orders, stood{id, order}) }
			var got []gestureReply
			v.SetCommandAcknowledgment(func(g VoiceGesture, ids []uint32, _ time.Time) {
				got = append(got, gestureReply{g, slices.Clone(ids)})
			})
			v.sel = selection{4}
			if err := a.HeadlessKey(tc.key); err != nil {
				t.Fatal(err)
			}
			var wantOrders []stood
			var wantReplies []gestureReply
			if tc.want {
				wantOrders, wantReplies = []stood{{4, StanceSupport}}, []gestureReply{{VoiceDefend, []uint32{4}}}
			}
			if !reflect.DeepEqual(orders, wantOrders) || !reflect.DeepEqual(got, wantReplies) {
				t.Fatalf("orders %v replies %v, want %v %v", orders, got, wantOrders, wantReplies)
			}
		})
	}
}

func TestOnlyTheSecondGameBindsL(t *testing.T) {
	for _, tc := range []struct {
		profile string
		ctrl    bool
		want    bool
	}{
		{"pickup-p", false, true},
		{"pickup-p", true, false},
		{"", false, false},
		{"", true, false},
	} {
		var in appInput
		in.readMissionKeys(missionKeyProfiles[tc.profile], onlyKey(ebiten.KeyL), tc.ctrl)
		if in.Support != tc.want {
			t.Errorf("%q Ctrl=%v: Support %v, want %v", tc.profile, tc.ctrl, in.Support, tc.want)
		}
	}
}
