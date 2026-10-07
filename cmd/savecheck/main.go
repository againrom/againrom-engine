// Command savecheck exercises 0143's save/load loop against a REAL install, in
// two separate process runs, and prints what it saw.
//
// It exists because `go test ./...` is green with no install by golden rule 2:
// every synthetic test in this tree writes a hand-built world into a hand-built
// envelope, and none of them can answer whether a save taken in a shipped
// mission comes back as that mission. The evidence ships and the assets never
// do — this tool is where the evidence is produced (golden rule 2's own
// sentence about a separate developer tool under cmd/).
//
//	savecheck -assets <root> -mission 10 -ticks 200 -saves <dir> save
//	savecheck -assets <root> -saves <dir> load
//
// The two runs are two PROCESSES on purpose: `save` writes a file and exits,
// `load` starts from nothing, reads the directory, restores and reports. A
// version of this that did both in one process would keep the front end, the
// campaign and the world in memory across the trip and could not tell a save
// that works from a save that is never actually read.
//
// It opens no window: the restore's own MapOpener is called and the world
// behind it read directly, which is exactly what the map screen would adopt.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"againrom/pkg/game"
	"againrom/pkg/sim"
)

func main() { os.Exit(run(os.Args[1:], os.Getenv, os.Stdout, os.Stderr)) }

func run(args []string, getenv func(string) string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("savecheck", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	assets := fs.String("assets", "", "path to the game asset root (overrides AGAINROM_ASSETS)")
	saves := fs.String("saves", "", "directory for saved games (required; never inside an install)")
	mission := fs.Int("mission", 10, "which campaign mission to start")
	ticks := fs.Int("ticks", 100, "how many world ticks to run before saving")
	town := fs.Bool("town", false, "take a TOWN save instead of a mission save: arrive, win what the campaign pays for, and write the campaign half alone")
	orig := fs.String("orig", "", "ALSO list the ORIGINAL game's saves from this directory, read-only (usually the install root)")
	name := fs.String("name", "", "load this row by name instead of the newest; use the name `load` prints")
	resave := fs.Bool("resave", false, "after loading, write the loaded state through the game SAVE route and print its name")
	rebuild := fs.Int("rebuild", 0, "construct this mission normally from the loaded party instead of resuming the saved world")
	controlMember := fs.Bool("control-member", false, "with -rebuild, append one controlled non-critical member cloned from the decoded hero")
	killMember := fs.String("kill-member", "", "kill this stable party member id through the live driver and report mission outcome")
	healWitness := fs.String("heal-witness", "", "run the full-health, wound-one, stop-at-full idle-heal witness for this stable member id")
	campaignProgress := fs.Bool("campaign-progress", false, "print restored campaign state and drive its live mission completion headlessly")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, "savecheck:", err)
		return 2
	}
	if fs.NArg() != 1 || (fs.Arg(0) != "save" && fs.Arg(0) != "load") {
		fmt.Fprintln(stderr, "savecheck: one command required: save | load")
		return 2
	}
	root := game.ResolveAssetRoot(*assets, getenv("AGAINROM_ASSETS"))
	if root == "" {
		fmt.Fprintln(stderr, "savecheck: no asset root configured")
		return 2
	}
	if *saves == "" {
		fmt.Fprintln(stderr, "savecheck: -saves is required; this tool never guesses a directory")
		return 2
	}
	front, err := game.NewFrontEnd(root)
	if err != nil {
		fmt.Fprintln(stderr, "savecheck:", err)
		return 1
	}
	store := game.SaveStore{Dir: *saves}
	// THE ORIGINAL STORE IS READ-ONLY AND DEFAULTS TO NOTHING. -orig is not
	// defaulted to the asset root: this tool is what a person points at a
	// directory on purpose, and a default would make "did it read the install"
	// unanswerable from the command line that ran.
	original := game.OriginalStore{Dir: *orig}
	if fs.Arg(0) == "save" {
		if *town {
			return doTownSave(front, store, stdout, stderr)
		}
		return doSave(front, store, *mission, *ticks, stdout, stderr)
	}
	return doLoad(front, store, original, *name, *resave, *rebuild, *controlMember, *killMember,
		*healWitness, *campaignProgress, stdout, stderr)
}

