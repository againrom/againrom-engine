package game

import (
	"encoding/binary"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/formats/sav"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
)

func groundSave(t *testing.T, sacks []sav.GroundSack) []byte {
	t.Helper()
	b := savedBody(10, nil) // one empty Player and an empty dead-actor list
	u16 := func(v uint16) { b = binary.LittleEndian.AppendUint16(b, v) }
	u32 := func(v uint32) { b = binary.LittleEndian.AppendUint32(b, v) }
	b = append(b, 1)
	u32(0)
	u32(0)
	u16(0)
	u16(0)
	b = append(b, make([]byte, 4+4374)...)
	u32(uint32(len(sacks)))
	next := uint16(3)
	classes := map[string]uint16{"Player": 1}
	obj := func(class string) {
		if id := classes[class]; id != 0 {
			u16(0x8000 | id)
		} else {
			u16(0xffff)
			u16(1)
			u16(uint16(len(class)))
			b = append(b, class...)
			classes[class] = next
			next++
		}
		next++
	}
	token := func(cell uint16, x, y byte, identity uint32, price int32) {
		u16(cell)
		u16(cell)
		b = append(b, x, y)
		b = append(b, make([]byte, 19)...)
		u32(uint32(price))
		u32(identity)
		u32(0)
	}
	for _, sack := range sacks {
		obj("Sack")
		token(sack.Cell, sack.FineX, sack.FineY, sack.Identity, 0)
		u32(sack.Gold)
		u32(uint32(len(sack.Items)))
		for _, item := range sack.Items {
			obj("Item")
			token(0, 0, 0, 0, item.Price)
			u32(uint32(len(item.Effects)))
			for _, effect := range item.Effects {
				obj("Effect")
				token(0, 0, 0, 0, 0)
				b = append(b, effect.Kind, effect.Mode)
				u32(effect.Operand)
				state := byte(0)
				if len(item.UnsupportedEffectStates) != 0 {
					state = 1
				}
				b = append(b, state)
			}
			u16(item.Code)
			u16(item.Stack)
			b = append(b, item.Kind)
			b = append(b, 0, 0)      // F45, F46
			u16(0)                   // F48
			u16(uint16(item.Weight)) // signed F4A
			b = append(b, 0)         // F47
		}
		u32(999)
		u32(888)
	}
	return savedContainer(savedTrailer(b))
}

func testGroundPopulation() []sav.GroundSack {
	return []sav.GroundSack{{Identity: 10, Cell: 0x0706, FineX: 128, FineY: 128, Gold: 19,
		Items: []sav.Piece{{Code: 0x1234, Stack: 2, Kind: 3, Price: -17, Weight: -9,
			Effects: []sav.ItemEffect{{Kind: 11, Operand: 2}, {Kind: 11, Mode: 1, Operand: 3}}}}},
		{Identity: 20, Cell: 0x0706, FineX: 128, FineY: 128, Gold: 7,
			Items: []sav.Piece{{Code: 0x3456, Stack: 1, Price: 50}}}}
}

