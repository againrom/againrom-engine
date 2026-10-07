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

// pickupItemLine is the line a pickup states for one carried Item: global
// string 85, a space and the Item's name, and when the count is above one
// ` (`, string 86, the count, string 87 and `)`, each part after one space
// (ITEM-PICKTEXT-145).
func pickupItemLine(words *ui.Words, name string, count uint32) string {
	authored := ui.AuthoredWords()
	line := pickupWord(words.PickedUp, authored.PickedUp) + " " + name
	if count > 1 {
		line += " (" + pickupWord(words.PickedUpNow, authored.PickedUpNow) + " " +
			strconv.FormatUint(uint64(count), 10) + " " +
			pickupWord(words.PickedUpPieces, authored.PickedUpPieces) + ")"
	}
	return line
}

// pickupGoldLine is the line a purse gain states: global string 88, the gain
// and string 89, each after one space (ITEM-PICKTEXT-145).
func pickupGoldLine(words *ui.Words, gain int32) string {
	authored := ui.AuthoredWords()
	return pickupWord(words.PickedUpGold, authored.PickedUpGold) + " " +
		strconv.FormatInt(int64(gain), 10) + " " +
		pickupWord(words.PickedUpGoldUnit, authored.PickedUpGoldUnit)
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
// each stating that Item's own count.
func pickupLinesForItems(items []sim.ItemStack, table *mapload.Table, words *ui.Words) []string {
	var lines []string
	for _, item := range items {
		lines = append(lines, pickupItemLine(words, itemName(data.ItemCode(item.Code), table), item.Count))
	}
	return lines
}

// pickupLinesForTake is the gold line when the purse gained, then each reached Item.
func pickupLinesForTake(reached []sim.ItemStack, gained int32, table *mapload.Table, words *ui.Words) []string {
	var lines []string
	if gained > 0 {
		lines = append(lines, pickupGoldLine(words, gained))
	}
	return append(lines, pickupLinesForItems(reached, table, words)...)
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