// doSave starts a mission through the SAME opener the game's own door uses,
// runs it, marks the town, and writes the save.
func doSave(front *game.FrontEnd, store game.SaveStore, mission, ticks int,
	stdout, stderr io.Writer) int {

	open := front.MissionOpenerWith(mission, front.NextParty())
	_, _, _, _, _, _, _, _, _, _, err := open()
	if err != nil {
		fmt.Fprintln(stderr, "savecheck: open:", err)
		return 1
	}
	w, ok := front.LiveWorld()
	if !ok {
		fmt.Fprintln(stderr, "savecheck: the opener left no live world")
		return 1
	}
	fmt.Fprintf(stdout, "savecheck: mission %d opened at tick %d, %d entities\n",
		mission, w.Tick(), len(w.Entities()))
	// ONE OF THE PARTY IS ORDERED, so the commanded set has something in it for
	// the round trip to carry: it is the residue field whose loss is
	// behavioural rather than cosmetic.
	if ents := w.Entities(); len(ents) > 0 {
		front.LiveOrder(uint32(ents[0].ID), int(ents[0].X)+1, int(ents[0].Y))
	}
	front.LiveAdvance(ticks)
	save, _, _ := front.SaveSeams(store, game.OriginalStore{}, nil)
	name, err := save(true)
	if err != nil {
		fmt.Fprintln(stderr, "savecheck: save:", err)
		return 1
	}
	fmt.Fprintf(stdout, "savecheck: saved %s at tick %d, hash %016x, gold %d\n",
		name, w.Tick(), w.Hash(), front.Town.Gold())
	return 0
}

func doTownSave(front *game.FrontEnd, store game.SaveStore, stdout, stderr io.Writer) int {
	front.Town.Arrive()
	for n := 1; n <= 60; n++ {
		front.Town.Won(n)
	}
	took, _ := front.Town.Take(game.TownShop, 0)
	save, _, _ := front.SaveSeams(store, game.OriginalStore{}, nil)
	name, err := save(false)
	if err != nil {
		fmt.Fprintln(stderr, "savecheck: save:", err)
		return 1
	}
	fmt.Fprintf(stdout, "savecheck: town save %s: gold %d, chapter %d, took shop offer %d, available %v\n",
		name, front.Town.Gold(), front.Town.Chapter(), took, front.Town.Available())
	return 0
}

