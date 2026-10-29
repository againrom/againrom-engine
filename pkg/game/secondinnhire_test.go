package game

import (
	"image"
	"reflect"
	"testing"

	"againrom/internal/synth"
	"againrom/pkg/ui"
)

// TalkTo kind 1 stores bank[511+ID]==0 into bank 511+ID; kind 2 stores
// nothing (R2-ENGINE-493, R2-SESSION-191).
func TestSecondInnHireToggle(t *testing.T) {
	c := innAt(2, 30, map[int]int32{514: 7})
	c.talkTo(innOffer{kind: 1, npc: 1})
	c.talkTo(innOffer{kind: 1, npc: 3})
	c.talkTo(innOffer{kind: 1, npc: 20, high: true})
	if c.bank[512] != 1 || c.bank[514] != 0 || c.bank[531] != 1 {
		t.Fatalf("toggle 512 %d 514 %d 531 %d", c.bank[512], c.bank[514], c.bank[531])
	}
	before := c.bank
	c.talkTo(innOffer{kind: 2, npc: 1})
	c.talkTo(innOffer{kind: 1, npc: 21})
	c.talkTo(innOffer{kind: 1, npc: 0})
	if c.bank != before {
		t.Fatal("kind 2 or a low ID outside 1..20 changed the bank")
	}
}

// Hire on a kind 1 entry answers npc<ID>accept<stage>, sets the high bit
// and toggles the selection; Fire clears the bit and toggles it back
// (R2-ENGINE-495). Hire on a kind 2 entry answers the reject section only;
// Fire on a hired kind 2 entry clears the page's bit and stores nothing
// (R2-SESSION-191).
func TestSecondInnHireAndFire(t *testing.T) {
	c := innAt(2, 30, map[int]int32{532: 1, 552: 2, 533: 2, 553: 2, 513: 1})
	c.enterInn()
	if rows := c.innRows(); !reflect.DeepEqual(rows, []ui.TownRow{{Text: "HIRE 1", Choosable: true}, {Text: "FIRE 2", Choosable: true},
		{Text: "TALK 22", Choosable: true}, {Text: "TALK 2108", Choosable: true}, {Text: "TALK 2110", Choosable: true}}) {
		t.Fatal("rows", rows)
	}
	if key := c.hireOrFire(1); key != "npc1accept30" || c.bank[512] != 1 || !c.hired(1) {
		t.Fatalf("Hire 1: %q bank %d", key, c.bank[512])
	}
	if key := c.hireOrFire(1); key != "" || c.bank[512] != 0 || c.hired(1) {
		t.Fatalf("Fire 1: %q bank %d", key, c.bank[512])
	}
	if key := c.hireOrFire(2); key != "" || c.bank[513] != 1 || c.hired(2) {
		t.Fatalf("Fire on kind 2: %q bank %d", key, c.bank[513])
	}
	if key := c.hireOrFire(2); key != "npc2reject30" || c.bank[513] != 1 || c.hired(2) {
		t.Fatalf("Hire on kind 2: %q bank %d", key, c.bank[513])
	}
	c.enterInn()
	if !c.hired(2) {
		t.Fatal("a new visit did not copy the bank's high bit")
	}
	if key := c.hireOrFire(9); key != "" || c.bank != innAt(2, 30, map[int]int32{532: 1, 552: 2, 533: 2, 553: 2, 513: 1}).bank {
		t.Fatal("an absent seat acted", key)
	}
}

// Every inn action takes the first copied option with the seat's low ID;
// at stage 50 with 533 and 553 set, NPC 2's topic 51 precedes hire 2, so
// Hire on hire 2 shows npc2reject50 and stores nothing (R2-ENGINE-498).
func TestSecondInnHireTakesFirstWord(t *testing.T) {
	c := innAt(2, 50, map[int]int32{533: 1, 553: 2})
	seats, hires := c.innSeats()
	if hires != 1 || seats[0] != (innOffer{kind: 1, npc: 2}) || seats[1].npc != 23 {
		t.Fatalf("seats %v hires %d", seats, hires)
	}
	before := c.bank
	if key := c.hireOrFire(2); key != "npc2reject50" || c.bank != before || c.hired(2) {
		t.Fatalf("Hire on hire 2: %q", key)
	}
	// Loss control: without topic 51 (947 set) the same seat accepts.
	c.bank[947] = 1
	if key := c.hireOrFire(2); key != "npc2accept50" || c.bank[513] != 1 {
		t.Fatalf("Hire without topic 51: %q", key)
	}
}

// A seat click on a hire actor or a talk actor selects the seat
// (R2-SESSION-213, DIV-3110); the hire button runs Hire or Fire on a hire
// actor and does nothing on a talk actor (R2-ENGINE-496, R2-ENGINE-497).
func TestSecondInnHireRouting(t *testing.T) {
	text := synth.File{Path: "text/town.txt", Data: []byte("#npc1accept30\r\n<NPC=1,PART=1>\r\nAccepted.\r\n#other\r\n")}
	f, app, screen := secondCampaignFixtureFiles(t, false, text)
	c := innAt(2, 30, map[int]int32{532: 1, 552: 2})
	c.room = secondTownInn
	f.Town.second = c
	app.SetTown(screen)
	d, ok := screen.composedTown()
	if !ok {
		t.Fatal("no composed town 2")
	}
	view := townRoomView{t: screen, d: d, room: "inn"}
	screen.enterSeats()
	if ctl, ok := view.ControlAt(image.Pt(200, 448)); !ok || ctl.Kind != ui.TownSquareControlSelect || ctl.Door != 0 {
		t.Fatalf("hire seat click %+v %v", ctl, ok)
	}
	if ctl, ok := view.ControlAt(image.Pt(248, 448)); !ok || ctl.Kind != ui.TownSquareControlSelect || ctl.Door != 1 {
		t.Fatalf("talk seat click %+v %v", ctl, ok)
	}
	if ctl, ok := view.ControlAt(image.Pt(554, 67)); !ok || ctl.Door != 0 {
		t.Fatalf("hire button on a hire seat %+v %v", ctl, ok)
	}
	if ctl, ok := view.ControlAt(image.Pt(554, 114)); ok {
		t.Fatalf("talk button on a hire seat %+v", ctl)
	}
	if got := view.hireCaption(); got == nil || *got != secondInnHireCaption {
		t.Fatal("caption on an unhired seat", got)
	}
	screen.Choose(0)
	if body, ok := screen.dialogueBody(); !ok || body == "" || c.bank[512] != 1 {
		t.Fatal("Hire through the row", body, c.bank[512])
	}
	if got := view.hireCaption(); got == nil || *got != secondInnFireCaption {
		t.Fatal("caption on a hired seat", got)
	}
	screen.AdvanceTownDialogue()
	screen.Choose(0)
	if _, ok := screen.dialogueBody(); ok || c.bank[512] != 0 {
		t.Fatal("Fire through the row", c.bank[512])
	}
	screen.selectSeat(screen.scene(d, "inn"), d.Room("inn").Seats, 1)
	if _, ok := view.ControlAt(image.Pt(554, 67)); ok {
		t.Fatal("hire button acts on a talk seat")
	}
	if got := view.hireCaption(); got != nil {
		t.Fatal("caption on a talk seat", *got)
	}
	if ctl, ok := view.ControlAt(image.Pt(554, 114)); !ok || ctl.Door != 1 {
		t.Fatalf("talk button on a talk seat %+v %v", ctl, ok)
	}
	if ctl, ok := view.ControlAt(image.Pt(554, 161)); !ok || ctl.Door != 4 {
		t.Fatalf("exit %+v %v", ctl, ok)
	}
}
