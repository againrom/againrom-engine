# 0121 — mission flow: specification

The game lists all 28 campaign maps and can open none of them as a mission: the only door into a
mission is the `-mission <n>` command-line flag, and a row chosen from the list opens the same map
as scenery — no script, no triggers, no ending. And a mission that is running has no rule that ends
it when the player's hero dies; the world's outcome comes from two script counters alone, and more
than half the campaign authors no losing action at all, so on those maps nothing can go wrong for
the player however the fight goes.

This story opens the door and builds the missing loss.

## Scope

In: the map list's rows, the routing of a chosen row, the picker's title, and the rule that a
mission with no living hero is lost. Out: character generation, difficulty selection, the viewport
column span, anything between missions, and the world's serialised byte form — which this story
does not touch.

## The map list

**FR-1 — the list carries campaign missions.** For every campaign map the scenario container holds,
the map list carries one additional row that opens it **as a mission**. Those rows come before
every existing row and are ordered by ascending mission number. Every row that was listed before is
still listed, in its existing order, with its existing text.

**FR-2 — a mission row's number comes from the map's own name.** The number is the row's file-name
stem read as a decimal integer. A campaign entry whose stem is not a positive decimal integer
yields no mission row and is listed only as a map. No list, table or range of mission numbers is
written anywhere in the source: adding a map to the container adds its mission row, and renaming
one moves it.

**FR-3 — choosing a mission row starts that mission.** It opens exactly what the `-mission <n>`
flag opens for the same number: the map's script compiled and running, the party placed at the
start the map decides, the view opened on the party, announcements shown as they are raised, and an
outcome that can be reached and dismissed.

**FR-4 — choosing a map row is unchanged.** It opens the map with no script compiled. No trigger is
evaluated, no announcement can be raised, no outcome can be reached, and no notice can open — for
every row, campaign maps included.

**FR-5 — a mission row is choosable exactly when its map row is.** A campaign entry whose metadata
will not decode is listed both ways and can be chosen neither way, and the mark that says so is
the one the map row already carries.

**FR-6 — the screen says what it now offers.** A mission row states the mission it starts and the
entry it starts it from, and is distinguishable from every map row by its text alone. The picker's
title names both kinds.

## The hero

**FR-7 — a mission with no living hero is lost.** When a mission was started with a hero and that
hero is no longer a living entity of the world — either the world no longer holds it, or it holds
it and it is not alive — the mission's outcome is a loss. A hero that is down but not dead is not
living for this rule.

**FR-8 — it is tested first.** The test runs ahead of both script counters, in the order the
engine's own outcome reporter uses: no living hero, then the losing counter, then the winning one.
A hero who dies on the same step a winning trigger fires loses the mission.

**FR-9 — it is told once, on the existing path.** The loss opens the same outcome notice a
script-authored loss opens, saying the same words, and dismissing it goes to the same destination.
It is announced once and no further step re-opens it.

**FR-10 — a mission with no hero is not lost by this rule.** A mission started with an empty party
never reaches FR-7 at all: there is no hero to have died, and such a mission is decided by its
script counters exactly as it was before.

**FR-11 — a map is not a mission.** A map opened by FR-4 gains no ending of any kind from FR-7: it
has no party, no hero and no outcome, and nothing on that path tests for one.

**FR-12 — the code stops claiming otherwise.** The statement in the map-opening path that a map
opened from the picker "advances under its own script" is false — that path compiles no script —
and is corrected to say what the two paths now are.

## Acceptance

- **AC-1** On a list built from a container holding campaign maps, the rows are: one mission row
  per campaign map in ascending mission number, then the whole previous list unchanged.
- **AC-2** A campaign entry whose stem is not a positive decimal integer produces no mission row.
- **AC-3** A mission row's text names its number and its entry; no map row's text changes.
- **AC-4** Choosing a mission row reaches the same mission door the `-mission` flag reaches, for
  the same number, and hands back the seam an outcome notice is dismissed through.
- **AC-5** Choosing a map row hands back no such seam, and the world it opens runs no script.
- **AC-6** A mission row over an undecodable campaign entry is present and unchoosable.
- **AC-7** With the hero's entity removed from a mission's world, the next settle reports a loss.
- **AC-8** With the hero at zero health but not dead, the next settle reports a loss.
- **AC-9** With the hero alive and the winning counter satisfied, the mission is won — the hero
  rule does not fire on a live hero.
- **AC-10** With the hero dead and a winning outcome already reached in the same world, the
  reported ending is the loss.
- **AC-11** A mission started with an empty party and a dead-or-absent hero id reports nothing.
- **AC-12** A map opened as a map opens no notice and reports no outcome after any number of steps.
- **AC-13** The loss opens the losing banner once, and dismissing it leaves for the menu.

## Properties

- **P-1** Nothing in this story writes to the simulation. Every new read is a read of entities the
  world already holds, and a run that shows notices and a run that does not have the same ticks and
  the same digests.
- **P-2** The world's serialised byte form is unchanged: no field is added, removed or widened, and
  its version does not move.
- **P-3** No mission number, map name or campaign extent is written as a constant. Both the set of
  mission rows and each row's number are functions of what the container holds.
- **P-4** The two doors into a mission stay one door: the row path reaches the mission through the
  same closure the flag reaches it through, so a change to how a mission opens cannot reach one and
  miss the other.

## Constraints

- **C-1** Tests are synthetic and read no game install.
- **C-2** No new package, and no import that moves the dependency graph.