// doLoad is the second process: it knows nothing but the directory.
func doLoad(front *game.FrontEnd, store game.SaveStore, original game.OriginalStore, want string,
	resave bool, rebuild int, controlMember bool, killMember, healWitness string, campaignProgress bool,
	stdout, stderr io.Writer) int {
	// THE LIST COMES THROUGH THE SEAM and not from store.List, because the seam
	// is what the LOAD GAME window calls: a tool that read the directory itself
	// would pass while the window showed nothing.
	_, listSeam, load := front.SaveSeams(store, original, nil)
	list := listSeam()
	if len(list) == 0 {
		fmt.Fprintln(stderr, "savecheck: no saves in", store.Dir, "or", original.Dir)
		return 1
	}
	// THE NOTE IS PRINTED WITH THE ROW. It is what the load window puts on its
	// message line while that row is highlighted, so a tool that showed the
	// list without it would be checking a different list from the one the
	// player reads.
	for _, e := range list {
		fmt.Fprintf(stdout, "savecheck: on disk %s  %q\n", e.Name, e.Label)
		if e.Note != "" {
			fmt.Fprintf(stdout, "savecheck:   note %s\n", e.Note)
		}
	}
	chosen := list[0].Name
	if want != "" {
		found := false
		for _, e := range list {
			if e.Name == want {
				chosen, found = e.Name, true
				break
			}
		}
		if !found {
			fmt.Fprintf(stderr, "savecheck: %q is not a row in the list above\n", want)
			return 1
		}
	}
	open, town, err := load(chosen)
	if err != nil {
		fmt.Fprintln(stderr, "savecheck: load:", err)
		return 1
	}
	if !town {
		if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
			fmt.Fprintln(stderr, "savecheck: reopen:", err)
			return 1
		}
	}
	fmt.Fprintf(stdout, "savecheck: loaded %s: town=%v gold=%d party=%d\n",
		chosen, town, front.Town.Gold(), len(front.NextParty()))
	// AN OLDER BYTE-FORM VERSION'S DISCLOSURE (1032 B3) IS READ HERE, through
	// the same LiveNotice a window would read to draw it — not a second field
	// this tool keeps for itself.
	if text, _, open := front.LiveNotice(); open {
		// PAGE BY PAGE, WITH THE LINE COUNTS (1032 return 1). The first line is
		// the page on screen, which is what LiveNotice has always returned; the
		// pages under it are the rest of the disclosure, each with the lines its
		// text produces and the lines the window draws. Those two numbers are the
		// whole point: before this they could not be read at all, and half of
		// every long disclosure was undrawn and unreachable with every witness in
		// the tree reporting it complete.
		fmt.Fprintf(stdout, "savecheck: load notice, page on screen: %s\n", text)
		pages := front.LiveNoticePages()
		fmt.Fprintf(stdout, "savecheck: load notice: %d page(s)\n", len(pages))
		for i, p := range pages {
			flag := ""
			if p.Drawn < p.Produced {
				flag = "  CLIPPED"
			}
			fmt.Fprintf(stdout, "savecheck:   page %d/%d, %d line(s) produced, %d drawn%s: %s\n",
				i+1, len(pages), p.Produced, p.Drawn, flag, p.Text)
		}
	}
	if town {
		fmt.Fprintf(stdout, "savecheck: town came back: chapter %d, available %v, done(10)=%v\n",
			front.Town.Chapter(), front.Town.Available(), front.Town.Done(10))
		// A town is a complete original-save authoring witness too. This arm
		// used to return before the -resave flag was observed, which meant the
		// tool advertised an original-to-Againrom round trip but could exercise
		// it only for the world-present shape. Drive the same SaveSeams closure
		// the UI calls, explicitly off-map, so an authored city file is proved
		// through the production SAVE route rather than through savtool.
		if resave {
			save, _, _ := front.SaveSeams(store, game.OriginalStore{}, nil)
			out, err := save(false)
			if err != nil {
				fmt.Fprintln(stderr, "savecheck: resave:", err)
				return 1
			}
			fmt.Fprintf(stdout, "savecheck: RESAVED town as %s through the game SAVE route\n", out)
		}
		return 0
	}
	if rebuild > 0 {
		party := front.LiveParty()
		if controlMember && len(party) > 0 {
			control := party[0]
			control.ID, control.Name = "control:noncritical", ""
			control.StartingHero, control.CompanionNPC, control.Temporary = false, 0, true
			control.Saved = nil
			party = append(party, control)
			fmt.Fprintln(stdout, "savecheck: appended one controlled non-critical party member for the real-map death witness")
		}
		open = front.MissionOpenerWith(rebuild, party)
		if _, _, _, _, _, _, _, _, _, _, err := open(); err != nil {
			fmt.Fprintln(stderr, "savecheck: rebuild:", err)
			return 1
		}
		fmt.Fprintf(stdout, "savecheck: rebuilt mission %d normally from %d decoded party members\n", rebuild, len(party))
	}
	w, ok := front.LiveWorld()
	if !ok {
		fmt.Fprintln(stderr, "savecheck: the loaded opener left no live world")
		return 1
	}
	cells, total := front.LiveExplored()
	pct := 0.0
	if total > 0 {
		pct = 100 * float64(cells) / float64(total)
	}
	fmt.Fprintf(stdout, "savecheck: resumed at tick %d, hash %016x, purse %d, %d entities, %d commanded, "+
		"%d of %d cells explored (%.1f%%)\n",
		w.Tick(), w.Hash(), w.Purse(sim.SelfSlot), len(w.Entities()), front.LiveCommanded(), cells, total, pct)
	reportUpgradeRepairs(stdout, w)
	if campaignProgress {
		before := front.LiveCampaignProgress()
		fmt.Fprintf(stdout, "savecheck: campaign before completion: restored=%v main=%d selected=%d children=%v announced=%v inn=%v shop=%v school=%v documents=%d markers=%v mission-time=%d\n",
			before.Restored, before.Main, before.Selected, before.Children, before.Announced,
			before.InnMission, before.ShopMission, before.TCMission, before.Documents,
			before.Markers, before.MissionTime)
		next, line, err := front.LiveCompleteCampaign()
		if err != nil {
			fmt.Fprintln(stderr, "savecheck: campaign completion:", err)
			return 1
		}
		after := front.LiveCampaignProgress()
		fmt.Fprintf(stdout, "savecheck: campaign completion: next=%d offered=%d line=%q\n", next, front.Offered, line)
		fmt.Fprintf(stdout, "savecheck: campaign after completion: restored=%v main=%d selected=%d children=%v announced=%v inn=%v available=%v\n",
			after.Restored, after.Main, after.Selected, after.Children, after.Announced,
			after.InnMission, front.Town.Available())
	}
	// THE RE-SAVE IS THE SECOND HALF OF A ROUND TRIP AND IS OPTIONAL. It
	// writes the state that was just loaded back out through the game's SAVE
	// route, so a third process can load that file and check what survived the
	// crossing. It is off by default because every other verb here reads and
	// writes nothing.
	if resave {
		save, _, _ := front.SaveSeams(store, game.OriginalStore{}, nil)
		out, err := save(true)
		if err != nil {
			fmt.Fprintln(stderr, "savecheck: resave:", err)
			return 1
		}
		fmt.Fprintf(stdout, "savecheck: RESAVED as %s, %d of %d cells explored\n",
			out, cells, total)
	}
	snapshot := front.LiveHeadlessSnapshot()
	if game.IsOriginal(chosen) {
		// WHAT THE PARTY ACTUALLY ARRIVED WITH, entity by entity. The resume's
		// own counted report goes to stderr from inside the opener; this is the
		// other end of the same claim, read off the world that was built — which
		// is the only place a restored statistic can be checked against the file
		// rather than against the code that read it.
		fmt.Fprintln(stdout, "savecheck: that was the ORIGINAL game's format --",
			game.OriginalSaveNote)
		for _, e := range w.Entities() {
			if e.Owner != sim.SelfSlot {
				continue
			}
			eq, _ := w.Equipped(e.ID)
			items, _ := w.Carried(e.ID)
			stable := "unbound"
			for _, member := range snapshot.Members {
				if member.Entity == uint32(e.ID) {
					stable = member.ID
					break
				}
			}
			fmt.Fprintf(stdout, "savecheck:   party %d stable=%s at (%d,%d) hp %d/%d mana %d/%d "+
				"class %d book=%#x auto=%d attack=%v target=%v worn %v carried %v\n",
				e.ID, stable, e.X, e.Y, e.HP, e.MaxHP, e.Mana, e.MaxMana, e.Class,
				e.KnownSpells, e.AutoSpell, e.HasAttackTarget, e.HasTarget, eq, items)
			if base, spread, ok := w.WeaponSpellDamage(e.ID); ok {
				fmt.Fprintf(stdout, "savecheck:     weapon spell damage %d-%d\n", base, base+spread)
			}
			for _, member := range snapshot.Members {
				if member.Entity != uint32(e.ID) {
					continue
				}
				for _, worn := range member.Worn {
					if worn.Slot == 1 {
						fmt.Fprintf(stdout, "savecheck:     live hand tooltip %q\n", worn.Info)
					}
				}
			}
			if doll, ok := front.LiveDoll(uint32(e.ID)); ok && doll.Equipment[0] != 0 && doll.Equipment[1] != 0 {
				fmt.Fprintf(stdout, "savecheck:     live doll weapon=%#04x shield=%#04x full=%x bare=%x weapon-only=%x shield-only=%x drawn=%v/%v/%v/%v\n",
					doll.Equipment[0], doll.Equipment[1], doll.Full, doll.Bare, doll.Weapon, doll.Shield,
					doll.FullDrawn, doll.BareDrawn, doll.WeaponDrawn, doll.ShieldDrawn)
			}
		}
	}
	if killMember != "" {
		id, ok := snapshotMemberEntity(snapshot, killMember)
		if !ok {
			fmt.Fprintf(stderr, "savecheck: stable member %q is not in the live party\n", killMember)
			return 1
		}
		front.LiveKill(id)
		events := front.LiveAdvanceCasts(64)
		fmt.Fprintf(stdout, "savecheck: killed %s entity %d through live driver: outcome=%v casts=%d\n",
			killMember, id, w.Outcome(), len(events))
	}
	if healWitness != "" {
		if !runHealWitness(front, w, snapshot, healWitness, stdout, stderr) {
			return 1
		}
	}
	return 0
}

