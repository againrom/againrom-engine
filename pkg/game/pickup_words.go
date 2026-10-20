package game

import (
	"reflect"
	"strconv"
	"time"

	"againrom/pkg/data"
	"againrom/pkg/mapload"
	"againrom/pkg/sim"
	"againrom/pkg/ui"
)

// announceLife is how long a pickup, gold or skill-raise line stays once it is
// the oldest on the message line: 3000 ms, in the white ramp
// (MISSION-MSGPOST-058).
const announceLife = 3 * time.Second

// pickupWord is an install word, or the authored one when the install states
// none.
func pickupWord(installed, authored string) string {
	if installed == "" {
		return authored
	}
	return installed
}

// namedStack is a carried Item a line states and the name the selection card
// shows for the character whose pack holds it; "" names nobody.
type namedStack struct {
	stack sim.ItemStack
	name  string
}

// heldStack is a carried Item a line states and the entity whose pack holds
// it, for a line posted before the map driver can name it.
type heldStack struct {
	stack  sim.ItemStack
	holder sim.EntityID
}

// itemLine is lead, a space and the Item's name, then in one pair of
// parentheses string 86, the count and string 87 when the count is above one,
// and the holder's name, after a comma when the count is there
// (ITEM-PICKTEXT-145 for the count form; the holder is DIV-3765).
func itemLine(words *ui.Words, lead, name string, count uint32, holder string) string {
	authored := ui.AuthoredWords()
	line := lead + " " + name
	var inside string
	if count > 1 {
		inside = pickupWord(words.PickedUpNow, authored.PickedUpNow) + " " +
			strconv.FormatUint(uint64(count), 10) + " " +
			pickupWord(words.PickedUpPieces, authored.PickedUpPieces)
	}
	if holder != "" {
		if inside != "" {
			inside += ", "
		}
		inside += holder
	}
	if inside != "" {
		line += " (" + inside + ")"
	}
	return line
}

// pickupItemLine is the line a pickup states for one carried Item: global
// string 85 leads it (ITEM-PICKTEXT-145).
func pickupItemLine(words *ui.Words, name string, count uint32, holder string) string {
	return itemLine(words, pickupWord(words.PickedUp, ui.AuthoredWords().PickedUp), name, count, holder)
}

// grantItemLine is the line an Item a mission script or the cheat gave states:
// the engine's received word leads it (DIV-3766).
func grantItemLine(words *ui.Words, name string, count uint32, holder string) string {
	return itemLine(words, pickupWord(words.Received, ui.AuthoredWords().Received), name, count, holder)
}

// pickupGoldLine is the line a sack's gold states: global string 88, the gain
// and string 89, each after one space (ITEM-PICKTEXT-145), then the taker's
// name in parentheses (DIV-3765).
func pickupGoldLine(words *ui.Words, gain int32, taker string) string {
	line := goldLine(words, pickupWord(words.PickedUpGold, ui.AuthoredWords().PickedUpGold), gain)
	if taker != "" {
		line += " (" + taker + ")"
	}
	return line
}

// grantGoldLine is the line gold a mission script or the cheat gave states:
// the received word, the gain and string 89. It names nobody: the gold goes
// to the shared purse (DIV-3767).
func grantGoldLine(words *ui.Words, gain int32) string {
	return goldLine(words, pickupWord(words.Received, ui.AuthoredWords().Received), gain)
}

func goldLine(words *ui.Words, lead string, gain int32) string {
	return lead + " " + strconv.FormatInt(int64(gain), 10) + " " +
		pickupWord(words.PickedUpGoldUnit, ui.AuthoredWords().PickedUpGoldUnit)
}

// purseGain is the gain the gold line states between two readings of the
// purse: the signed difference when the purse rose, and 0 for a purse that did
// not (ITEM-PICKTEXT-145).
func purseGain(before, after uint32) int32 {
	if int32(before) >= int32(after) {
		return 0
	}
	return int32(after) - int32(before)
}