func TestOriginalGroundRestoresStackEffectsAndMergesBeforePickup(t *testing.T) {
	saved := groundSave(t, testGroundPopulation())
	src := missionSource{"scenario/10.alm": synth.ALM(synth.ALMOptions{Width: 40, Height: 40})}
	ms, report, err := ResumeOriginalSave(src, saved, nil, mapload.DifficultyNormal, []mapload.PartyMember{{Class: 1}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	w := ms.World
	sacks := w.Sacks()
	if len(sacks) != 1 || sacks[0].Gold != 26 || len(sacks[0].ItemInstances) != 3 || !report.GroundApplied || report.GroundItems != 3 || report.Sacks != 1 {
		t.Fatalf("sacks=%+v report=%+v", sacks, report)
	}
	registry := w.SavedObjects()
	for _, identity := range []uint32{10, 20} {
		if _, ok := liveGroundContainerTail(registry, identity); ok {
			t.Fatalf("Sack %#x has a live container tail despite the same-cell decline", identity)
		}
	}
	items := sacks[0].ItemInstances
	if !items[0].WeightPresent || items[0].Weight != -9 || !items[2].WeightPresent || items[2].Weight != 0 {
		t.Fatalf("ground weights lost: %+v", items)
	}
	if items[0].Price != -17 || items[0].Kind != 3 || !reflect.DeepEqual(items[0].Effects, items[1].Effects) || len(items[0].Effects) != 2 || items[2].Code != 0x3456 {
		t.Fatalf("lost item state: %+v", items)
	}
	id := ms.Start.IDs[0]
	before := w.Purse(sim.SelfSlot)
	if err := w.TakeSack(id, 6, 7); err != nil {
		t.Fatal(err)
	}
	if len(w.Sacks()) != 0 || w.Purse(sim.SelfSlot) != before+26 {
		t.Fatal("pickup failed to remove ground population or credit purse")
	}
	b, err := w.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	var back sim.World
	if err := back.UnmarshalBinary(b); err != nil || back.Hash() != w.Hash() {
		t.Fatalf("native post-pickup roundtrip: %v", err)
	}
}

// TestOriginalGroundRestoresSackDespiteOneUnsupportedItemEffect is STORY
// 1136's own fix: an item's own unsupported Effect record no longer refuses
// the Sack that holds it, or any other item in it. groundSave's own writer
// stamps state 1 on every Effect of an item whose UnsupportedEffectStates is
// non-empty (its only lever for the byte this test needs), so the first
// item's two Effects arrive diverted, and the second item — sharing the same
// Sack — restores exactly as it did before this story.
func TestOriginalGroundRestoresSackDespiteOneUnsupportedItemEffect(t *testing.T) {
	sacks := testGroundPopulation()
	sacks[0].Items[0].UnsupportedEffectStates = []uint8{1}
	saved := groundSave(t, sacks)
	src := missionSource{"scenario/10.alm": synth.ALM(synth.ALMOptions{Width: 40, Height: 40})}
	ms, report, err := ResumeOriginalSave(src, saved, nil, mapload.DifficultyNormal, []mapload.PartyMember{{Class: 1}}, nil)
	if err != nil {
		t.Fatalf("a Sack with one unsupported item Effect was refused: %v", err)
	}
	got := ms.World.Sacks()
	if len(got) != 1 || len(got[0].ItemInstances) != 3 || !report.GroundApplied || report.GroundItems != 3 {
		t.Fatalf("sacks=%+v report=%+v", got, report)
	}
	items := got[0].ItemInstances
	if items[0].Code != 0x1234 || len(items[0].Effects) != 0 {
		t.Fatalf("the unsupported-effect item's own supported state was lost or fabricated: %+v", items[0])
	}
	if items[2].Code != 0x3456 || items[2].Price != 50 {
		t.Fatalf("an unrelated item in the same Sack was affected: %+v", items[2])
	}
	if report.UnsupportedItemEffects != 2 {
		t.Fatalf("UnsupportedItemEffects = %d, want 2 (both of the one item's own diverted Effect references)", report.UnsupportedItemEffects)
	}
}

func TestOriginalGroundRefusesUnsupportedStateBeforeFrontendReset(t *testing.T) {
	for _, test := range []string{"count", "code", "expansion"} {
		t.Run(test, func(t *testing.T) {
			sacks := testGroundPopulation()
			switch test {
			case "count":
				sacks[0].Items[0].Stack = 0
			case "code":
				sacks[0].Items[0].Code = 0
			case "expansion":
				sacks[0].Items[0].Stack = 65535
				sacks[0].Items[0].Effects = make([]sav.ItemEffect, 16)
			}
			f := &FrontEnd{InstallResources: InstallResources{Campaign: resolved(saveCampaign(), nil)}, CampaignSession: CampaignSession{Offered: 77, Carried: saveParty()}}
			if _, _, err := f.RestoreOriginal(groundSave(t, sacks)); err == nil {
				t.Fatal("unsupported ground state was accepted")
			}
			if f.Offered != 77 || len(f.Carried) == 0 {
				t.Fatal("refusal reset the previous game")
			}
		})
	}
}