func snapshotMemberEntity(state game.HeadlessState, stable string) (uint32, bool) {
	for _, member := range state.Members {
		if member.ID == stable {
			return member.Entity, true
		}
	}
	return 0, false
}

func runHealWitness(front *game.FrontEnd, w *sim.World, state game.HeadlessState, healerStable string,
	stdout, stderr io.Writer) bool {
	healerID, ok := snapshotMemberEntity(state, healerStable)
	if !ok {
		fmt.Fprintf(stderr, "savecheck: heal witness member %q is not in the live party\n", healerStable)
		return false
	}
	party := make(map[sim.EntityID]bool, len(state.Members))
	for _, member := range state.Members {
		party[sim.EntityID(member.Entity)] = true
	}
	entities := w.Entities()
	var healer sim.Entity
	foundHealer := false
	allFull := true
	for _, e := range entities {
		if !party[e.ID] {
			continue
		}
		if e.ID == sim.EntityID(healerID) {
			healer, foundHealer = e, true
		}
		if e.Alive() && e.MaxHP > 0 && e.HP < e.MaxHP {
			allFull = false
		}
	}
	if !foundHealer {
		fmt.Fprintf(stderr, "savecheck: heal witness entity %d is absent\n", healerID)
		return false
	}
	// Mission 40 begins inside an active battle. The owner-directed idle-Heal
	// witness needs an idle mage, so remove only actors currently hostile to the
	// healer through the live kill-command path before arming Heal. The map,
	// party, terrain, spell table and mission driver remain the shipped mission;
	// this controlled state transition is printed explicitly for closure.
	removedHostiles := 0
	relations := w.Relations()
	for _, e := range entities {
		if party[e.ID] || !e.Alive() || !relations.Hostile(healer.Owner, e.Owner) {
			continue
		}
		front.LiveKill(uint32(e.ID))
		removedHostiles++
	}
	if removedHostiles > 0 {
		front.LiveAdvanceCasts(1)
		entities = w.Entities()
		healer = entityByID(entities, sim.EntityID(healerID))
		fmt.Fprintf(stdout, "savecheck: heal witness removed %d hostile actors through live death commands to establish the idle scene\n", removedHostiles)
	}
	if !allFull {
		fmt.Fprintln(stderr, "savecheck: heal witness requires the loaded party to start fully healthy")
		return false
	}
	// Arm the shipped Heal row through the same canonical toggle command the
	// spellbook uses. This keeps the real-save witness independent of whatever
	// UI selection the original format did not serialize.
	front.LiveAutocast(healerID, 6)
	front.LiveAdvanceCasts(1)
	if rule, ok := w.Spell(6); ok {
		fmt.Fprintf(stdout, "savecheck: live world heal row cost=%d range=%d restorative=%v target=%v\n",
			rule.ManaCost, rule.MaxRange, rule.Restorative, rule.TargetsUnit)
	} else {
		fmt.Fprintln(stdout, "savecheck: live world has no spell row 6")
	}
	beforeMana := healer.Mana
	healthyEvents := front.LiveAdvanceCasts(64)
	healthyCasts := castsBy(healthyEvents, sim.EntityID(healerID))

	entities = w.Entities()
	foundHealer = false
	for _, e := range entities {
		if e.ID == sim.EntityID(healerID) {
			healer, foundHealer = e, true
			break
		}
	}
	if !foundHealer {
		fmt.Fprintln(stderr, "savecheck: healer disappeared during healthy window")
		return false
	}
	var target sim.Entity
	foundTarget := false
	bestDistance := int32(0)
	for _, e := range entities {
		if !party[e.ID] || e.ID == healer.ID || !e.Alive() || e.MaxHP <= 0 {
			continue
		}
		dx, dy := e.X-healer.X, e.Y-healer.Y
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		d := dx
		if dy > d {
			d = dy
		}
		if !foundTarget || d < bestDistance || d == bestDistance && e.ID < target.ID {
			target, bestDistance, foundTarget = e, d, true
		}
	}
	if !foundTarget {
		fmt.Fprintln(stderr, "savecheck: heal witness found no other living party member")
		return false
	}
	moveCasts := 0
	if bestDistance > 1 {
		// The witness changes only position, through the ordinary live move-order
		// seam, so the real mission supplies an in-range target without authored
		// archive bytes or a direct world mutation.
		front.LiveOrder(uint32(healer.ID), int(target.X)+1, int(target.Y))
		for range 256 {
			moveCasts += castsBy(front.LiveAdvanceCasts(1), healer.ID)
			healer = entityByID(w.Entities(), healer.ID)
			target = entityByID(w.Entities(), target.ID)
			dx, dy := target.X-healer.X, target.Y-healer.Y
			if dx < 0 {
				dx = -dx
			}
			if dy < 0 {
				dy = -dy
			}
			bestDistance = dx
			if dy > bestDistance {
				bestDistance = dy
			}
			if bestDistance <= 1 {
				break
			}
		}
		// Replace the positioning order with an ordinary move-to-current-cell;
		// the command machinery clears its own empty route before eligibility is
		// measured, without a direct mutation of canonical order state.
		front.LiveOrder(uint32(healer.ID), int(healer.X), int(healer.Y))
		moveCasts += castsBy(front.LiveAdvanceCasts(1), healer.ID)
		healer = entityByID(w.Entities(), healer.ID)
	}
	front.LiveDamage(uint32(target.ID), 8)
	front.LiveAdvanceCasts(1)
	wounded := entityByID(w.Entities(), target.ID)
	healerBefore := entityByID(w.Entities(), healer.ID)
	startFacing := healerBefore.Facing
	startXP := healerBefore.SkillXP
	admission := w.BookSpellRefusal(healer.ID, target.ID, 6)
	hostilePair := w.Relations().Hostile(healerBefore.Owner, wounded.Owner)
	attackers := 0
	for _, e := range w.Entities() {
		if e.HasAttackTarget && e.AttackTarget == healer.ID && e.Alive() {
			attackers++
		}
	}
	healCasts := 0
	admitted := false
	castTicks := 0
	admittedFacing := startFacing
	var afterHeal sim.Entity
	var afterMana int32
	var healEvent sim.CastEvent
	for range 96 {
		events := front.LiveAdvanceCasts(1)
		for _, event := range events {
			if event.Caster == healer.ID {
				healCasts++
				healEvent = event
			}
		}
		if _, _, ok := w.CastingSpell(healer.ID); ok {
			castTicks++
			if !admitted {
				admittedFacing = entityByID(w.Entities(), healer.ID).Facing
			}
			admitted = true
		}
		if healCasts != 0 {
			afterHeal = entityByID(w.Entities(), target.ID)
			afterMana = entityByID(w.Entities(), healer.ID).Mana
			break
		}
	}
	feedbackBursts, feedbackSprites := front.LiveHealFeedback()
	healerAfter := entityByID(w.Entities(), healer.ID)
	xpDelta := int32(0)
	if int(healEvent.School) < len(healerAfter.SkillXP) {
		xpDelta = healerAfter.SkillXP[healEvent.School] - startXP[healEvent.School]
	}
	stopEvents := front.LiveAdvanceCasts(64)
	stopCasts := castsBy(stopEvents, healer.ID)
	afterStop := entityByID(w.Entities(), target.ID)
	stopMana := entityByID(w.Entities(), healer.ID).Mana
	stopBursts, stopSprites := front.LiveHealFeedback()

	fmt.Fprintf(stdout, "savecheck: heal witness %s entity %d: healthy64 casts=%d mana=%d; "+
		"positioning casts=%d target=%d distance=%d; wound hp=%d/%d owners=%d/%d hostile=%v admission=%q healer hp=%d/%d book=%#x auto=%d attack=%v target=%v wait=%d attackers=%d; "+
		"after wound/heal96 admitted=%v cast-ticks=%d hp=%d/%d casts=%d restored=%d mana=%d school=%d xp-delta=%d "+
		"facing=%d->%d release=(%d,%d)->(%d,%d) feedback=%d/%d; "+
		"full64 casts=%d hp=%d/%d mana=%d feedback=%d/%d\n",
		healerStable, healerID, healthyCasts, beforeMana, moveCasts, target.ID, bestDistance,
		wounded.HP, wounded.MaxHP, healerBefore.Owner, wounded.Owner, hostilePair, admission,
		healerBefore.HP, healerBefore.MaxHP, healerBefore.KnownSpells, healerBefore.AutoSpell,
		healerBefore.HasAttackTarget, healerBefore.HasTarget, healerBefore.CastWait, attackers,
		admitted, castTicks, afterHeal.HP, afterHeal.MaxHP, healCasts, healEvent.HealthRestored, afterMana,
		healEvent.School, xpDelta, startFacing, admittedFacing,
		healEvent.FromX, healEvent.FromY, healEvent.ToX, healEvent.ToY, feedbackBursts, feedbackSprites,
		stopCasts, afterStop.HP, afterStop.MaxHP, stopMana, stopBursts, stopSprites)
	wantXP := int32(0)
	if rule, ok := w.Spell(6); ok {
		wantXP = int32((int64(rule.ManaCost) + 1) / 2)
	}
	return healthyCasts == 0 && moveCasts == 0 && bestDistance <= 1 && healCasts == 1 && afterHeal.HP == afterHeal.MaxHP &&
		healEvent.HealthRestored > 0 && xpDelta == wantXP && feedbackBursts == 1 && feedbackSprites > 1 &&
		stopCasts == 0 && afterStop.HP == afterStop.MaxHP && afterMana < beforeMana && stopBursts == 0 && stopSprites == 0
}