// pickupLinesForItems is one pickup line per carried Item, in the given order,
// each stating that Item's own count and its holder.
func pickupLinesForItems(items []namedStack, table *mapload.Table, words *ui.Words) []string {
	var lines []string
	for _, item := range items {
		lines = append(lines, pickupItemLine(words, itemName(data.ItemCode(item.stack.Code), table), item.stack.Count, item.name))
	}
	return lines
}

// grantLinesForItems is one received line per carried Item a grant reached.
func grantLinesForItems(items []namedStack, table *mapload.Table, words *ui.Words) []string {
	var lines []string
	for _, item := range items {
		lines = append(lines, grantItemLine(words, itemName(data.ItemCode(item.stack.Code), table), item.stack.Count, item.name))
	}
	return lines
}

// pickupLinesForTake is the gold line, naming the taker, when the purse
// gained, then each reached Item.
func pickupLinesForTake(reached []namedStack, gained int32, taker string, table *mapload.Table, words *ui.Words) []string {
	var lines []string
	if gained > 0 {
		lines = append(lines, pickupGoldLine(words, gained, taker))
	}
	return append(lines, pickupLinesForItems(reached, table, words)...)
}

// named is each stack held by holder, named for it.
func (mw *mapWorld) named(stacks []sim.ItemStack, holder sim.EntityID) []namedStack {
	name := mw.shownName(holder)
	out := make([]namedStack, 0, len(stacks))
	for _, st := range stacks {
		out = append(out, namedStack{stack: st, name: name})
	}
	return out
}

// announcePurseGrant posts the received line for gold a mission script or the
// cheat put in a purse that went from before to after (DIV-3767).
func (mw *mapWorld) announcePurseGrant(before, after uint32) {
	if gain := purseGain(before, after); gain > 0 {
		words := mw.view.Words()
		announce(mw.view, []string{grantGoldLine(&words, gain)})
	}
}

// announceItemGrant posts a received line for each stack of holder's pack a
// mission script or the cheat reached, from before to after, when holder is
// one of this player's characters (DIV-3766).
func (mw *mapWorld) announceItemGrant(holder sim.EntityID, before, after []sim.ItemStack) {
	e, ok := mw.world.Entity(holder)
	if !ok || e.Owner != sim.SelfSlot {
		return
	}
	words := mw.view.Words()
	announce(mw.view, grantLinesForItems(mw.named(reachedStacks(before, after), holder), mw.invParty.table, &words))
}

// observeGrants states each script money grant to the local player and each
// Item a script gave one of the local player's characters.
func (mw *mapWorld) observeGrants(credits []sim.PurseCredit, items []sim.ItemGrant) {
	for _, c := range credits {
		if c.Player == sim.SelfSlot {
			mw.announcePurseGrant(c.Before, c.After)
		}
	}
	for _, g := range items {
		mw.announceItemGrant(g.Holder, g.Before, g.After)
	}
}

// announce posts each line to the map message line in white for announceLife.
func announce(view *ui.Viewer, lines []string) {
	for _, line := range lines {
		view.PostMessage(line, ui.MessageWhite, announceLife)
	}
}

// reachedStacks is each stack of after that before does not hold unchanged,
// in after's order: a stack a take created, or one it merged units into. A
// merge keeps the destination Item with the summed count (ITEM-MERGE-129),
// and the pickup flag the line answers is on that Item (ITEM-GROUNDMOVE-130),
// so its line states the summed count. A stack is the same stack by its saved
// object, or, without one, by every field but its count.
func reachedStacks(before, after []sim.ItemStack) []sim.ItemStack {
	used := make([]bool, len(before))
	var out []sim.ItemStack
	for _, a := range after {
		kept := false
		for k, b := range before {
			if !used[k] && sameCarriedStack(a, b) {
				used[k], kept = true, a.Count <= b.Count
				break
			}
		}
		if !kept {
			out = append(out, a)
		}
	}
	return out
}

func sameCarriedStack(a, b sim.ItemStack) bool {
	if a.ObjectID != 0 || b.ObjectID != 0 {
		return a.ObjectID == b.ObjectID
	}
	a.Count, b.Count = 0, 0
	return reflect.DeepEqual(a, b)
}