func castsBy(events []sim.CastEvent, caster sim.EntityID) int {
	n := 0
	for _, event := range events {
		if event.Caster == caster {
			n++
		}
	}
	return n
}

func entityByID(entities []sim.Entity, id sim.EntityID) sim.Entity {
	for _, e := range entities {
		if e.ID == id {
			return e
		}
	}
	return sim.Entity{}
}

func reportUpgradeRepairs(stdout io.Writer, w *sim.World) {
	loads, capacities, mapIDs := 0, 0, 0
	for _, e := range w.Entities() {
		if e.Load > 0 {
			loads++
		}
		if e.Capacity > 0 {
			capacities++
		}
		if e.MapUnitID != 0 {
			mapIDs++
		}
	}
	checks, bound := 0, 0
	if sc := w.Script(); sc != nil {
		for _, c := range sc.Checks() {
			checks++
			if c.HasItem {
				bound++
			}
		}
	}
	fmt.Fprintf(stdout, "savecheck: after upgrade repair: %d item-weight entries, %d entities, "+
		"%d with a non-zero load, %d with a non-zero capacity, %d with a map placement id, "+
		"%d of %d check(s) bind an item, %d spell rule(s), %d structure(s)\n",
		len(w.ItemWeights()), len(w.Entities()), loads, capacities, mapIDs, bound, checks, len(w.Spells()), len(w.Structures()))
}
